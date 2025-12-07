# SRE Database Configuration

### Setup Instructions

1. **Configure AWS credentials:**
   Ensure you have AWS credentials configured through one of these methods:
   - AWS CLI: `aws configure`
   - Environment variables: `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
   - IAM roles (if running on EC2)

2. **Test the database connection:**
   ```bash
   nsx sre db long-running --rds-instance your-rds-instance --debug
   ```

### Dynamic Database Connections

The NSX SRE CLI now uses **dynamic database connections only**. All database commands require the `--rds-instance` flag to specify which RDS instance to connect to directly.

**No pre-configured database support** - this simplifies the setup and makes the CLI more flexible for ad-hoc database operations.

### Required IAM Permissions

The AWS user/role needs these permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "rds-db:connect"
      ],
      "Resource": "arn:aws:rds-db:*:*:dbuser:*/sre_inc_responder"
    },
    {
      "Effect": "Allow",
      "Action": [
        "rds:DescribeDBInstances"
      ],
      "Resource": "*"
    }
  ]
}
```

### Usage Examples

```bash
# Connect to specific RDS instance (required)
nsx sre db long-running --rds-instance my-prod-db

# Connect to specific database on RDS instance
nsx sre db long-running --rds-instance my-prod-db --database mydb

# Specify different AWS region
nsx sre db long-running --rds-instance my-west-db --region us-west-2

# Set minimum duration threshold (default: 30 seconds)
nsx sre db long-running --rds-instance my-prod-db --min-duration 60

# Debug mode for troubleshooting
nsx sre db long-running --rds-instance my-prod-db --debug
```

## Environment Variables

- `NSX_CONFIG_PATH`: Override default config directory (default: `~/.config/nsx`)
- `NSX_SKIN`: Set default skin (default: `betdev`)

## Security Notes

- Database authentication uses AWS RDS IAM tokens
- Ensure your AWS credentials are properly secured
- The `sre_inc_responder` database user must exist and have appropriate permissions
- All database connections are established dynamically using AWS IAM authentication

## Improvements

- [ ] copy all feature [e] export
- [ ] list instances with search if none specified
- [ ] create Notino doc
