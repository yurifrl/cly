package awssdk

import (
	"context"
	"fmt"
	"log"
)

// ExampleGetSecretMap demonstrates how to use the GetSecretMap function
func ExampleGetSecretMap() {
	// Simple usage with default region (us-east-1)
	secretMap, err := GetSecretMap("my-secret-name")
	if err != nil {
		log.Fatalf("Failed to get secret: %v", err)
	}

	// Print the secret values
	for key, value := range secretMap {
		fmt.Printf("Key: %s, Value: %s\n", key, value)
	}
}

// ExampleGetSecretMapWithRegion demonstrates how to use the GetSecretMapWithRegion function
func ExampleGetSecretMapWithRegion() {
	// Usage with custom region
	secretMap, err := GetSecretMapWithRegion("my-secret-name", "us-west-2")
	if err != nil {
		log.Fatalf("Failed to get secret: %v", err)
	}

	// Print the secret values
	for key, value := range secretMap {
		fmt.Printf("Key: %s, Value: %s\n", key, value)
	}
}

// ExampleNewSecretsManagerClient demonstrates how to use the client directly
func ExampleNewSecretsManagerClient() {
	ctx := context.Background()

	// Create a new client
	client, err := NewSecretsManagerClient(ctx, "us-east-1")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Get a secret
	secretMap, err := client.GetSecretMap(ctx, "my-secret-name")
	if err != nil {
		log.Fatalf("Failed to get secret: %v", err)
	}

	// Print the secret values
	for key, value := range secretMap {
		fmt.Printf("Key: %s, Value: %s\n", key, value)
	}

	// Optionally list all secrets
	secrets, err := client.ListSecrets(ctx)
	if err != nil {
		log.Fatalf("Failed to list secrets: %v", err)
	}

	fmt.Printf("Found %d secrets\n", len(secrets))
	for _, secret := range secrets {
		fmt.Printf("Secret: %s\n", *secret.Name)
	}
}
