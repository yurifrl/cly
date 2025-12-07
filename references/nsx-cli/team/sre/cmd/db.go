package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/team/sre/internal/database"
	"github.com/NSXBet/nsx-cli/team/sre/internal/views"
	"github.com/NSXBet/nsx-cli/team/sre/srecfg"
)

const (
	// MinimumAllowedDuration is the minimum value allowed for min-duration flag
	MinimumAllowedDuration = 10 // seconds
)

var (
	minDuration int
	dbName      string
	rdsInstance string
	region      string
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database operations for SRE",
	Long: heredoc.Doc(`
		This command provides database operations and utilities for Site Reliability Engineering tasks.
		
		Available operations:
		- long-running: List and manage long running queries
		- kill: Kill database processes (via long-running interface)
		- refresh: Refresh query results in real-time
		- copy: Copy query text to clipboard
		
		Features:
		- AWS RDS IAM authentication with dynamic instance connections
		- Support for MySQL, PostgreSQL, Aurora MySQL, and Aurora PostgreSQL
		- Connection pooling for better performance
		- Interactive table interface with sorting and real-time updates
		- Status messages for user feedback
	`),
}

var longRunningCmd = &cobra.Command{
	Use:   "long-running",
	Short: "List long running queries",
	Long: heredoc.Doc(`
		List long running queries from MySQL databases. Use --min-duration to filter by minimum execution time in seconds.
		
		This command connects directly to any RDS instance using AWS RDS IAM authentication.
		The --rds-instance flag is required to specify which RDS instance to connect to.
		
		Prerequisites:
		  • AWS CLI configured with 'DeveloperBase' role
		  • Proper IAM permissions for RDS access
		  • Network access to RDS instances (VPN if required)
		
		Examples:
		  # Connect to a specific RDS instance
		  nsx sre db long-running --rds-instance my-rds-instance
		  
		  # Set minimum duration threshold
		  nsx sre db long-running --rds-instance my-rds-instance --min-duration 60
		  
		  # Specify a specific database name
		  nsx sre db long-running --rds-instance my-rds-instance --database mydb  --region us-west-2
		  
		  # Debug mode for detailed connection info
		  nsx sre db long-running --rds-instance my-rds-instance --debug
		  
		Troubleshooting:
		  If authentication fails, ensure you're logged in with the DeveloperBase role:
		    aws configure sso
		    aws sts get-caller-identity
		  
		  The output should show "AWSReservedSSO_DevelopersBase" in the role ARN.
	`),
	RunE: runLongRunningQueries,
}

func init() {
	RootCmd.AddCommand(dbCmd)
	dbCmd.AddCommand(longRunningCmd)

	longRunningCmd.Flags().
		IntVarP(&minDuration, "min-duration", "m", 30, fmt.Sprintf("Minimum query duration in seconds (minimum allowed: %d)", MinimumAllowedDuration))
	longRunningCmd.Flags().StringVarP(&dbName, "database", "n", "", "Specific database name to connect to")
	longRunningCmd.Flags().StringVarP(&rdsInstance, "rds-instance", "i", "", "RDS instance identifier to connect to (required)")
	longRunningCmd.Flags().StringVarP(&region, "region", "r", "us-east-1", "AWS region for the RDS instance")

	// Make rds-instance flag required
	_ = longRunningCmd.MarkFlagRequired("rds-instance")
}

func runLongRunningQueries(cmd *cobra.Command, args []string) error {
	// Validate minimum duration
	if minDuration < MinimumAllowedDuration {
		return fmt.Errorf("minimum duration must be at least %d seconds (provided: %d)", MinimumAllowedDuration, minDuration)
	}

	interact.Info("Checking for long running queries (min duration: %d seconds)...", minDuration)
	interact.Debug("Using RDS instance: %s in region: %s", rdsInstance, region)

	// Create database config for the specified RDS instance
	dbConfig := srecfg.DatabaseConnection{
		DBInstanceIdentifier: rdsInstance,
		DBName:               dbName,
		Region:               region,
	}

	// Check the specified RDS instance
	results, err := checkDatabaseLongRunning(rdsInstance, dbConfig, minDuration)
	if err != nil {
		return handleDatabaseError(err, rdsInstance)
	}

	if len(results) == 0 {
		interact.Success("No long running queries found")
		return nil
	}

	// Create a refresh function that re-queries the database
	refreshFunc := func() ([]views.QueryResult, error) {
		results, err := checkDatabaseLongRunning(rdsInstance, dbConfig, minDuration)
		if err != nil {
			// For refresh operations, return the raw error so status messages work properly
			return nil, err
		}
		return results, nil
	}

	// Create a kill function that executes KILL command on the database
	killFunc := func(processID int) error {
		err := killDatabaseProcess(rdsInstance, dbConfig, processID)
		if err != nil {
			// For kill operations, return raw error as it's handled by status messages
			return err
		}
		return nil
	}

	// Display results using the DB process list view with refresh and kill capabilities
	view := views.NewDBProcessListView("Long Running Queries", results, refreshFunc, killFunc)

	// Ensure connections are cleaned up when done
	defer func() {
		pool := database.GetPool()
		pool.CleanupExpired()
	}()

	return view.Display()
}

func checkDatabaseLongRunning(dbName string, dbConfig srecfg.DatabaseConnection, minDuration int) ([]views.QueryResult, error) {
	// Create context with 30-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get connection from pool
	pool := database.GetPool()
	conn, err := pool.GetConnection(ctx, dbName, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection from pool: %v", err)
	}

	db := conn.DB()
	engine := conn.Engine()

	// Use engine-specific query
	switch engine {
	case "mysql", "aurora-mysql":
		return checkMySQLLongRunning(db, minDuration)
	case "postgres", "aurora-postgresql":
		return checkPostgreSQLLongRunning(db, minDuration)
	default:
		return nil, fmt.Errorf("unsupported database engine for long-running queries: %s", engine)
	}
}

func checkMySQLLongRunning(db *sql.DB, minDuration int) ([]views.QueryResult, error) {
	// Query for long running processes in MySQL
	query := `
		SELECT 
			ID,
			USER,
			HOST,
			COALESCE(DB, '') as DB,
			COMMAND,
			TIME,
			STATE,
			COALESCE(INFO, '') as QUERY_PREVIEW
		FROM 
			INFORMATION_SCHEMA.PROCESSLIST 
		WHERE 
			COMMAND != 'Sleep' 
			AND TIME >= ?
		ORDER BY 
			TIME DESC
	`

	rows, err := db.Query(query, minDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to execute MySQL query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var results []views.QueryResult

	for rows.Next() {
		var id int
		var user, host, database, command, state, queryPreview string
		var duration int

		if err := rows.Scan(&id, &user, &host, &database, &command, &duration, &state, &queryPreview); err != nil {
			return nil, fmt.Errorf("failed to scan MySQL row: %v", err)
		}

		// Skip ignored users (system users)
		if views.ShouldIgnoreUser(user) {
			interact.Debug("Skipping system user: %s", user)
			continue
		}

		// Format duration
		durationStr := views.FormatDuration(duration)

		results = append(results, views.QueryResult{
			ID:           id,
			User:         user,
			Host:         host,
			Database:     database,
			Command:      command,
			Duration:     duration,
			DurationStr:  durationStr,
			State:        state,
			QueryPreview: queryPreview,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func checkPostgreSQLLongRunning(db *sql.DB, minDuration int) ([]views.QueryResult, error) {
	// Query for long running processes in PostgreSQL
	query := `
		SELECT 
			pid as ID,
			usename as USER,
			client_addr::text || ':' || client_port as HOST,
			COALESCE(datname, '') as DB,
			state as COMMAND,
			EXTRACT(EPOCH FROM (now() - query_start))::int as TIME,
			state as STATE,
			COALESCE(LEFT(query, 500), '') as QUERY_PREVIEW
		FROM 
			pg_stat_activity 
		WHERE 
			state != 'idle'
			AND pid != pg_backend_pid()
			AND EXTRACT(EPOCH FROM (now() - query_start)) >= $1
		ORDER BY 
			query_start ASC
	`

	rows, err := db.Query(query, minDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PostgreSQL query: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var results []views.QueryResult

	for rows.Next() {
		var id int
		var user, host, database, command, state, queryPreview string
		var duration int

		if err := rows.Scan(&id, &user, &host, &database, &command, &duration, &state, &queryPreview); err != nil {
			return nil, fmt.Errorf("failed to scan PostgreSQL row: %v", err)
		}

		// Skip ignored users (system users)
		if views.ShouldIgnoreUser(user) {
			interact.Debug("Skipping system user: %s", user)
			continue
		}

		// Format duration
		durationStr := views.FormatDuration(duration)

		results = append(results, views.QueryResult{
			ID:           id,
			User:         user,
			Host:         host,
			Database:     database,
			Command:      command,
			Duration:     duration,
			DurationStr:  durationStr,
			State:        state,
			QueryPreview: queryPreview,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// killDatabaseProcess kills a specific database process by ID
func killDatabaseProcess(dbName string, dbConfig srecfg.DatabaseConnection, processID int) error {
	ctx := context.Background()

	// Get connection pool
	pool := database.GetPool()

	// Get connection from pool
	conn, err := pool.GetConnection(ctx, dbName, dbConfig)
	if err != nil {
		return fmt.Errorf("failed to get connection from pool: %v", err)
	}

	db := conn.DB()
	engine := conn.Engine()

	// Use engine-specific kill logic
	switch engine {
	case "mysql", "aurora-mysql":
		return killMySQLProcess(db, processID)
	case "postgres", "aurora-postgresql":
		return killPostgreSQLProcess(db, processID)
	default:
		return fmt.Errorf("unsupported database engine for killing processes: %s", engine)
	}
}

func killMySQLProcess(db *sql.DB, processID int) error {
	// Check if the process exists in this database
	checkQuery := "SELECT ID FROM INFORMATION_SCHEMA.PROCESSLIST WHERE ID = ?"
	var foundID int
	err := db.QueryRow(checkQuery, processID).Scan(&foundID)
	if err != nil {
		return fmt.Errorf("process %d not found in MySQL database", processID)
	}

	// Process found, execute RDS kill query procedure
	killQuery := "CALL mysql.rds_kill_query(?)"
	_, err = db.Exec(killQuery, processID)
	if err != nil {
		return fmt.Errorf("failed to kill MySQL process %d: %v", processID, err)
	}

	return nil
}

func killPostgreSQLProcess(db *sql.DB, processID int) error {
	// Check if the process exists in this database
	checkQuery := "SELECT pid FROM pg_stat_activity WHERE pid = $1 AND pid != pg_backend_pid()"
	var foundID int
	err := db.QueryRow(checkQuery, processID).Scan(&foundID)
	if err != nil {
		return fmt.Errorf("process %d not found in PostgreSQL database", processID)
	}

	// Process found, terminate the query/connection
	// First try to cancel the query gracefully
	killQuery := "SELECT pg_cancel_backend($1)"
	var result bool
	err = db.QueryRow(killQuery, processID).Scan(&result)
	if err != nil {
		return fmt.Errorf("failed to cancel PostgreSQL process %d: %v", processID, err)
	}

	if !result {
		// If cancel didn't work, try to terminate the connection
		killQuery = "SELECT pg_terminate_backend($1)"
		err = db.QueryRow(killQuery, processID).Scan(&result)
		if err != nil {
			return fmt.Errorf("failed to terminate PostgreSQL process %d: %v", processID, err)
		}

		if !result {
			return fmt.Errorf("failed to kill PostgreSQL process %d: backend returned false", processID)
		}
	}

	return nil
}

// handleDatabaseError provides user-friendly error messages for database connection issues
func handleDatabaseError(err error, rdsInstance string) error {
	errMsg := err.Error()

	// Check for common AWS authentication errors
	if strings.Contains(errMsg, "NoCredentialsError") ||
		strings.Contains(errMsg, "SharedConfigCredentialsError") ||
		strings.Contains(errMsg, "failed to load AWS config") {
		return fmt.Errorf(heredoc.Doc(`
			❌ AWS authentication failed for RDS instance '%s'

			🔧 Please ensure you are logged in with the 'DeveloperBase' role:

			   1. Configure AWS CLI with DeveloperBase role:
			      aws configure sso
			      
			   2. Or set environment variables:
			      export AWS_PROFILE=DeveloperBase
			      
			   3. Verify your authentication:
			      aws sts get-caller-identity
			      
			   Note: The role ARN should contain "AWSReservedSSO_DevelopersBase"

			🔍 Error details: %v
		`), rdsInstance, err)
	}

	// Check for permission-related errors
	if strings.Contains(errMsg, "AccessDenied") ||
		strings.Contains(errMsg, "UnauthorizedOperation") ||
		strings.Contains(errMsg, "not authorized") {
		return fmt.Errorf(heredoc.Doc(`
			❌ Access denied for RDS instance '%s'

			🔧 Please check your AWS permissions:

			   1. Ensure you have the 'DeveloperBase' role assigned
			   2. Verify the following IAM permissions:
			      • rds-db:connect on RDS instances
			      • rds:DescribeDBInstances
			      
			   3. Check if the RDS instance exists and is accessible:
			      aws rds describe-db-instances --db-instance-identifier %s --region %s

			🔍 Error details: %v
		`), rdsInstance, rdsInstance, region, err)
	}

	// Check for timeout errors first (more specific)
	if strings.Contains(errMsg, "context deadline exceeded") ||
		strings.Contains(errMsg, "timeout") {
		return fmt.Errorf(heredoc.Doc(`
			❌ Connection timeout to RDS instance '%s' (30 seconds)

			🔧 The connection attempt timed out. Please check:

			   1. Network connectivity - ensure you're connected to VPN if required
			   2. RDS instance is running and accessible:
			      aws rds describe-db-instances --db-instance-identifier %s --region %s
			   3. Security groups allow connections from your IP
			   4. AWS region is correct (currently: %s)
			   5. RDS instance identifier is correct

			💡 If the RDS instance is starting up, it may take a few minutes to accept connections.

			🔍 Error details: %v
		`), rdsInstance, rdsInstance, region, region, err)
	}

	// Check for other network/connectivity errors
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") {
		return fmt.Errorf(heredoc.Doc(`
			❌ Connection failed to RDS instance '%s'

			🔧 Please check network connectivity:

			   1. Verify the RDS instance identifier is correct
			   2. Ensure you're connected to the correct network/VPN
			   3. Check the AWS region is correct (currently: %s)
			   4. Verify the RDS instance is running:
			      aws rds describe-db-instances --db-instance-identifier %s --region %s

			🔍 Error details: %v
		`), rdsInstance, region, rdsInstance, region, err)
	}

	// Check for RDS-specific authentication errors
	if strings.Contains(errMsg, "authentication failed") ||
		strings.Contains(errMsg, "Access denied for user") ||
		strings.Contains(errMsg, "sre_inc_responder") {
		return fmt.Errorf(heredoc.Doc(`
			❌ Database authentication failed for RDS instance '%s'

			🔧 Please check database user permissions:

			   1. Ensure the 'sre_inc_responder' user exists in the database
			   2. Verify the user has proper IAM database authentication enabled
			   3. Check if you have the correct AWS role (DeveloperBase) assigned
			   4. Test AWS CLI authentication:
			      aws sts get-caller-identity
			      
			   Note: The role ARN should contain "AWSReservedSSO_DevelopersBase"

			🔍 Error details: %v
		`), rdsInstance, err)
	}

	// Default error message with helpful context
	return fmt.Errorf(heredoc.Doc(`
		❌ Failed to connect to RDS instance '%s'

		🔧 Troubleshooting steps:

		   1. Verify AWS authentication with DeveloperBase role:
		      aws sts get-caller-identity
		      (Should show "AWSReservedSSO_DevelopersBase" in role ARN)
		      
		   2. Check RDS instance exists:
		      aws rds describe-db-instances --db-instance-identifier %s --region %s
		      
		   3. Enable debug mode for more details:
		      nsx sre db long-running --rds-instance %s --debug

		🔍 Error details: %v
	`), rdsInstance, rdsInstance, region, rdsInstance, err)
}
