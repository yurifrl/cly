package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"github.com/NSXBet/nsx-cli/shared/awssdk"
	"github.com/NSXBet/nsx-cli/shared/interact"
	"github.com/NSXBet/nsx-cli/team/sre/internal/models"
)

var connectRegion string

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to RDS/Aurora database with admin credentials",
	Long: heredoc.Doc(`
		Connect to RDS/Aurora database with admin credentials

		This command provides an interactive way to connect to AWS RDS instances and
		Aurora clusters using admin credentials stored in AWS Secrets Manager.

		Features:
		  • Lists all RDS instances and Aurora clusters in the specified region
		  • Interactive selection with built-in filtering
		  • Automatic credential retrieval from Secrets Manager
		  • Direct connection via mysql or psql client
		  • Supports MySQL, Aurora MySQL, PostgreSQL, and Aurora PostgreSQL

		Prerequisites:
		  • AWS CLI configured with appropriate permissions
		  • mysql client installed (for MySQL databases)
		  • psql client installed (for PostgreSQL databases)
		  • Database must have Secrets Manager integration enabled

		Examples:
		  # Connect to a database in us-east-1 (default)
		  nsx sre db connect

		  # Connect to a database in us-west-2
		  nsx sre db connect --region us-west-2

		  # Enable debug output for troubleshooting
		  nsx sre db connect --debug

		Required AWS Permissions:
		  - rds:DescribeDBInstances
		  - rds:DescribeDBClusters
		  - secretsmanager:GetSecretValue
	`),
	RunE: runConnectCommand,
}

func init() {
	connectCmd.Flags().StringVarP(&connectRegion, "region", "r", "us-east-1", "AWS region")
	dbCmd.AddCommand(connectCmd)
}

func runConnectCommand(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. List databases (instances + clusters)
	targets, err := listDatabases(ctx, connectRegion)
	if err != nil {
		return err
	}

	if len(targets) == 0 {
		interact.Info("No databases found in region %s", connectRegion)
		return nil
	}

	// 2. Show selection UI
	target, err := selectDatabase(targets)
	if err != nil {
		return err
	}

	// 3. Get credentials from Secrets Manager
	username, password, err := getCredentials(ctx, target)
	if err != nil {
		return err
	}

	// 4. Exec into mysql/psql
	return connectToDatabase(target, username, password)
}

func listDatabases(ctx context.Context, region string) ([]models.DatabaseTarget, error) {
	interact.Info("Fetching RDS instances and Aurora clusters from %s...", region)

	client, err := awssdk.NewRDSClient(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("failed to create RDS client: %w", err)
	}

	var targets []models.DatabaseTarget

	// List RDS instances
	instances, err := client.ListDBInstances(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list RDS instances: %w", err)
	}

	instanceCount := 0
	for _, instance := range instances {
		// Skip cluster members (they're accessed via cluster endpoint)
		if instance.DBClusterIdentifier != nil && *instance.DBClusterIdentifier != "" {
			interact.Debug("Skipping cluster member: %s", *instance.DBInstanceIdentifier)
			continue
		}

		// Skip if no endpoint
		if instance.Endpoint == nil {
			interact.Debug("Skipping instance without endpoint: %s", *instance.DBInstanceIdentifier)
			continue
		}

		target := models.DatabaseTarget{
			Type:          "instance",
			Identifier:    *instance.DBInstanceIdentifier,
			Engine:        *instance.Engine,
			EngineVersion: *instance.EngineVersion,
			Endpoint:      *instance.Endpoint.Address,
			Port:          *instance.Endpoint.Port,
			Status:        *instance.DBInstanceStatus,
			SecretArn:     nil, // Will set if available
			IsCluster:     false,
		}

		// Check for MasterUserSecret
		if instance.MasterUserSecret != nil && instance.MasterUserSecret.SecretArn != nil {
			target.SecretArn = instance.MasterUserSecret.SecretArn
		}

		targets = append(targets, target)
		instanceCount++
	}

	// List Aurora clusters
	clusters, err := client.ListDBClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list Aurora clusters: %w", err)
	}

	clusterCount := 0
	for _, cluster := range clusters {
		// Skip if no endpoint
		if cluster.Endpoint == nil || *cluster.Endpoint == "" {
			interact.Debug("Skipping cluster without endpoint: %s", *cluster.DBClusterIdentifier)
			continue
		}

		target := models.DatabaseTarget{
			Type:          "cluster",
			Identifier:    *cluster.DBClusterIdentifier,
			Engine:        *cluster.Engine,
			EngineVersion: *cluster.EngineVersion,
			Endpoint:      *cluster.Endpoint,
			Port:          *cluster.Port,
			Status:        *cluster.Status,
			SecretArn:     nil, // Will set if available
			IsCluster:     true,
		}

		// Check for MasterUserSecret
		if cluster.MasterUserSecret != nil && cluster.MasterUserSecret.SecretArn != nil {
			target.SecretArn = cluster.MasterUserSecret.SecretArn
		}

		targets = append(targets, target)
		clusterCount++
	}

	interact.Success("Found %d databases (%d instances, %d clusters)",
		len(targets), instanceCount, clusterCount)

	return targets, nil
}

func selectDatabase(targets []models.DatabaseTarget) (*models.DatabaseTarget, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no databases found")
	}

	// Build display options
	options := make([]string, len(targets))
	for i, target := range targets {
		options[i] = target.DisplayString()
	}

	// Custom filter function for search (case-insensitive)
	searchFilter := func(filter string, value string, index int) bool {
		return strings.Contains(strings.ToLower(value), strings.ToLower(filter))
	}

	// Show selection prompt with filtering
	var selectedIndex int
	prompt := &survey.Select{
		Message:  fmt.Sprintf("Select a database (%d available):", len(targets)),
		Options:  options,
		PageSize: 15,
		Filter:   searchFilter,
	}

	err := survey.AskOne(prompt, &selectedIndex)
	if err != nil {
		return nil, fmt.Errorf("database selection cancelled: %w", err)
	}

	selected := &targets[selectedIndex]
	interact.Info("Selected: %s", selected.Identifier)

	return selected, nil
}

func getCredentials(ctx context.Context, target *models.DatabaseTarget) (username, password string, err error) {
	// Check if MasterUserSecret ARN is available
	if target.SecretArn == nil || *target.SecretArn == "" {
		return "", "", fmt.Errorf(heredoc.Doc(`
			Database '%s' does not have Secrets Manager integration.

			This database may be using:
			  - IAM database authentication
			  - Legacy master user credentials
			  - Manual Secrets Manager secret (not managed by RDS)

			To enable Secrets Manager integration:
			  1. Go to RDS Console
			  2. Select your database
			  3. Click "Modify"
			  4. Under "Credentials management", choose "Manage master credentials in AWS Secrets Manager"
		`), target.Identifier)
	}

	interact.Info("Retrieving admin credentials from Secrets Manager...")
	interact.Debug("Secret ARN: %s", *target.SecretArn)

	// Extract region from ARN for client initialization
	// ARN format: arn:aws:secretsmanager:region:account:secret:name-SUFFIX
	region, err := parseRegionFromArn(*target.SecretArn)
	if err != nil {
		return "", "", err
	}

	interact.Debug("Using ARN as secret ID, Region: %s", region)

	// Get secret value using existing AWS SDK wrapper
	// AWS Secrets Manager accepts full ARNs as secret IDs
	secretMap, err := awssdk.GetSecretMapWithRegion(*target.SecretArn, region)
	if err != nil {
		return "", "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Extract username and password
	username, ok := secretMap["username"]
	if !ok {
		return "", "", fmt.Errorf("secret does not contain 'username' field")
	}

	password, ok = secretMap["password"]
	if !ok {
		return "", "", fmt.Errorf("secret does not contain 'password' field")
	}

	if username == "" || password == "" {
		return "", "", fmt.Errorf("username or password is empty in secret")
	}

	interact.Success("Retrieved admin credentials for user: %s", username)

	return username, password, nil
}

// parseRegionFromArn extracts the region from an ARN
func parseRegionFromArn(arn string) (region string, err error) {
	// ARN format: arn:aws:secretsmanager:region:account:secret:name-SUFFIX
	parts := strings.Split(arn, ":")
	if len(parts) < 4 {
		return "", fmt.Errorf("invalid ARN format: %s", arn)
	}

	region = parts[3]
	if region == "" {
		return "", fmt.Errorf("region not found in ARN: %s", arn)
	}

	return region, nil
}

func connectToDatabase(target *models.DatabaseTarget, username, password string) error {
	interact.Info("Connecting to %s (%s)...", target.Identifier, target.Engine)

	// Check database status
	if target.Status != "available" {
		interact.Warn("Database status is '%s' (not available)", target.Status)
		interact.Info("Connection may fail if database is not ready")
	}

	var binary string
	var args []string

	if target.IsMySQL() {
		// MySQL connection
		binary = "mysql"
		args = []string{
			"-h", target.Endpoint,
			"-P", fmt.Sprintf("%d", target.Port),
			"-u", username,
			fmt.Sprintf("-p%s", password), // No space between -p and password
			"--ssl-mode=REQUIRED",         // RDS requires SSL
		}

		interact.Info("Connecting to MySQL database...")

	} else if target.IsPostgreSQL() {
		// PostgreSQL connection
		binary = "psql"
		args = []string{
			"-h", target.Endpoint,
			"-p", fmt.Sprintf("%d", target.Port),
			"-U", username,
			"-d", target.GetDefaultDatabase(),
		}

		// PostgreSQL password via environment variable (safer than command line)
		if err := os.Setenv("PGPASSWORD", password); err != nil {
			return fmt.Errorf("failed to set PGPASSWORD environment variable: %w", err)
		}
		defer func() {
			_ = os.Unsetenv("PGPASSWORD") // Clean up (though exec will replace process)
		}()

		interact.Info("Connecting to PostgreSQL database...")

	} else {
		return fmt.Errorf("unsupported database engine: %s", target.Engine)
	}

	// Check if binary exists
	binaryPath, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf(heredoc.Doc(`
			%s client not found. Please install %s client:

			  macOS:         brew install %s
			  Ubuntu/Debian: apt-get install %s-client
			  Amazon Linux:  yum install %s
		`), binary, binary, binary, binary, binary)
	}

	interact.Debug("Using %s binary: %s", binary, binaryPath)
	interact.Debug("Connection endpoint: %s:%d", target.Endpoint, target.Port)

	// Build command with binary as first arg (required by Exec)
	execArgs := append([]string{binary}, args...)

	interact.Success("Launching %s client...\n", binary)

	// Replace current process with database client
	// This gives user full interactive shell experience and prevents password in process list
	err = syscall.Exec(binaryPath, execArgs, os.Environ())
	if err != nil {
		return fmt.Errorf("failed to exec into %s: %w", binary, err)
	}

	// This line will never be reached if Exec succeeds
	return nil
}
