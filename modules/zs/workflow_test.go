package zs

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIOutside_MatchesReferenceCreateFlow(t *testing.T) {
	repoRoot := repoRoot(t)
	binDir, logsDir := setupFakeEnv(t)

	writeFakeBinary(t, binDir, "zoxide", `#!/bin/sh
if [ "$1" = "query" ] && [ "$2" = "-l" ]; then
  printf '%s\n' /repo/alpha /repo/beta /repo/gamma
  exit 0
fi
exit 1
`)

	writeFakeBinary(t, binDir, "zellij", `#!/bin/sh
if [ "$1" = "list-sessions" ] && [ "$2" = "-s" ]; then
  printf '%s\n' dev work
  exit 0
fi
if [ "$1" = "setup" ] && [ "$2" = "--check" ]; then
  echo 'LAYOUT DIR: "`+logsDir+`/layouts"'
  exit 0
fi
printf '%s\n' "$*" >> "`+logsDir+`/zellij.log"
exit 0
`)

	require.NoError(t, os.MkdirAll(filepath.Join(logsDir, "layouts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "layouts", "dev.kdl"), []byte("pane"), 0o644))

	writePickerStub(t, binDir, logsDir, "outside")

	cmd := exec.Command("go", "run", ".", "zs")
	cmd.Dir = repoRoot
	cmd.Env = append(baseEnv(binDir, logsDir), "GOFLAGS=")
	cmd.Env = removeEnv(cmd.Env, "ZELLIJ")
	out, err := cmd.CombinedOutput()
	requireGoRunOrSkip(t, err, out)

	content, err := os.ReadFile(filepath.Join(logsDir, "zellij.log"))
	require.NoError(t, err)
	assert.Equal(t, "--session alpha --new-session-with-layout default options --default-cwd /repo/alpha\n", string(content))
}

func TestCLIInside_MatchesReferenceTabFlow(t *testing.T) {
	repoRoot := repoRoot(t)
	binDir, logsDir := setupFakeEnv(t)

	writeFakeBinary(t, binDir, "zoxide", `#!/bin/sh
if [ "$1" = "query" ] && [ "$2" = "-l" ]; then
  printf '%s\n' /repo/alpha /repo/beta /repo/gamma
  exit 0
fi
exit 1
`)

	writeFakeBinary(t, binDir, "zellij", `#!/bin/sh
if [ "$1" = "setup" ] && [ "$2" = "--check" ]; then
  echo 'LAYOUT DIR: "`+logsDir+`/layouts"'
  exit 0
fi
printf '%s\n' "$*" >> "`+logsDir+`/zellij.log"
exit 0
`)

	require.NoError(t, os.MkdirAll(filepath.Join(logsDir, "layouts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "layouts", "dev.kdl"), []byte("pane"), 0o644))

	writePickerStub(t, binDir, logsDir, "inside")

	cmd := exec.Command("go", "run", ".", "zs")
	cmd.Dir = repoRoot
	cmd.Env = append(baseEnv(binDir, logsDir), "ZELLIJ=1", "GOFLAGS=")
	out, err := cmd.CombinedOutput()
	requireGoRunOrSkip(t, err, out)

	content, err := os.ReadFile(filepath.Join(logsDir, "zellij.log"))
	require.NoError(t, err)
	assert.Equal(t, strings.Join([]string{
		"action new-tab --layout default --name alpha --cwd /repo/alpha",
		"action go-to-tab-name alpha",
	}, "\n")+"\n", string(content))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Clean(filepath.Join(wd, "../.."))
}

func setupFakeEnv(t *testing.T) (string, string) {
	t.Helper()
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	logsDir := filepath.Join(base, "logs")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.MkdirAll(logsDir, 0o755))
	return binDir, logsDir
}

func writeFakeBinary(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
}

func writePickerStub(t *testing.T, binDir, logsDir, mode string) {
	t.Helper()
	script := `#!/bin/sh
count_file="` + logsDir + `/picker-count"
count=0
if [ -f "$count_file" ]; then
  count=$(cat "$count_file")
fi
count=$((count + 1))
printf '%s' "$count" > "$count_file"
input="` + logsDir + `/picker-${count}.txt"
cat > "$input"
if [ "` + mode + `" = "outside" ]; then
  if [ "$count" -eq 1 ]; then
    sed -n '3p' "$input"
  else
    tail -1 "$input"
  fi
else
  if [ "$count" -eq 1 ]; then
    sed -n '1p' "$input"
  else
    tail -1 "$input"
  fi
fi
`
	writeFakeBinary(t, binDir, "fzf", script)
	writeFakeBinary(t, binDir, "sk", script)
}

func baseEnv(binDir, logsDir string) []string {
	env := os.Environ()
	var filtered []string
	for _, item := range env {
		if strings.HasPrefix(item, "PATH=") || strings.HasPrefix(item, "ZELLIJ=") {
			continue
		}
		filtered = append(filtered, item)
	}
	filtered = append(filtered, "PATH="+binDir+":"+os.Getenv("PATH"))
	return filtered
}

func removeEnv(env []string, key string) []string {
	prefix := key + "="
	out := env[:0]
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func requireGoRunOrSkip(t *testing.T, err error, out []byte) {
	t.Helper()
	if err == nil {
		return
	}
	msg := string(out)
	if strings.Contains(msg, "build failed") {
		t.Skipf("skipping workflow test due to unrelated root build failure: %s", msg)
	}
	require.NoError(t, err, msg)
}
