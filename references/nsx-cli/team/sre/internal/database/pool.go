package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/go-sql-driver/mysql"

	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/team/sre/srecfg"
)

const (
	// Fixed user for SRE incident responder
	SREUser = "sre_inc_responder"

	// AWS RDS CA certificate bundle URL
	RDSCACertBundleURL = "https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem"
)

type DBInstanceInfo struct {
	Endpoint string
	Port     int32
	Engine   string
}

// PooledConnection represents a connection in the pool with token expiry tracking
type PooledConnection struct {
	db          *sql.DB
	rdsClient   *rds.Client
	config      srecfg.DatabaseConnection
	engine      string
	tokenExpiry time.Time
	lastUsed    time.Time
	mu          sync.RWMutex
}

// ConnectionPool manages a pool of database connections
type ConnectionPool struct {
	connections map[string]*PooledConnection
	mu          sync.RWMutex
}

// Global connection pool instance
var (
	globalPool *ConnectionPool
	poolOnce   sync.Once
)

// GetPool returns the global connection pool instance
func GetPool() *ConnectionPool {
	poolOnce.Do(func() {
		globalPool = &ConnectionPool{
			connections: make(map[string]*PooledConnection),
		}
	})
	return globalPool
}

// GetConnection gets a connection from the pool or creates a new one
func (p *ConnectionPool) GetConnection(
	ctx context.Context,
	dbName string,
	dbConfig srecfg.DatabaseConnection,
) (*PooledConnection, error) {
	p.mu.RLock()
	conn, exists := p.connections[dbName]
	p.mu.RUnlock()

	if exists {
		conn.mu.Lock()
		defer conn.mu.Unlock()

		// Check if token is still valid (with 1 minute buffer)
		if time.Now().Before(conn.tokenExpiry.Add(-1 * time.Minute)) {
			// Check if connection is still alive
			if pingErr := conn.db.PingContext(ctx); pingErr == nil {
				conn.lastUsed = time.Now()
				interact.Debug("Reusing existing connection for database: %s", dbName)
				return conn, nil
			} else {
				interact.Debug("Connection ping failed for database %s: %v", dbName, pingErr)
			}
		} else {
			interact.Debug("Token expired for database %s, creating new connection", dbName)
		}

		// Close the old connection if it exists
		if conn.db != nil {
			_ = conn.db.Close()
		}
	}

	// Create new connection
	interact.Debug("Creating new connection for database: %s", dbName)
	newConn, err := p.createConnection(ctx, dbConfig)
	if err != nil {
		return nil, err
	}

	// Store in pool
	p.mu.Lock()
	p.connections[dbName] = newConn
	p.mu.Unlock()

	return newConn, nil
}

// createConnection creates a new pooled connection
func (p *ConnectionPool) createConnection(ctx context.Context, dbConfig srecfg.DatabaseConnection) (*PooledConnection, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(dbConfig.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create RDS client
	rdsClient := rds.NewFromConfig(cfg)

	// Get DB instance information
	dbInfo, err := getDBInstanceInfo(ctx, rdsClient, dbConfig.DBInstanceIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get DB instance info: %w", err)
	}

	// Generate auth token
	token, err := generateAuthToken(ctx, dbInfo.Endpoint, dbInfo.Port, SREUser, dbConfig.Region, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth token: %w", err)
	}

	// Build DSN based on engine type
	var dsn string
	var driverName string

	switch dbInfo.Engine {
	case "mysql", "aurora-mysql":
		dsn, err = buildMySQLDSN(dbInfo.Endpoint, dbInfo.Port, SREUser, token)
		if err != nil {
			return nil, fmt.Errorf("failed to build MySQL DSN: %w", err)
		}
		driverName = "mysql"
	case "postgres", "aurora-postgresql":
		dsn = buildPostgreSQLDSN(dbInfo.Endpoint, dbInfo.Port, SREUser, token, dbConfig.DBName)
		driverName = "postgres"
	default:
		return nil, fmt.Errorf("unsupported database engine: %s", dbInfo.Engine)
	}

	// Open database connection
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(10 * time.Minute) // Shorter than token expiry

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PooledConnection{
		db:          db,
		rdsClient:   rdsClient,
		config:      dbConfig,
		engine:      dbInfo.Engine,
		tokenExpiry: time.Now().Add(15 * time.Minute), // RDS tokens expire in 15 minutes
		lastUsed:    time.Now(),
	}, nil
}

// DB returns the underlying sql.DB connection
func (pc *PooledConnection) DB() *sql.DB {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.db
}

// Engine returns the database engine type
func (pc *PooledConnection) Engine() string {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.engine
}

// CloseAll closes all connections in the pool
func (p *ConnectionPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for dbName, conn := range p.connections {
		if conn.db != nil {
			_ = conn.db.Close()
			interact.Debug("Closed connection for database: %s", dbName)
		}
	}
	p.connections = make(map[string]*PooledConnection)
}

// CleanupExpired removes expired connections from the pool
func (p *ConnectionPool) CleanupExpired() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for dbName, conn := range p.connections {
		conn.mu.RLock()
		expired := now.After(conn.tokenExpiry)
		lastUsed := conn.lastUsed
		conn.mu.RUnlock()

		// Close connections that are expired or haven't been used in 30 minutes
		if expired || now.Sub(lastUsed) > 30*time.Minute {
			if conn.db != nil {
				_ = conn.db.Close()
			}
			delete(p.connections, dbName)
			interact.Debug("Cleaned up expired connection for database: %s", dbName)
		}
	}
}

// Helper functions moved from connection.go
func getDBInstanceInfo(ctx context.Context, rdsClient *rds.Client, instanceIdentifier string) (*DBInstanceInfo, error) {
	// Implementation from the original connection.go
	input := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &instanceIdentifier,
	}

	result, err := rdsClient.DescribeDBInstances(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe DB instances: %w", err)
	}

	if len(result.DBInstances) == 0 {
		return nil, fmt.Errorf("no DB instances found for identifier: %s", instanceIdentifier)
	}

	instance := result.DBInstances[0]
	return &DBInstanceInfo{
		Endpoint: *instance.Endpoint.Address,
		Port:     *instance.Endpoint.Port,
		Engine:   *instance.Engine,
	}, nil
}

func generateAuthToken(
	ctx context.Context,
	endpoint string,
	port int32,
	username, region string,
	awsConfig aws.Config,
) (string, error) {
	// Build authentication token using the provided AWS config's credentials provider
	token, err := auth.BuildAuthToken(ctx, fmt.Sprintf("%s:%d", endpoint, port), region, username, awsConfig.Credentials)
	if err != nil {
		return "", fmt.Errorf("failed to build auth token: %w", err)
	}

	return token, nil
}

func buildMySQLDSN(endpoint string, port int32, username, token string) (string, error) {
	// Configure TLS for RDS with proper certificate validation
	tlsConfig, err := loadRDSTLSConfig(context.Background(), endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to load RDS TLS config: %w", err)
	}

	err = mysql.RegisterTLSConfig("rds", tlsConfig)
	if err != nil {
		return "", fmt.Errorf("failed to register TLS config: %w", err)
	}

	// Build DSN with IAM token using custom TLS config
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?tls=rds&allowCleartextPasswords=true",
		username,
		token,
		endpoint,
		port,
	)
	return dsn, nil
}

func buildPostgreSQLDSN(endpoint string, port int32, username, token, dbname string) string {
	// For PostgreSQL, default to 'postgres' database if no database specified
	if dbname == "" {
		dbname = "postgres"
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		endpoint, port, username, token, dbname)

	return dsn
}

// loadRDSTLSConfig downloads and configures the RDS CA certificate bundle for secure TLS connections
func loadRDSTLSConfig(ctx context.Context, serverName string) (*tls.Config, error) {
	interact.Debug("Loading RDS CA certificate bundle for server: %s", serverName)

	// Download the RDS CA certificate bundle
	req, err := http.NewRequestWithContext(ctx, "GET", RDSCACertBundleURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for RDS CA bundle: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download RDS CA bundle: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download RDS CA bundle: HTTP %d", resp.StatusCode)
	}

	// Read the certificate bundle
	certPEM, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read RDS CA bundle: %w", err)
	}

	// Create a certificate pool and add the RDS CA certificates
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(certPEM) {
		return nil, fmt.Errorf("failed to parse RDS CA certificates")
	}

	// Configure TLS with proper certificate validation
	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		ServerName: serverName,
		MinVersion: tls.VersionTLS12, // Enforce minimum TLS 1.2
	}

	interact.Debug("Successfully loaded RDS CA certificates for secure TLS connection")
	return tlsConfig, nil
}
