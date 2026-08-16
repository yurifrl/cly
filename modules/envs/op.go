package envs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type fetchResult struct {
	Secret Secret
	Fields []Field
	Files  []Attachment
	Err    error
}

type sessions map[string]string

func signIn(ctx context.Context, opBinary string, secrets []Secret) (sessions, error) {
	tokens := make(sessions)
	for _, secret := range secrets {
		if _, exists := tokens[secret.Account]; exists {
			continue
		}
		command := exec.CommandContext(ctx, opBinary, "signin", "--account", secret.Account)
		if output, err := command.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("sign in to %s: %w: %s", secret.Account, err, output)
		}
		tokens[secret.Account] = ""
	}
	return tokens, nil
}

func fetchAll(ctx context.Context, opBinary string, secrets []Secret, tokens sessions) <-chan fetchResult {
	results := make(chan fetchResult, len(secrets))
	for _, secret := range secrets {
		go func(secret Secret) {
			command := opCommand(ctx, opBinary, tokens, secret.Account, "item", "get", secret.Name, "--vault", secret.Vault, "--account", secret.Account, "--format=json")
			output, err := command.Output()
			if err != nil {
				results <- fetchResult{Secret: secret, Err: fmt.Errorf("fetch %s: %w", secret.Name, err)}
				return
			}
			item, err := parseItem(output)
			if err != nil {
				results <- fetchResult{Secret: secret, Err: err}
				return
			}
			results <- fetchResult{Secret: secret, Fields: fieldsFromItem(item), Files: item.Files}
		}(secret)
	}
	return results
}

func downloadAttachments(ctx context.Context, opBinary, destination string, secret Secret, files []Attachment, tokens sessions) (int, error) {
	count := 0
	for _, file := range files {
		path := filepath.Join(destination, file.Name)
		command := opCommand(ctx, opBinary, tokens, secret.Account, "read", fmt.Sprintf("op://%s/%s/%s", secret.Vault, secret.Name, file.ID), "--account", secret.Account, "--out-file", path, "--force")
		if output, err := command.CombinedOutput(); err != nil {
			return count, fmt.Errorf("download %s: %w: %s", file.Name, err, output)
		}
		count++
	}
	return count, nil
}

func opCommand(ctx context.Context, opBinary string, tokens sessions, account string, args ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, opBinary, args...)
	if token := tokens[account]; token != "" {
		command.Env = append(os.Environ(), "OP_SESSION="+token)
	}
	return command
}
