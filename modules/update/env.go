package update

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// loadBuildEnv resolves env vars for `cly u` builds.
//
// Strategy:
//  1. If sourceDir/.env exists, parse it.
//  2. Else if refresh is true AND sourceDir/.env.op exists, run
//     `op inject` to a temp file, parse it, clean it up.
//  3. Else return os.Environ() unchanged.
//
// The op CLI is slow (network call to 1Password); skipping it on every
// `cly u` keeps builds fast. Use --refresh-env to force re-resolution.
func loadBuildEnv(sourceDir string, refresh bool) ([]string, error) {
	env := os.Environ()
	envPath := filepath.Join(sourceDir, ".env")
	opTplPath := filepath.Join(sourceDir, ".env.op")

	if _, err := os.Stat(envPath); err == nil {
		extra, perr := parseEnvFile(envPath)
		if perr != nil {
			return env, fmt.Errorf("parse .env: %w", perr)
		}
		return append(env, extra...), nil
	}

	if !refresh {
		return env, nil
	}

	if _, err := os.Stat(opTplPath); err != nil {
		return env, nil
	}
	if _, err := exec.LookPath("op"); err != nil {
		return env, nil
	}

	tmp, err := os.CreateTemp("", "cly-env-*.env")
	if err != nil {
		return env, err
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)

	cmd := exec.Command("op", "inject",
		"--account", "my.1password.com",
		"-i", opTplPath,
		"-o", tmpPath,
		"-f",
	)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return env, fmt.Errorf("op inject: %w", err)
	}

	extra, err := parseEnvFile(tmpPath)
	if err != nil {
		return env, fmt.Errorf("parse op-injected env: %w", err)
	}
	return append(env, extra...), nil
}

// parseEnvFile reads a simple KEY=VALUE file. Surrounding double-quotes on
// the value are stripped. Blank lines and lines starting with '#' are ignored.
func parseEnvFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "\"'")
		out = append(out, k+"="+v)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
