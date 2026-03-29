# Justfile for cly project

# Variables
version := `git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo dev`
build_time := `date -u +%Y%m%d%H%M%S`
binary_name := "cly"
build_dir := "dist"

# Default task - show available recipes
default:
    @just --list

# Build tasks
build: _ensure_build_dir
    go build -ldflags="-s -w -X github.com/yurifrl/cly/cmd.Version={{version}} -X github.com/yurifrl/cly/cmd.BuildTime={{build_time}}" -o {{build_dir}}/{{binary_name}} .

build-mcp: _ensure_build_dir
    go build -ldflags="-s -w -X main.version={{version}}" -o {{build_dir}}/mcp ./cmd/mcp

build-all: build build-mcp

# Install tasks
alias i := install

install:
    go build -ldflags="-s -w -X github.com/yurifrl/cly/cmd.Version={{version}} -X github.com/yurifrl/cly/cmd.BuildTime={{build_time}}" -o /tmp/cly-bootstrap .
    /tmp/cly-bootstrap update

install-mcp: _ensure_build_dir
    go build -ldflags="-s -w -X main.version={{version}}" -o {{build_dir}}/mcp ./cmd/mcp
    sudo mv {{build_dir}}/mcp /usr/local/bin/mcp
    go run ./cmd/mcp completion fish > ~/.cache/fish_completions/mcp.fish

install-all: install install-mcp

# Test tasks
test:
    go test ./...

test-mcp *args:
    go test ./modules/mcp/... -v {{args}}

# Run tasks
run *args:
    go run . {{args}}

run-mcp *args:
    go run ./cmd/mcp {{args}}

# Version management
version-show:
    @git describe --tags --abbrev=0

# Bump version tasks
bump-patch:
    #!/usr/bin/env bash
    current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
    new=$(echo $current | awk -F. '{printf "%d.%d.%d\n", $1, $2, $3+1}')
    git tag -a "v$new" -m "Release v$new"
    echo "Version bumped to v$new"

bump-minor:
    #!/usr/bin/env bash
    current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
    new=$(echo $current | awk -F. '{printf "%d.%d.%d\n", $1, $2+1, 0}')
    git tag -a "v$new" -m "Release v$new"
    echo "Version bumped to v$new"

bump-major:
    #!/usr/bin/env bash
    current=$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
    new=$(echo $current | awk -F. '{printf "%d.%d.%d\n", $1+1, 0, 0}')
    git tag -a "v$new" -m "Release v$new"
    echo "Version bumped to v$new"

# Git worktree management
worktree name:
    git worktree add .worktrees/{{name}} -b {{name}}

# Reference management (submodules)
ref-init:
    git submodule update --init --recursive

ref-update:
    git submodule update --remote --merge

# Helper tasks (private - prefixed with _)
_ensure_build_dir:
    @mkdir -p {{build_dir}}