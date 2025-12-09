# Tasks: Add Dotfiles Command

## Phase 1: Core Infrastructure

- [x] Create `modules/dotfiles/` directory structure
- [x] Add config parser for `dotfiles.conf` format (source -> destination)
- [x] Add config key `modules.dotfiles.directory` with default value
- [x] Write tests for config parsing (valid lines, comments, empty lines, invalid format)

## Phase 2: Symlink Operations

- [x] Implement symlink creation (files and directories)
- [x] Handle existing symlinks (remove and recreate)
- [x] Error on existing files/dirs (not symlinks)
- [x] Create parent directories as needed
- [x] Validate trailing slash for directories
- [x] Write tests for all symlink operations

## Phase 3: Main Command

- [x] Register `cly dotfiles` command via Cobra
- [x] Implement sync logic (read config, process mappings)
- [x] Add `-i` flag for install commands
- [x] Skip `!` lines unless `-i` flag provided
- [x] Write integration tests for command execution

## Phase 4: Install Commands

- [x] Parse `!` prefixed lines as shell commands
- [x] Implement generic `github_release_download` function (stub - returns error)
- [x] Execute install commands only with `-i` flag
- [x] Write tests for install command execution

## Phase 5: Status Subcommand

- [x] Implement `cly dotfiles status` subcommand
- [x] Show each mapping with state (linked, missing, conflict, broken)
- [x] Display install command count
- [x] Write tests for status output

## Phase 6: Unlink Subcommand

- [x] Implement `cly dotfiles unlink` subcommand
- [x] Remove all symlinks managed by config
- [x] Skip non-symlink destinations
- [x] Write tests for unlink operation

## Validation

- [x] All tests pass
- [x] Manual test with real dotfiles.conf
- [ ] Update documentation (deferred - no docs requested)
