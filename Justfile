# Justfile for cly project

# Variables
version := `cat VERSION`
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
    go build -ldflags="-s -w -X github.com/yurifrl/cly/cmd.Version={{version}} -X github.com/yurifrl/cly/cmd.BuildTime={{build_time}}" -o ~/.local/bin/{{binary_name}} .
    cly completion fish install

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
    @echo "v{{version}}"

# Bump version tasks
bump-patch:
    #!/usr/bin/env bash
    current=$(cat VERSION)
    new=$(echo $current | awk -F. '{printf "%d.%d.%d\n", $1, $2, $3+1}')
    echo $new > VERSION
    echo "Version bumped to $new"

bump-minor:
    #!/usr/bin/env bash
    current=$(cat VERSION)
    new=$(echo $current | awk -F. '{printf "%d.%d.%d\n", $1, $2+1, 0}')
    echo $new > VERSION
    echo "Version bumped to $new"

bump-major:
    #!/usr/bin/env bash
    current=$(cat VERSION)
    new=$(echo $current | awk -F. '{printf "%d.%d.%d\n", $1+1, 0, 0}')
    echo $new > VERSION
    echo "Version bumped to $new"

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