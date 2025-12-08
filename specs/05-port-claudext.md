# Port claudext to CLY

**Goal**: Port the Fish shell `claudext` function to a Go CLI command in CLY

**Source**: `~/DotFiles/home/.config/fish/functions/claudext.fish`

---

## What is claudext?

A Fish shell wrapper around the `claude` CLI that adds:
- **Workspace management** - Auto-navigate to work directories
- **Session naming** - Named sessions with random generation
- **Session persistence** - Save/resume sessions by name
- **Archive management** - Archive completed work by date
- **Tab integration** - Auto-rename Zellij tabs

---

## Core Features to Port

### 1. Workspace Navigation
**Fish commands:**
- `-w, --workdir` - Go to newest folder in ~/Workdir/Wip
- `-wd, --cd-work-dir` - Go to ~/Workdir/Wip root
- `-g, --goto DIR` - Go to specific folder

**CLY implementation:**
```bash
cly claude workdir              # Go to newest
cly claude goto <folder>        # Go to specific
cly claude new [--name NAME]    # Create new folder
```

### 2. Session Management
**Fish features:**
- Named sessions (random or user-specified)
- Session persistence in `~/.config/claudext.json`
- Resume by name with `-us, --ultra-switch`
- Resume from .paid file with `-rf, --resume-from`

**CLY implementation:**
```bash
cly claude start [--name NAME]           # New named session
cly claude resume <name>                 # Resume by name
cly claude resume-file <file.paid>       # Resume from file
cly claude save [description]            # Save current session
```

### 3. Archive Management
**Fish command:**
- `-a, --archive` - Move folders with today's timestamp to ~/.AIDump

**CLY implementation:**
```bash
cly claude archive              # Archive today's folders
cly claude archive --date YYMMDD  # Archive specific date
```

### 4. Random Name Generation
**Fish logic:**
- 2-word names: ColorAnimal (e.g., "RedWolf", "BlueFox")
- Word lists: colors (13), animals (16)
- Used for folder names and session names

**CLY implementation:**
```go
// pkg/names/generator.go
func GenerateRandomName() string {
    colors := []string{"red", "blue", "green", ...}
    animals := []string{"cat", "dog", "fox", ...}
    color := capitalize(randomChoice(colors))
    animal := capitalize(randomChoice(animals))
    return color + animal
}
```

---

## Architecture Design

### Module Structure
```
modules/claude/
├── cmd.go              # Main command registration
├── workdir.go          # Workspace navigation
├── session.go          # Session management
├── archive.go          # Archive functionality
├── config.go           # Config file handling (~/.config/claudext.json)
└── names.go            # Random name generation
```

### Configuration File
**Location**: `~/.config/cly/claude.json`

**Format**:
```json
{
  "workdir": "~/Workdir/Wip",
  "archive_dir": "~/.AIDump",
  "sessions": {
    "RedWolf": "session-id-abc123",
    "BlueFox": "session-id-def456"
  }
}
```

### Command Structure
```
cly claude
├── workdir              # Navigate to newest work folder
├── goto <folder>        # Go to specific folder
├── new [--name NAME]    # Create new work folder
├── start [--name NAME]  # Start named Claude session
├── resume <name>        # Resume by session name
├── resume-file <file>   # Resume from .paid file
├── save [desc]          # Save current session
├── archive [--date]     # Archive completed work
└── list                 # List saved sessions
```

---

## Implementation Phases

### Phase 1: Basic Workspace Management
**Deliverables:**
- `cly claude new` - Create timestamped folders
- `cly claude workdir` - Navigate to newest
- `cly claude goto <folder>` - Navigate to specific

**Files:**
```go
modules/claude/cmd.go       // Main command + subcommands
modules/claude/workdir.go   // Directory management
modules/claude/names.go     // Random name generation
```

### Phase 2: Session Management
**Deliverables:**
- `cly claude start --name NAME` - Named sessions
- `cly claude save` - Save session to config
- `cly claude list` - List saved sessions

**Files:**
```go
modules/claude/session.go   // Session CRUD
modules/claude/config.go    // JSON config file handling
```

### Phase 3: Session Resume
**Deliverables:**
- `cly claude resume <name>` - Resume by name
- `cly claude resume-file <file.paid>` - Resume from file

**Integration:**
- Parse .paid filename for session ID
- Look up session ID from config
- Call `claude --resume <id>`

### Phase 4: Archive Management
**Deliverables:**
- `cly claude archive` - Archive today's folders
- Move to `~/.AIDump/{date}/`

**Files:**
```go
modules/claude/archive.go   // Archive logic
```

---

## Key Implementation Details

### 1. Random Name Generation
```go
// modules/claude/names.go
package claude

import (
    "math/rand"
    "strings"
)

var (
    colors = []string{"red", "blue", "green", "yellow", "purple",
                      "orange", "pink", "cyan", "magenta", "lime",
                      "teal", "navy", "maroon", "olive"}
    animals = []string{"cat", "dog", "fox", "wolf", "bear", "lion",
                       "tiger", "shark", "eagle", "hawk", "dove",
                       "owl", "rabbit", "deer", "mouse", "rat"}
)

func GenerateRandomName() string {
    color := capitalize(colors[rand.Intn(len(colors))])
    animal := capitalize(animals[rand.Intn(len(animals))])
    return color + animal
}

func capitalize(s string) string {
    if len(s) == 0 {
        return s
    }
    return strings.ToUpper(s[:1]) + s[1:]
}
```

### 2. Config File Management
```go
// modules/claude/config.go
package claude

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Config struct {
    WorkDir    string            `json:"workdir"`
    ArchiveDir string            `json:"archive_dir"`
    Sessions   map[string]string `json:"sessions"`
}

func LoadConfig() (*Config, error) {
    path := filepath.Join(os.Getenv("HOME"), ".config/cly/claude.json")
    // Load or create default
}

func (c *Config) SaveSession(name, sessionID string) error {
    c.Sessions[name] = sessionID
    return c.Save()
}

func (c *Config) GetSession(name string) (string, bool) {
    id, ok := c.Sessions[name]
    return id, ok
}
```

### 3. Workspace Management
```go
// modules/claude/workdir.go
package claude

import (
    "os"
    "path/filepath"
    "time"
)

func FindNewestFolder(baseDir string) (string, error) {
    // Walk directories, find most recent modification
}

func CreateWorkFolder(baseDir, name string) (string, error) {
    timestamp := time.Now().Format("060102") // YYMMDD
    if name == "" {
        name = GenerateRandomName()
    }
    folderName := timestamp + "-" + strings.ToLower(name)
    fullPath := filepath.Join(baseDir, folderName)
    // Create folder + CLAUDE.md + README.md
    return fullPath, os.MkdirAll(fullPath, 0755)
}
```

### 4. Calling Claude CLI
```go
// Execute claude command
func runClaude(args []string, workdir string) error {
    cmd := exec.Command("claude", args...)
    cmd.Dir = workdir
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

---

## Feature Mapping

### Fish → Go Command Mapping

| Fish claudext | CLY equivalent | Notes |
|---------------|----------------|-------|
| `claudext -w` | `cly claude workdir` | Navigate to newest folder |
| `claudext -wd` | `cly claude workdir --root` | Go to Wip root |
| `claudext -n --name Foo` | `cly claude new --name Foo` | Create new folder |
| `claudext -g myproject` | `cly claude goto myproject` | Go to specific folder |
| `claudext -a` | `cly claude archive` | Archive today's work |
| `claudext --save-session "desc"` | `cly claude save "desc"` | Save session |
| `claudext -us RedWolf` | `cly claude resume RedWolf` | Resume by name |
| `claudext -rf file.paid` | `cly claude resume-file file.paid` | Resume from file |
| `claudext --name Foo` | `cly claude start --name Foo` | Named session |

### Dependencies Removed in Go Version

**Fish-specific**:
- Zellij tab renaming (optional, can shell out if available)
- `glow` for markdown rendering (optional, can use lipgloss)
- Fish string functions (Go has strings package)

**Keep**:
- Session persistence (~/.config/cly/claude.json)
- Folder timestamp format (YYMMDD)
- Random name generation (ColorAnimal)
- Archive structure (~/.AIDump/{date}/)

---

## Open Questions

1. **Should we integrate with Zellij?**
   - Option A: Shell out to `zellij action rename-tab` if available
   - Option B: Skip (not essential)
   - Recommendation: Optional integration

2. **How to handle `claude` CLI dependency?**
   - Option A: Require `claude` in PATH
   - Option B: Make it configurable
   - Recommendation: Check PATH, error if not found

3. **Config file location?**
   - Fish uses: `~/.config/claudext.json`
   - CLY convention: `~/.config/cly/claude.json`
   - Recommendation: Use CLY convention

4. **Should this be a utility or demo?**
   - Answer: **Utility** (real functionality)
   - Location: `modules/claude/` (not modules/demo/)

---

## Migration Path for Users

### For existing claudext users:

1. **Migrate sessions**:
```bash
# Copy sessions to new location
cp ~/.config/claudext.json ~/.config/cly/claude.json
```

2. **Update aliases**:
```fish
# Old
alias cx='claudext'

# New
alias cx='cly claude'
```

3. **Feature parity**:
- All flags work the same
- Session names preserved
- Folder structure unchanged

---

## Success Criteria

- [ ] `cly claude new` creates timestamped folders with CLAUDE.md
- [ ] `cly claude workdir` navigates to newest folder
- [ ] `cly claude start --name Foo` creates named session
- [ ] `cly claude save` persists session to config
- [ ] `cly claude resume Foo` restores session by name
- [ ] `cly claude archive` moves folders to ~/.AIDump
- [ ] Config saved to ~/.config/cly/claude.json
- [ ] Random names generate (ColorAnimal format)

---

## Future Enhancements

### Phase 5: TUI Interface
Replace shell wrapper with interactive TUI:
- Bubbles list for session selection
- Bubbles table for workspace browsing
- Bubbles textarea for session descriptions

### Phase 6: Session Analytics
- Track session duration
- Count prompts per session
- Show session history

### Phase 7: Multi-Provider
Extend beyond Claude CLI to support:
- OpenAI CLI
- Anthropic API direct
- Custom LLM endpoints

---

## Notes

- **540+ lines** of Fish → ~300 lines of Go (estimated)
- Main complexity: Session JSON management, folder scanning
- Benefits of Go: Cross-platform (works without Fish shell), faster, single binary
- This would be CLY's second utility module (after UUID)
