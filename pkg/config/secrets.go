package config

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// SecretResolver resolves secret references to their actual values
type SecretResolver interface {
	Resolve(ctx context.Context, ref string) (string, error)
}

// OpResolver resolves 1Password secret references using the op CLI
type OpResolver struct {
	cliPath string
}

// NewOpResolver creates a resolver using the op CLI
func NewOpResolver() *OpResolver {
	return &OpResolver{cliPath: "op"}
}

// Resolve resolves a 1Password reference (op://vault/item/field) to its value
func (r *OpResolver) Resolve(ctx context.Context, ref string) (string, error) {
	if !strings.HasPrefix(ref, "op://") {
		return "", fmt.Errorf("invalid secret reference format: %s (must start with op://)", ref)
	}

	parts := strings.Split(strings.TrimPrefix(ref, "op://"), "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid secret reference format: %s (expected op://vault/item/field)", ref)
	}

	cmd := exec.CommandContext(ctx, r.cliPath, "read", ref)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("timeout resolving secret %s: %w", ref, ctx.Err())
		}
		return "", fmt.Errorf("failed to resolve secret %s: %w", ref, err)
	}

	return strings.TrimSpace(string(output)), nil
}

// hasOpReferences checks if data contains any op:// references
func hasOpReferences(data interface{}) bool {
	switch v := data.(type) {
	case map[string]map[string]interface{}:
		for _, moduleConfig := range v {
			if hasOpReferences(moduleConfig) {
				return true
			}
		}
	case map[string]interface{}:
		for _, value := range v {
			if str, ok := value.(string); ok && strings.HasPrefix(str, "op://") {
				return true
			} else if nested, ok := value.(map[string]interface{}); ok {
				if hasOpReferences(nested) {
					return true
				}
			}
		}
	}
	return false
}

// resolveSecretsInPlace recursively walks a map structure and resolves any op:// references
func resolveSecretsInPlace(ctx context.Context, resolver SecretResolver, data interface{}) error {
	// Fast path: skip if no op:// references exist
	if !hasOpReferences(data) {
		return nil
	}

	switch v := data.(type) {
	case map[string]map[string]interface{}:
		for _, moduleConfig := range v {
			if err := resolveSecretsInPlace(ctx, resolver, moduleConfig); err != nil {
				return err
			}
		}
	case map[string]interface{}:
		for key, value := range v {
			if str, ok := value.(string); ok && strings.HasPrefix(str, "op://") {
				resolved, err := resolver.Resolve(ctx, str)
				if err != nil {
					return fmt.Errorf("failed to resolve secret for key %s: %w", key, err)
				}
				v[key] = resolved
			} else if nested, ok := value.(map[string]interface{}); ok {
				if err := resolveSecretsInPlace(ctx, resolver, nested); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
