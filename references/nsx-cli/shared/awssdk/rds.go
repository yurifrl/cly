package awssdk

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// RDSClient wraps the AWS RDS client
type RDSClient struct {
	client *rds.Client
}

// NewRDSClient creates a new RDS client using AWS CLI credentials
func NewRDSClient(ctx context.Context, region string) (*RDSClient, error) {
	// Load AWS configuration using the same mechanism as aws-cli
	// This will automatically use credentials from:
	// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
	// 2. Shared credentials file (~/.aws/credentials)
	// 3. IAM roles (if running on EC2/ECS/Lambda)
	// 4. AWS SSO profiles
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := rds.NewFromConfig(cfg)
	return &RDSClient{client: client}, nil
}

// ListDBInstances lists all RDS instances with pagination
func (r *RDSClient) ListDBInstances(ctx context.Context) ([]types.DBInstance, error) {
	var instances []types.DBInstance

	paginator := rds.NewDescribeDBInstancesPaginator(r.client, &rds.DescribeDBInstancesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB instances: %w", err)
		}
		instances = append(instances, page.DBInstances...)
	}

	return instances, nil
}

// ListDBClusters lists all Aurora clusters with pagination
func (r *RDSClient) ListDBClusters(ctx context.Context) ([]types.DBCluster, error) {
	var clusters []types.DBCluster

	paginator := rds.NewDescribeDBClustersPaginator(r.client, &rds.DescribeDBClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe DB clusters: %w", err)
		}
		clusters = append(clusters, page.DBClusters...)
	}

	return clusters, nil
}
