package awssdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// SecretsManagerClient wraps the AWS Secrets Manager client
type SecretsManagerClient struct {
	client *secretsmanager.Client
}

// NewSecretsManagerClient creates a new Secrets Manager client using AWS CLI credentials
func NewSecretsManagerClient(ctx context.Context, region string) (*SecretsManagerClient, error) {
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

	client := secretsmanager.NewFromConfig(cfg)
	return &SecretsManagerClient{client: client}, nil
}

// GetSecretMap retrieves a secret from AWS Secrets Manager and returns it as a map[string]string
func (s *SecretsManagerClient) GetSecretMap(ctx context.Context, secretName string) (map[string]string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	}

	result, err := s.client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret '%s': %w", secretName, err)
	}

	// If the secret is binary, return an error as we expect string secrets
	if result.SecretBinary != nil {
		return nil, fmt.Errorf("secret '%s' is binary, expected string", secretName)
	}

	// If the secret is a string, try to parse it as JSON
	if result.SecretString != nil {
		var secretMap map[string]string
		if err := json.Unmarshal([]byte(*result.SecretString), &secretMap); err != nil {
			// If it's not valid JSON, treat it as a single key-value pair
			secretMap = map[string]string{
				"value": *result.SecretString,
			}
		}
		return secretMap, nil
	}

	return map[string]string{}, nil
}

// GetSecretMap is a convenience function that creates a client and retrieves a secret
func GetSecretMap(secretName string) (map[string]string, error) {
	ctx := context.Background()

	// Default to us-east-1 if no region is specified
	// You can modify this or make it configurable
	client, err := NewSecretsManagerClient(ctx, "us-east-1")
	if err != nil {
		return nil, err
	}

	return client.GetSecretMap(ctx, secretName)
}

// GetSecretMapWithRegion allows specifying a custom region
func GetSecretMapWithRegion(secretName, region string) (map[string]string, error) {
	ctx := context.Background()

	client, err := NewSecretsManagerClient(ctx, region)
	if err != nil {
		return nil, err
	}

	return client.GetSecretMap(ctx, secretName)
}

// ListSecrets lists all secrets in the account (optional utility function)
func (s *SecretsManagerClient) ListSecrets(ctx context.Context) ([]types.SecretListEntry, error) {
	input := &secretsmanager.ListSecretsInput{}

	var secrets []types.SecretListEntry
	paginator := secretsmanager.NewListSecretsPaginator(s.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}
		secrets = append(secrets, page.SecretList...)
	}

	return secrets, nil
}

// SecretResult represents a secret name or an error
type SecretResult struct {
	Name  string
	Error error
}

// ListSecretsAsync lists secrets asynchronously, streaming results through a channel
func (s *SecretsManagerClient) ListSecretsAsync(ctx context.Context) <-chan SecretResult {
	resultChan := make(chan SecretResult, 10) // Buffer for better throughput

	go func() {
		defer close(resultChan)

		input := &secretsmanager.ListSecretsInput{}
		paginator := secretsmanager.NewListSecretsPaginator(s.client, input)

		for paginator.HasMorePages() {
			// Check if context was cancelled
			select {
			case <-ctx.Done():
				resultChan <- SecretResult{Error: ctx.Err()}
				return
			default:
			}

			page, err := paginator.NextPage(ctx)
			if err != nil {
				resultChan <- SecretResult{Error: fmt.Errorf("failed to list secrets: %w", err)}
				return
			}

			// Send each secret name as we get them
			for _, secret := range page.SecretList {
				if secret.Name != nil {
					select {
					case resultChan <- SecretResult{Name: *secret.Name}:
					case <-ctx.Done():
						resultChan <- SecretResult{Error: ctx.Err()}
						return
					}
				}
			}
		}
	}()

	return resultChan
}
