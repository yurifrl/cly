# Dotfiles Symlink Manager

Manage dotfile symlinks from a declarative config file.

---

## What It Does

**Symlink Management** - Create/update symlinks from source to destination

**Install Commands** - Execute shell commands during install (`-i` flag)

**Conflict Detection** - Error on existing files, replace symlinks

---

## Usage Examples

### Basic sync
```bash
cly dotfiles

# With install (also runs ! commands)
cly dotfiles -i
```

**What happens:**
- Reads `dotfiles.conf` from configured directory
- Creates symlinks: `source -> destination`
- Skips `!` commands unless `-i` flag
- Removes existing symlinks before recreating

### Check status
```bash
cly dotfiles status
```

**What happens:**
- Shows each mapping and its current state
- Marks: linked, missing, conflict, broken

```
~/.gitconfig        ✓ linked
~/.config/nvim/     ✓ linked
~/.config/fish/     ✗ conflict (file exists)
~/.tool-versions    ○ missing source
```

### Dry run
```bash
cly dotfiles sync --dry-run
```

**What happens:**
- Shows what would happen
- No actual changes

---

## Features

### Config File Format

**Location:** `~/.config/cly/dotfiles.conf` (or `--config` flag)

**Format:**
```conf
# Comments start with #

# Files: source -> destination
./home/.gitconfig -> ~/.gitconfig

# Directories: trailing slash required
./home/.config/nvim/ -> ~/.config/nvim/

# Init commands: prefix with !
!launchctl load ~/Library/LaunchAgents/foo.plist

# Special functions
!zellij_plugin https://github.com/user/repo
```

### Symlink Operations

**Creates parent directories** if they don't exist

**Handles conflicts:**
- Existing symlink: removes and recreates
- Existing file/dir: errors (user must resolve)

**Validates:**
- Source exists
- Directory entries have trailing `/`
- File entries don't point to directories

### Install Commands

Lines starting with `!` are install commands - only run with `-i` flag.

**Special functions available:**
- `zellij_plugin <github_url>` - downloads `.wasm` from latest release

### Status Display

```
cly dotfiles status

Dotfiles: /path/to/dotfiles.conf

Mappings:
  ~/.gitconfig           ✓ linked → ./home/.gitconfig
  ~/.config/nvim/        ✓ linked → ./home/.config/nvim/
  ~/.config/fish/        ✗ conflict (regular file exists)
  ~/.tool-versions       ○ source missing

Init commands: 3 (use --init to execute)
```

---

## Error Handling

**Missing config:**
```bash
cly dotfiles sync
# ❌ Config not found: ~/.config/cly/dotfiles.conf
#    Create it or use --config /path/to/dotfiles.conf
```

**Invalid format:**
```bash
# Config line: ./foo ->
# ❌ Line 5: Invalid format
#    Expected: source -> destination
```

**Source missing:**
```bash
# ⚠️  Skipping: ./missing -> ~/.dest
#    Source does not exist
```

**Directory mismatch:**
```bash
# ❌ ./config/nvim is a directory but missing trailing /
#    Use: ./config/nvim/ -> ~/.config/nvim/
```

---

## Testing Strategy

**Integration Tests:**
- Create temp directories with test configs
- Run actual commands via CLI
- Verify symlinks created correctly
- Verify backups created on conflict
- Test dry-run produces no changes

**Test Coverage:**
- [ ] Basic symlink creation (file and directory)
- [ ] Error on existing files (not symlinks)
- [ ] Replace existing symlinks
- [ ] Skip missing sources with warning
- [ ] Install commands (`!`) only run with `-i`
- [ ] Dry run shows plan without changes
- [ ] Status shows correct states
- [ ] Invalid config lines produce errors
- [ ] Custom config path via `--config`

---

## Config Resolution

**Default directory:** has a default value, configurable in cly config

**Config file:** `<dotfiles_dir>/dotfiles.conf`

**Override:** `--config /path/to/dotfiles.conf`

**Relative paths in config:** resolved relative to dotfiles directory

---

## Open Questions

- [ ] Keep `zellij_plugin` function or make it generic (`github_release_download`)?
  - generic
- [ ] Add `unlink` subcommand to remove all managed symlinks?
  - yes
- [ ] Add `diff` subcommand to show what changed since last sync?
  - no it would add complexity
