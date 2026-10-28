package envs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type fetchResult struct {
	Name   string
	Fields []Field
	Err    error
}

// discoverSecrets resolves glob patterns and explicit names into actual 1Password item names.
func discoverSecrets(ctx context.Context, opBinary string, config Config) ([]string, error) {
	var allItems []string

	for _, pattern := range config.Secrets {
		if strings.ContainsAny(pattern, "*?[") {
			// Glob pattern — list items from vault and filter
			items, err := listVaultItems(ctx, opBinary, config)
			if err != nil {
				return nil, fmt.Errorf("list vault items: %w", err)
			}
			for _, item := range items {
				matched, _ := filepath.Match(pattern, item)
				if matched {
					allItems = append(allItems, item)
				}
			}
		} else {
			// Explicit item name
			allItems = append(allItems, pattern)
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0, len(allItems))
	for _, item := range allItems {
		if !seen[item] {
			seen[item] = true
			unique = append(unique, item)
		}
	}
	return unique, nil
}

func listVaultItems(ctx context.Context, opBinary string, config Config) ([]string, error) {
	cmd := exec.CommandContext(ctx, opBinary, "item", "list",
		"--vault", config.Vault,
		"--account", config.Account,
		"--format", "json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %s", err, stderr.String())
	}

	var items []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &items); err != nil {
		return nil, fmt.Errorf("decode item list: %w", err)
	}

	names := make([]string, len(items))
	for i, item := range items {
		names[i] = item.Title
	}
	return names, nil
}

func fetchItem(ctx context.Context, opBinary string, config Config, name string) fetchResult {
	cmd := exec.CommandContext(ctx, opBinary, "item", "get", name,
		"--vault", config.Vault,
		"--account", config.Account,
		"--format", "json",
		"--reveal")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fetchResult{Name: name, Err: fmt.Errorf("%w: %s", err, stderr.String())}
	}

	var parsed item
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil {
		return fetchResult{Name: name, Err: fmt.Errorf("decode: %w", err)}
	}

	return fetchResult{Name: name, Fields: decodeFields(parsed)}
}

func fetchAll(ctx context.Context, opBinary string, config Config, names []string) <-chan fetchResult {
	results := make(chan fetchResult, len(names))
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()
			results <- fetchItem(ctx, opBinary, config, n)
		}(name)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}
