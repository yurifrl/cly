# OpenCode Notifications Integration - Draft

## Overview

Integrate OpenCode's plugin system with cly's existing notification infrastructure to send desktop notifications when OpenCode sessions become idle. This provides immediate feedback when AI tasks complete, similar to the existing Claude Code hook system.

## Goals

1. **Seamless Integration**: Leverage existing cly notification infrastructure (beeep, Zellij, sound system)
2. **Easy Installation**: Single command to install OpenCode plugin - no configuration steps
3. **Configurable Behavior**: Control message length, session filtering, and notification style via YAML
4. **No External Dependencies**: Embed plugin files in cly binary, no npm publishing required
5. **Consistent UX**: Follow existing cly notify patterns and command structure

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│ OpenCode Plugin (TypeScript)                                │
│ Location: ~/.config/opencode/plugin/cly-notify.ts           │
│                                                              │
│ ┌──────────────────────────────────────────────────────────┐│
│ │ 1. Listen to session.idle events                         ││
│ │ 2. Check if notifications enabled (env vars)             ││
│ │ 3. Filter subagent sessions (if configured)              ││
│ │ 4. Extract last assistant message                        ││
│ │ 5. Clean & truncate to max length                        ││
│ │ 6. Execute: cly notify hook opencode --message "{text}"  ││
│ └──────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ CLY Notify Module (Go)                                      │
│                                                              │
│ ┌──────────────────────────────────────────────────────────┐│
│ │ Hook Handler: cly notify hook opencode                   ││
│ │ - Read config from ~/.config/cly/config.yaml             ││
│ │ - Apply title, sound from config                         ││
│ │ - Send via beeep (desktop notification)                  ││
│ │ - Send via Zellij (if enabled)                           ││
│ └──────────────────────────────────────────────────────────┘│
│                                                              │
│ ┌──────────────────────────────────────────────────────────┐│
│ │ Installation: cly notify opencode install                ││
│ │ - Extract embedded TS plugin files                       ││
│ │ - Copy to ~/.config/opencode/plugin/                     ││
│ │ - Generate env var exports                               ││
│ └──────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```
OpenCode Session Idle Event
         │
         ├─> Fetch session.get to check for parentID (subagent check)
         ├─> Fetch session.messages via OpenCode API
         ├─> Extract last assistant message (text parts only)
         ├─> Clean: remove code blocks, inline code, normalize whitespace
         ├─> Truncate: to configured message_length chars
         │
         └─> Execute: cly notify hook opencode --message "{extracted_text}"
                   │
                   ├─> Read ~/.config/cly/config.yaml
                   ├─> Check if opencode hook enabled
                   ├─> Apply filter_subagents setting
                   ├─> Load opencode hook config (title, sound, etc)
                   ├─> Send desktop notification via beeep
                   └─> Send Zellij notification (if enabled)
```

## Implementation Plan

### Phase 1: TypeScript Plugin Files

**Files to Create:**
- `modules/notify/opencode/plugin/index.ts` - Plugin implementation

**Plugin Logic (index.ts):**

Note: Plugin only extracts message text. All configuration (enabled, filtering, truncation, sound) handled by cly.

```typescript
import type { Plugin } from "@opencode-ai/plugin"

export const ClyNotifyPlugin: Plugin = async ({ client, $ }) => {
  return {
    async event(input) {
      if (input.event.type === "session.idle") {
        const sessionID = input.event.properties.sessionID
        
        try {
          // Fetch session info
          const sessionResult = await client.session.get({ 
            path: { id: sessionID } 
          })

          // Fetch last message
          const response = await client.session.messages({ 
            path: { id: sessionID }
          })
          
          let messageText = "Task completed"
          
          if (response.data && response.data.length > 0) {
            const lastMessage = response.data[response.data.length - 1]
            
            // Only extract from assistant messages
            if (lastMessage.info.role === "assistant") {
              const parts = lastMessage.parts as any[]
              const textParts = parts.filter(p => p.type === "text")
              
              if (textParts.length > 0) {
                const lastText = textParts[textParts.length - 1].text
                
                // Clean text: remove code blocks, inline code, normalize spaces
                const cleanText = lastText
                  .replace(/```[\s\S]*?```/g, "")  // Remove code blocks
                  .replace(/`[^`]*`/g, "")         // Remove inline code
                  .replace(/\n/g, " ")             // Replace newlines with spaces
                  .replace(/\s+/g, " ")            // Collapse multiple spaces
                  .trim()
                
                // Use full cleaned text (truncation handled by cly)
                messageText = cleanText
              }
            }
          }

          // Call cly notify hook - it handles all config, filtering, truncation
          await $`cly notify hook opencode --message ${messageText}`.quiet()
          
        } catch (e) {
          // Fail silently - don't interrupt OpenCode on notification errors
        }
      }
    },
  }
}
```

**Key Design Decisions:**
- **Fail silently**: Notification failures don't interrupt OpenCode
- **Simple plugin logic**: Plugin only extracts message, cly handles all config/filtering/truncation
- **Clean text extraction**: Remove code blocks to keep notifications readable
- **No configuration in plugin**: All config read by cly from YAML

### Phase 2: Go Installation & Management

**Files to Create:**
- `modules/notify/opencode/install.go` - Plugin installation logic
- `modules/notify/opencode/cmd.go` - Commands (install, uninstall, status)

**Files to Modify:**
- `modules/notify/cmd.go` - Register opencode subcommand, add --message flag
- `modules/config/config.yaml` - Add opencode hook configuration
- `pkg/config/config.go` - Add FilterSubagents, MessageLength fields to HookConfig

#### install.go

```go
package opencode

import (
    "embed"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
)

//go:embed plugin/*
var pluginFiles embed.FS

// Install extracts embedded plugin files to OpenCode plugin directory
func Install() error {
    homeDir, _ := os.UserHomeDir()
    targetDir := filepath.Join(homeDir, ".config", "opencode", "plugin")
    
    // Create directory
    if err := os.MkdirAll(targetDir, 0755); err != nil {
        return fmt.Errorf("creating plugin dir: %w", err)
    }
    
    // Extract each plugin file
    entries, _ := pluginFiles.ReadDir("plugin")
    for _, entry := range entries {
        content, _ := pluginFiles.ReadFile("plugin/" + entry.Name())
        
        // Determine target filename
        var targetName string
        switch entry.Name() {
        case "index.ts":
            targetName = "cly-notify.ts"
        case "package.json", "tsconfig.json":
            continue // Skip metadata files - not needed for OpenCode plugin
        default:
            targetName = entry.Name()
        }
        
        targetPath := filepath.Join(targetDir, targetName)
        
        if err := os.WriteFile(targetPath, content, 0644); err != nil {
            return fmt.Errorf("writing %s: %w", entry.Name(), err)
        }
    }
    
    fmt.Println("✓ Plugin installed to:", targetDir)
    fmt.Println("✓ OpenCode will auto-load on next startup")
    fmt.Println()
    fmt.Println("Next steps:")
    fmt.Println("  1. Add env vars to your shell profile:")
    fmt.Println("       cly notify opencode env >> ~/.zshrc")
    fmt.Println("  2. Reload shell or run: source ~/.zshrc")
    fmt.Println("  3. Restart OpenCode")
    
    return nil
}

// Uninstall removes plugin from OpenCode plugin directory
func Uninstall() error {
    homeDir, _ := os.UserHomeDir()
    pluginPath := filepath.Join(homeDir, ".config", "opencode", "plugin", "cly-notify.ts")
    
    if err := os.Remove(pluginPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("removing plugin: %w", err)
    }
    
    fmt.Println("✓ Plugin uninstalled")
    fmt.Println()
    fmt.Println("Note: Remove env vars from shell profile if no longer needed")
    return nil
}

// Status checks plugin installation and environment
func Status() error {
    homeDir, _ := os.UserHomeDir()
    pluginPath := filepath.Join(homeDir, ".config", "opencode", "plugin", "cly-notify.ts")
    
    fmt.Println("OpenCode Plugin Status:")
    fmt.Println()
    
    // Check plugin file
    if _, err := os.Stat(pluginPath); err == nil {
        fmt.Println("✓ Plugin installed:", pluginPath)
    } else {
        fmt.Println("✗ Plugin not installed")
        fmt.Println("  Run: cly notify opencode install")
        return nil
    }
    
    // Check cly in PATH
    if clyPath, err := exec.LookPath("cly"); err == nil {
        fmt.Println("✓ cly binary found:", clyPath)
    } else {
        fmt.Println("⚠ cly binary not in PATH")
        fmt.Println("  Plugin may fail to execute cly commands")
    }
    
    // Check env vars
    fmt.Println()
    fmt.Println("Environment Variables:")
    envVars := []string{
        "CLY_NOTIFY_ENABLED",
        "CLY_NOTIFY_OPENCODE_ENABLED",
        "CLY_NOTIFY_OPENCODE_FILTER_SUBAGENTS",
        "CLY_NOTIFY_OPENCODE_MESSAGE_LENGTH",
        "CLY_PATH",
    }
    
    for _, key := range envVars {
        if val := os.Getenv(key); val != "" {
            fmt.Printf("  ✓ %s=%s\n", key, val)
        } else {
            fmt.Printf("  ✗ %s (not set)\n", key)
        }
    }
    
    fmt.Println()
    fmt.Println("To set env vars, run:")
    fmt.Println("  cly notify opencode env >> ~/.zshrc")
    
    return nil
}

// Env generates shell export statements for plugin configuration
func Env() error {
    cfg := pkgconfig.Get()
    notifyConfig := cfg.GetNotify()
    
    // Get opencode hook config
    hookConfig, ok := notifyConfig.Hooks["opencode"]
    if !ok {
        return fmt.Errorf("opencode hook not configured")
    }
    
    // Find cly binary path
    clyPath, _ := exec.LookPath("cly")
    if clyPath == "" {
        clyPath = "cly" // Fallback
    }
    
    // Generate exports
    fmt.Println("# CLY OpenCode Notification Plugin")
    fmt.Printf("export CLY_NOTIFY_ENABLED=%t\n", notifyConfig.Enabled)
    fmt.Printf("export CLY_NOTIFY_OPENCODE_ENABLED=%t\n", hookConfig.Enabled)
    fmt.Printf("export CLY_NOTIFY_OPENCODE_FILTER_SUBAGENTS=%t\n", hookConfig.FilterSubagents)
    fmt.Printf("export CLY_NOTIFY_OPENCODE_MESSAGE_LENGTH=%d\n", hookConfig.MessageLength)
    fmt.Printf("export CLY_PATH=\"%s\"\n", clyPath)
    
    return nil
}
```

**Key Design Decisions:**
- **Embedded files**: Use Go 1.16+ embed to include TS files in binary
- **Auto-load**: OpenCode auto-discovers plugins in `~/.config/opencode/plugin/`
- **Single .ts file**: Only copy `cly-notify.ts` (package.json/tsconfig.json not needed)
- **Env var generation**: `cly notify opencode env` outputs shell exports
- **Status command**: Comprehensive health check for troubleshooting

#### cmd.go

```go
package opencode

import (
    "github.com/spf13/cobra"
    pkgconfig "github.com/yurifrl/cly/pkg/config"
)

func Register(parent *cobra.Command) {
    cmd := &cobra.Command{
        Use:   "opencode",
        Short: "OpenCode plugin management",
        Long:  "Install and manage OpenCode notification plugin",
    }
    
    installCmd := &cobra.Command{
        Use:   "install",
        Short: "Install OpenCode notification plugin",
        Long:  "Extract plugin files to ~/.config/opencode/plugin/",
        RunE: func(cmd *cobra.Command, args []string) error {
            return Install()
        },
    }
    
    uninstallCmd := &cobra.Command{
        Use:   "uninstall",
        Short: "Uninstall OpenCode plugin",
        RunE: func(cmd *cobra.Command, args []string) error {
            return Uninstall()
        },
    }
    
    statusCmd := &cobra.Command{
        Use:   "status",
        Short: "Check plugin installation status",
        Long:  "Verify plugin installation and environment configuration",
        RunE: func(cmd *cobra.Command, args []string) error {
            return Status()
        },
    }
    
    envCmd := &cobra.Command{
        Use:   "env",
        Short: "Generate environment variable exports",
        Long:  "Print shell export statements for plugin configuration",
        RunE: func(cmd *cobra.Command, args []string) error {
            return Env()
        },
    }
    
    cmd.AddCommand(installCmd, uninstallCmd, statusCmd, envCmd)
    parent.AddCommand(cmd)
}
```

### Phase 3: Hook Handler Modifications

**Modify: modules/notify/cmd.go**

Add `--message` flag to hook command to support dynamic message injection:

```go
func createHookCmd() *cobra.Command {
    var messageOverride string  // NEW
    
    cmd := &cobra.Command{
        Use:   "hook <hookname>",
        Short: "Send notification for a hook",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            hookName := args[0]
            
            // ... existing checks ...
            
            // Get hook config
            hookConfig, ok := notifyConfig.Hooks[hookName]
            if !ok {
                return fmt.Errorf("hook '%s' not configured", hookName)
            }
            
            // Override message if flag provided (NEW)
            if messageOverride != "" {
                hookConfig.Message = messageOverride
            }
            
            // ... rest of existing logic ...
        },
    }
    
    // NEW: Add --message flag
    cmd.Flags().StringVar(&messageOverride, "message", "", "Override configured message")
    
    return cmd
}
```

**Register opencode subcommand:**

```go
// In Register() function, add:
import "github.com/yurifrl/cly/modules/notify/opencode"

func Register(parent *cobra.Command) {
    // ... existing code ...
    
    // Add opencode subcommand
    opencode.Register(notifyCmd)
}
```

### Phase 4: Configuration Structure

**Modify: modules/config/config.yaml**

Add opencode hook with new configuration fields:

```yaml
modules:
  notify:
    enabled: true
    sound: false
    use_zellij_status: true
    use_zellij_notify: true
    icon: ""
    hooks:
      notification:
        # ... existing notification hook ...
      stop:
        # ... existing stop hook ...
      opencode:                        # NEW
        enabled: true
        title: "✅ OpenCode Complete"
        message: "Task completed"      # Overridden by plugin --message flag
        sound: "Glass"
        filter_subagents: false        # NEW: notify for all sessions
        message_length: 200             # NEW: max chars to extract
```

**Modify: pkg/config/config.go**

Add new fields to HookConfig struct:

```go
type HookConfig struct {
    Enabled          bool   `yaml:"enabled" mapstructure:"enabled"`
    Title            string `yaml:"title" mapstructure:"title"`
    Message          string `yaml:"message" mapstructure:"message"`
    Sound            string `yaml:"sound" mapstructure:"sound"`
    ZellijStatus     string `yaml:"zellij_status" mapstructure:"zellij_status"`
    ZellijEvent      string `yaml:"zellij_event" mapstructure:"zellij_event"`
    FilterSubagents  bool   `yaml:"filter_subagents" mapstructure:"filter_subagents"`   // NEW
    MessageLength    int    `yaml:"message_length" mapstructure:"message_length"`        // NEW
}
```

Update `GetNotify()` to parse new fields:

```go
func (c *Config) GetNotify() NotifyConfig {
    // ... existing code ...
    
    for hookName, hookData := range hooks {
        if hookMap, ok := hookData.(map[string]interface{}); ok {
            hook := HookConfig{}
            // ... existing field parsing ...
            
            // NEW: Parse filter_subagents
            if filterSubagents, ok := hookMap["filter_subagents"].(bool); ok {
                hook.FilterSubagents = filterSubagents
            }
            
            // NEW: Parse message_length
            if messageLength, ok := hookMap["message_length"].(int); ok {
                hook.MessageLength = messageLength
            }
            
            notify.Hooks[hookName] = hook
        }
    }
}
```

Update `Env()` function in `opencode/install.go` to use new fields:

```go
fmt.Printf("export CLY_NOTIFY_OPENCODE_FILTER_SUBAGENTS=%t\n", hookConfig.FilterSubagents)
fmt.Printf("export CLY_NOTIFY_OPENCODE_MESSAGE_LENGTH=%d\n", hookConfig.MessageLength)
```

## Environment Variables

The plugin reads configuration from environment variables:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CLY_NOTIFY_ENABLED` | bool | `true` | Master enable for all notifications |
| `CLY_NOTIFY_OPENCODE_ENABLED` | bool | `true` | Enable OpenCode notifications |
| `CLY_NOTIFY_OPENCODE_FILTER_SUBAGENTS` | bool | `false` | Skip subagent session notifications |
| `CLY_NOTIFY_OPENCODE_MESSAGE_LENGTH` | int | `200` | Max characters to extract from message |
| `CLY_PATH` | string | `cly` | Path to cly binary |

**Why Environment Variables?**

OpenCode plugins have no direct filesystem access to read YAML config files. Environment variables provide a standard way for plugins to access configuration. The `cly notify opencode env` command generates these exports automatically from the YAML config.

## Installation & Usage

### 1. Install Plugin

```bash
cly notify opencode install
```

Output:
```
✓ Plugin installed to: /Users/username/.config/opencode/plugin
✓ OpenCode will auto-load on next startup

Next steps:
  1. Add env vars to your shell profile:
       cly notify opencode env >> ~/.zshrc
  2. Reload shell or run: source ~/.zshrc
  3. Restart OpenCode
```

### 2. Configure Environment

```bash
# Generate and append env vars to shell profile
cly notify opencode env >> ~/.zshrc

# Reload shell
source ~/.zshrc
```

Generated exports:
```bash
# CLY OpenCode Notification Plugin
export CLY_NOTIFY_ENABLED=true
export CLY_NOTIFY_OPENCODE_ENABLED=true
export CLY_NOTIFY_OPENCODE_FILTER_SUBAGENTS=false
export CLY_NOTIFY_OPENCODE_MESSAGE_LENGTH=200
export CLY_PATH="/usr/local/bin/cly"
```

### 3. Verify Installation

```bash
cly notify opencode status
```

Output:
```
OpenCode Plugin Status:

✓ Plugin installed: /Users/username/.config/opencode/plugin/cly-notify.ts
✓ cly binary found: /usr/local/bin/cly

Environment Variables:
  ✓ CLY_NOTIFY_ENABLED=true
  ✓ CLY_NOTIFY_OPENCODE_ENABLED=true
  ✓ CLY_NOTIFY_OPENCODE_FILTER_SUBAGENTS=false
  ✓ CLY_NOTIFY_OPENCODE_MESSAGE_LENGTH=200
  ✓ CLY_PATH=/usr/local/bin/cly
```

### 4. Test Manually

```bash
# Test notification directly
cly notify hook opencode --message "Test notification from OpenCode"
```

### 5. Use with OpenCode

1. Restart OpenCode (plugin auto-loads)
2. Start a coding task
3. Wait for session to become idle
4. Desktop notification appears with last assistant message

## Configuration Examples

### Notify for All Sessions (Default)

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: true
        title: "✅ OpenCode Complete"
        message: "Task completed"
        sound: "Glass"
        filter_subagents: false    # Notify for main + subagent sessions
        message_length: 200
```

### Notify Main Sessions Only

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: true
        title: "✅ Main Task Complete"
        message: "Task completed"
        sound: "Blow"
        filter_subagents: true     # Skip subagent sessions
        message_length: 150
```

### Long Messages

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: true
        title: "✅ OpenCode"
        message: "Task completed"
        sound: "Glass"
        filter_subagents: false
        message_length: 500        # Extract up to 500 chars
```

### Quiet Mode (No Notifications)

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: false             # Disable notifications
```

Or via environment:
```bash
export CLY_NOTIFY_OPENCODE_ENABLED=false
```

### Different Sound per Hook

```yaml
modules:
  notify:
    hooks:
      notification:
        sound: "Glass"             # Claude Code start
      stop:
        sound: "Blow"              # Claude Code complete
      opencode:
        sound: "Submarine"         # OpenCode complete
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `cly notify opencode install` | Install plugin to `~/.config/opencode/plugin/` |
| `cly notify opencode uninstall` | Remove plugin files |
| `cly notify opencode status` | Check installation and environment |
| `cly notify opencode env` | Generate shell export statements |
| `cly notify hook opencode --message "text"` | Send notification (called by plugin) |

## File Structure

```
cly/
├── modules/
│   └── notify/
│       ├── cmd.go                    # Modified: register opencode subcommand, add --message flag
│       ├── hooks.go                   # Existing: Claude Code hooks
│       ├── sound.go                   # Existing: sound management
│       ├── debug.go                   # Existing: debug commands
│       └── opencode/                  # NEW SUBMODULE
│           ├── cmd.go                 # Commands: install, uninstall, status, env
│           ├── install.go             # Installation logic
│           └── plugin/                # Embedded plugin files
│               ├── index.ts           # Plugin implementation
│               ├── package.json       # Metadata (not extracted)
│               └── tsconfig.json      # Metadata (not extracted)
├── pkg/
│   └── config/
│       └── config.go                  # Modified: add FilterSubagents, MessageLength
└── modules/config/
    └── config.yaml                    # Modified: add opencode hook
```

## Testing Plan (Manual)

Since no automated tests requested, here's the manual testing checklist:

### Installation Testing

1. **Fresh Install**
   ```bash
   cly notify opencode install
   ```
   - [ ] Plugin file created at `~/.config/opencode/plugin/cly-notify.ts`
   - [ ] Instructions printed for env var setup
   - [ ] No errors

2. **Status Check (Before Env)**
   ```bash
   cly notify opencode status
   ```
   - [ ] Shows plugin installed
   - [ ] Shows cly binary found
   - [ ] Shows env vars not set

3. **Env Generation**
   ```bash
   cly notify opencode env
   ```
   - [ ] Outputs valid bash export statements
   - [ ] Values match config.yaml

4. **Status Check (After Env)**
   ```bash
   source <(cly notify opencode env)
   cly notify opencode status
   ```
   - [ ] All env vars show as set
   - [ ] Values correct

5. **Uninstall**
   ```bash
   cly notify opencode uninstall
   cly notify opencode status
   ```
   - [ ] Plugin file removed
   - [ ] Status shows not installed

### Notification Testing

1. **Direct Hook Test**
   ```bash
   cly notify hook opencode --message "Test notification"
   ```
   - [ ] Desktop notification appears
   - [ ] Title matches config
   - [ ] Message shows "Test notification"
   - [ ] Sound plays (if enabled)
   - [ ] Zellij notification (if in Zellij)

2. **Long Message Test**
   ```bash
   cly notify hook opencode --message "$(python -c 'print("x" * 500)')"
   ```
   - [ ] Notification appears
   - [ ] Message truncated to configured length
   - [ ] Ellipsis added if truncated

3. **OpenCode Integration**
   - [ ] Restart OpenCode (plugin auto-loads)
   - [ ] Start task, wait for idle
   - [ ] Notification appears with extracted message
   - [ ] Message cleaned (no code blocks)
   - [ ] Message truncated to length

4. **Subagent Filtering**
   - [ ] Set `filter_subagents: true` in config
   - [ ] Regenerate env: `cly notify opencode env >> ~/.zshrc`
   - [ ] Restart OpenCode
   - [ ] Trigger subagent task
   - [ ] No notification for subagent
   - [ ] Notification for main session

### Configuration Testing

1. **Enable/Disable**
   - [ ] Set `enabled: false` in config
   - [ ] Regenerate env
   - [ ] No notifications appear

2. **Sound Variations**
   - [ ] Test with `sound: "Glass"`
   - [ ] Test with `sound: "Blow"`
   - [ ] Test with `sound: "Submarine"`
   - [ ] Test with sound globally disabled

3. **Message Length**
   - [ ] Set `message_length: 50`
   - [ ] Regenerate env
   - [ ] Long messages truncated to 50 chars

## Edge Cases & Error Handling

### Plugin Failures (Silent)

Plugin failures don't interrupt OpenCode:
- API call failures → silent fail
- cly binary not found → silent fail
- Message extraction errors → use default "Task completed"

### Configuration Errors

- Missing opencode hook in config → `cly notify opencode env` fails with clear error
- Invalid message_length (non-int) → defaults to 200
- Invalid filter_subagents (non-bool) → defaults to false

### File Permissions

- Plugin directory not writable → install fails with error
- OpenCode config not readable → plugin won't load (OpenCode handles this)

### Environment Issues

- CLY_PATH not set → plugin uses `cly` from PATH
- cly not in PATH → plugin fails silently on notification attempt
- Env vars not loaded → plugin checks fail early, no notifications sent

## Dependencies

### Go Dependencies (Existing)

All required dependencies already in project:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration
- `github.com/yurifrl/cly/pkg/config` - Config management
- `github.com/yurifrl/cly/pkg/notify` - Notification infrastructure
- `github.com/yurifrl/cly/pkg/style` - Terminal styling

### TypeScript Dependencies (Embedded)

Plugin requires (bundled by OpenCode):
- `@opencode-ai/plugin` - Plugin API types

No npm install required - OpenCode handles this automatically.

## Migration Path (None Required)

This is a new feature with no breaking changes:
- Existing notify hooks (notification, stop) unaffected
- Existing config structure compatible
- New fields have sensible defaults
- Users opt-in by running install command

## Rollback Plan

If issues arise:

1. **Uninstall plugin:**
   ```bash
   cly notify opencode uninstall
   ```

2. **Remove env vars:**
   Edit `~/.zshrc` and remove CLY_NOTIFY_* exports

3. **Reload shell:**
   ```bash
   source ~/.zshrc
   ```

4. **Restart OpenCode**

All existing functionality remains intact.

## Future Enhancements (Out of Scope)

Potential future improvements not included in this draft:

1. **Custom notification format templates**
   - Allow users to customize message format
   - Template variables: `{{session_id}}`, `{{duration}}`, etc.

2. **Multiple notification backends**
   - ntfy.sh integration
   - Slack/Discord webhooks
   - Email notifications

3. **Notification filtering rules**
   - Only notify for errors
   - Only notify after N minutes idle
   - Per-project notification settings

4. **Rich notification content**
   - Include file changes summary
   - Show git diff stats
   - Add action buttons (open in editor, etc.)

5. **Analytics/Logging**
   - Track notification delivery
   - Session duration metrics
   - Most active coding hours

## Success Criteria

This implementation is considered successful when:

1. ✅ Plugin installs with single command
2. ✅ Notifications work out-of-the-box after env setup
3. ✅ All configuration options respected
4. ✅ No OpenCode interruptions on errors
5. ✅ Existing notify infrastructure reused
6. ✅ Clear status/debugging commands available
7. ✅ Manual testing checklist passes

## Questions & Decisions Log

### Q1: Why environment variables instead of direct YAML reading?

**A:** OpenCode plugins run in Bun's sandbox with no filesystem access to user directories. Environment variables are the standard way for plugins to receive configuration. The `cly notify opencode env` command bridges this gap by generating exports from YAML.

### Q2: Why only copy index.ts and not package.json?

**A:** OpenCode auto-installs dependencies for plugins in `~/.config/opencode/plugin/`. The plugin only needs the .ts file - OpenCode handles the rest. package.json and tsconfig.json are metadata for development only.

### Q3: Why default to filter_subagents=false?

**A:** User requested "notify for everything" as default behavior. This provides maximum visibility into OpenCode activity. Advanced users can enable filtering if they find subagent notifications noisy.

### Q4: Why message_length default of 200 chars?

**A:** Based on typical desktop notification limits:
- macOS notifications: ~200-300 chars visible
- Linux notify-send: ~100-200 chars comfortable
- Windows toasts: ~150-200 chars

200 chars balances detail vs readability across platforms.

### Q5: Why fail silently in plugin?

**A:** Notifications are non-critical - OpenCode should never be interrupted because a notification failed. Silent failures ensure robust user experience. Users can debug with `cly notify opencode status` if notifications aren't appearing.

---

## Ready for Implementation

This draft provides:
- ✅ Complete architecture overview
- ✅ Detailed implementation plan (Phase 1-4)
- ✅ All file changes documented
- ✅ Configuration structure defined
- ✅ Installation workflow specified
- ✅ Testing plan (manual)
- ✅ Edge cases considered
- ✅ Success criteria defined

**Next Steps:**
1. Review this draft
2. Clarify any questions or concerns
3. Adjust configuration defaults if needed
4. Proceed with implementation following this plan
