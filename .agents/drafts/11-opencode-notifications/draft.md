# OpenCode Notifications Integration - Draft v2

## Overview

Integrate OpenCode's plugin system with cly's existing notification infrastructure to send desktop notifications when OpenCode sessions become idle. This provides immediate feedback when AI tasks complete, similar to the existing Claude Code hook system.

## Goals

1. **Seamless Integration**: Leverage existing cly notification infrastructure (beeep, Zellij, sound system)
2. **Zero Configuration**: Single command install - no shell profile editing required
3. **Configurable Behavior**: Control message length and notification style via YAML
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
│ │ 2. Extract last assistant message                        ││
│ │ 3. Clean text (remove code blocks, normalize spaces)     ││
│ │ 4. Execute: cly notify hook opencode --message "{text}"  ││
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
│ │ - Check if enabled                                       ││
│ │ - Truncate message to configured length                  ││
│ │ - Apply title, sound from config                         ││
│ │ - Send via beeep (desktop notification)                  ││
│ │ - Send via Zellij (if enabled)                           ││
│ └──────────────────────────────────────────────────────────┘│
│                                                              │
│ ┌──────────────────────────────────────────────────────────┐│
│ │ Installation: cly notify opencode install                ││
│ │ - Extract embedded TS plugin file                        ││
│ │ - Copy to ~/.config/opencode/plugin/                     ││
│ └──────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

```
OpenCode Session Idle Event
         │
         ├─> Fetch session.messages via OpenCode API
         ├─> Extract last assistant message (text parts only)
         ├─> Clean: remove code blocks, inline code, normalize whitespace
         │
         └─> Execute: cly notify hook opencode --message "{extracted_text}"
                   │
                   ├─> Read ~/.config/cly/config.yaml
                   ├─> Check if opencode hook enabled
                   ├─> Truncate message to configured message_length
                   ├─> Load opencode hook config (title, sound, etc)
                   ├─> Send desktop notification via beeep
                   └─> Send Zellij notification (if enabled)
```

## Implementation Plan

### Phase 1: TypeScript Plugin

**File to Create:**
- `modules/notify/opencode/plugin/index.ts` - Plugin implementation

**Plugin Logic:**

```typescript
import type { Plugin } from "@opencode-ai/plugin"

export const ClyNotifyPlugin: Plugin = async ({ client, $ }) => {
  return {
    async event(input) {
      if (input.event.type === "session.idle") {
        const sessionID = input.event.properties.sessionID
        
        try {
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
- **Simple plugin logic**: Plugin only extracts and cleans message text
- **All config in cly**: enabled/disabled, truncation, sound all handled by cly
- **Clean text extraction**: Remove code blocks to keep notifications readable

### Phase 2: Go Installation & Management

**Files to Create:**
- `modules/notify/opencode/install.go` - Plugin installation logic
- `modules/notify/opencode/cmd.go` - Commands (install, uninstall, status)

**Files to Modify:**
- `modules/notify/cmd.go` - Register opencode subcommand, add --message flag with truncation
- `modules/config/config.yaml` - Add opencode hook configuration
- `pkg/config/config.go` - Add `MessageLength` field to HookConfig

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

// Install extracts embedded plugin file to OpenCode plugin directory
func Install() error {
    homeDir, _ := os.UserHomeDir()
    targetDir := filepath.Join(homeDir, ".config", "opencode", "plugin")
    
    // Create directory
    if err := os.MkdirAll(targetDir, 0755); err != nil {
        return fmt.Errorf("creating plugin dir: %w", err)
    }
    
    // Extract plugin file
    content, err := pluginFiles.ReadFile("plugin/index.ts")
    if err != nil {
        return fmt.Errorf("reading plugin file: %w", err)
    }
    
    targetPath := filepath.Join(targetDir, "cly-notify.ts")
    if err := os.WriteFile(targetPath, content, 0644); err != nil {
        return fmt.Errorf("writing plugin: %w", err)
    }
    
    fmt.Println("✓ Plugin installed to:", targetDir)
    fmt.Println("✓ OpenCode will auto-load on next startup")
    fmt.Println()
    fmt.Println("Next steps:")
    fmt.Println("  1. Restart OpenCode")
    fmt.Println("  2. Notifications will appear when tasks complete")
    
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
    return nil
}

// Status checks plugin installation
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
    
    return nil
}
```

**Key Design Decisions:**
- **Embedded files**: Use Go 1.16+ embed to include TS file in binary
- **Auto-load**: OpenCode auto-discovers plugins in `~/.config/opencode/plugin/`
- **Single .ts file**: Only copy `cly-notify.ts` (no package.json/tsconfig needed)
- **Status command**: Basic health check for troubleshooting

#### cmd.go

```go
package opencode

import (
    "github.com/spf13/cobra"
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
        Long:  "Extract plugin file to ~/.config/opencode/plugin/",
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
    
    cmd.AddCommand(installCmd, uninstallCmd, statusCmd)
    parent.AddCommand(cmd)
}
```

### Phase 3: Hook Handler Modifications

**Modify: modules/notify/cmd.go**

Add `--message` flag to hook command and implement truncation:

```go
func createHookCmd() *cobra.Command {
    var messageOverride string  // NEW
    
    cmd := &cobra.Command{
        Use:   "hook <hookname>",
        Short: "Send notification for a hook",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            hookName := args[0]
            
            // Check CLAUDE_VERBOSE env (default 1)
            if os.Getenv("CLAUDE_VERBOSE") == "0" {
                return nil // Silent mode
            }
            
            // Load config
            cfg := pkgconfig.Get()
            if cfg == nil {
                return fmt.Errorf("failed to load config")
            }
            
            notifyConfig := cfg.GetNotify()
            
            // Check if notifications are enabled
            if !notifyConfig.Enabled {
                return nil
            }
            
            // Get hook config
            hookConfig, ok := notifyConfig.Hooks[hookName]
            if !ok {
                return fmt.Errorf("hook '%s' not configured", hookName)
            }
            
            // Check if hook is enabled
            if !hookConfig.Enabled {
                return nil
            }
            
            // Override message if flag provided (NEW)
            if messageOverride != "" {
                hookConfig.Message = messageOverride
                
                // Truncate to configured length (NEW)
                if hookConfig.MessageLength > 0 && len(hookConfig.Message) > hookConfig.MessageLength {
                    hookConfig.Message = hookConfig.Message[:hookConfig.MessageLength] + "..."
                }
            }
            
            // ... rest of existing logic (generate group, send notification) ...
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

Add opencode hook with configuration:

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
        message_length: 200             # NEW: max chars before truncation
```

**Modify: pkg/config/config.go**

Add new field to HookConfig struct:

```go
type HookConfig struct {
    Enabled          bool   `yaml:"enabled" mapstructure:"enabled"`
    Title            string `yaml:"title" mapstructure:"title"`
    Message          string `yaml:"message" mapstructure:"message"`
    Sound            string `yaml:"sound" mapstructure:"sound"`
    ZellijStatus     string `yaml:"zellij_status" mapstructure:"zellij_status"`
    ZellijEvent      string `yaml:"zellij_event" mapstructure:"zellij_event"`
    MessageLength    int    `yaml:"message_length" mapstructure:"message_length"`  // NEW
}
```

Update `GetNotify()` to parse new field:

```go
func (c *Config) GetNotify() NotifyConfig {
    // ... existing code ...
    
    for hookName, hookData := range hooks {
        if hookMap, ok := hookData.(map[string]interface{}); ok {
            hook := HookConfig{}
            // ... existing field parsing ...
            
            // NEW: Parse message_length
            if messageLength, ok := hookMap["message_length"].(int); ok {
                hook.MessageLength = messageLength
            }
            
            notify.Hooks[hookName] = hook
        }
    }
}
```

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
  1. Restart OpenCode
  2. Notifications will appear when tasks complete
```

### 2. Verify Installation

```bash
cly notify opencode status
```

Output:
```
OpenCode Plugin Status:

✓ Plugin installed: /Users/username/.config/opencode/plugin/cly-notify.ts
✓ cly binary found: /usr/local/bin/cly
```

### 3. Test Manually

```bash
# Test notification directly
cly notify hook opencode --message "Test notification from OpenCode"
```

### 4. Use with OpenCode

1. Restart OpenCode (plugin auto-loads)
2. Start a coding task
3. Wait for session to become idle
4. Desktop notification appears with last assistant message

## Configuration Examples

### Default Configuration

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: true
        title: "✅ OpenCode Complete"
        message: "Task completed"
        sound: "Glass"
        message_length: 200      # Truncate messages to 200 chars
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
        message_length: 500      # Extract up to 500 chars
```

### Quiet Mode (No Notifications)

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: false           # Disable notifications
```

### Different Sound

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: true
        title: "✅ OpenCode"
        message: "Task completed"
        sound: "Submarine"       # Available: Glass, Blow, Submarine, Ping, etc.
        message_length: 200
```

### No Message Truncation

```yaml
modules:
  notify:
    hooks:
      opencode:
        enabled: true
        title: "✅ OpenCode"
        message: "Task completed"
        sound: "Glass"
        message_length: 0        # 0 = no truncation
```

## Commands Reference

| Command | Description |
|---------|-------------|
| `cly notify opencode install` | Install plugin to `~/.config/opencode/plugin/` |
| `cly notify opencode uninstall` | Remove plugin files |
| `cly notify opencode status` | Check installation status |
| `cly notify hook opencode --message "text"` | Send notification (called by plugin) |

## File Structure

```
cly/
├── modules/
│   └── notify/
│       ├── cmd.go                    # Modified: register opencode, add --message flag
│       ├── hooks.go                   # Existing: Claude Code hooks
│       ├── sound.go                   # Existing: sound management
│       ├── debug.go                   # Existing: debug commands
│       └── opencode/                  # NEW SUBMODULE
│           ├── cmd.go                 # Commands: install, uninstall, status
│           ├── install.go             # Installation logic
│           └── plugin/                # Embedded plugin file
│               └── index.ts           # Plugin implementation
├── pkg/
│   └── config/
│       └── config.go                  # Modified: add MessageLength
└── modules/config/
    └── config.yaml                    # Modified: add opencode hook
```

## Testing Plan (Manual)

### Installation Testing

1. **Fresh Install**
   ```bash
   cly notify opencode install
   ```
   - [ ] Plugin file created at `~/.config/opencode/plugin/cly-notify.ts`
   - [ ] Instructions printed
   - [ ] No errors

2. **Status Check**
   ```bash
   cly notify opencode status
   ```
   - [ ] Shows plugin installed
   - [ ] Shows cly binary found

3. **Uninstall**
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

### Configuration Testing

1. **Enable/Disable**
   - [ ] Set `enabled: false` in config
   - [ ] No notifications appear

2. **Sound Variations**
   - [ ] Test with `sound: "Glass"`
   - [ ] Test with `sound: "Blow"`
   - [ ] Test with `sound: "Submarine"`
   - [ ] Test with sound globally disabled

3. **Message Length**
   - [ ] Set `message_length: 50`
   - [ ] Long messages truncated to 50 chars
   - [ ] Set `message_length: 0`
   - [ ] No truncation applied

## Edge Cases & Error Handling

### Plugin Failures (Silent)

Plugin failures don't interrupt OpenCode:
- API call failures → silent fail
- cly binary not found → silent fail
- Message extraction errors → use default "Task completed"

### Configuration Errors

- Missing opencode hook in config → hook command fails with clear error
- Invalid message_length (non-int) → defaults to 0 (no truncation)

### File Permissions

- Plugin directory not writable → install fails with error
- OpenCode config not readable → plugin won't load (OpenCode handles this)

### Environment Issues

- cly not in PATH → plugin fails silently on notification attempt
- Corrupted plugin file → OpenCode logs error on startup

## Dependencies

### Go Dependencies (Existing)

All required dependencies already in project:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration
- `github.com/yurifrl/cly/pkg/config` - Config management
- `github.com/yurifrl/cly/pkg/notify` - Notification infrastructure
- `github.com/yurifrl/cly/pkg/style` - Terminal styling

### TypeScript Dependencies (Handled by OpenCode)

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

2. **Restart OpenCode**

All existing functionality remains intact.

## Success Criteria

This implementation is considered successful when:

1. ✅ Plugin installs with single command
2. ✅ Notifications work immediately after restart
3. ✅ All configuration options respected
4. ✅ No OpenCode interruptions on errors
5. ✅ Existing notify infrastructure reused
6. ✅ Clear status/debugging commands available
7. ✅ Manual testing checklist passes

## Questions & Decisions Log

### Q1: Why no environment variables?

**A:** The plugin can execute `cly` commands directly using OpenCode's `$` shell API. The `cly notify hook opencode` command reads config from `~/.config/cly/config.yaml`, eliminating the need for environment variable passing.

### Q2: Why only copy index.ts and not package.json?

**A:** OpenCode auto-installs dependencies for plugins in `~/.config/opencode/plugin/`. The plugin only needs the .ts file - OpenCode handles dependency resolution automatically.

### Q3: Why message_length default of 200 chars?

**A:** Based on typical desktop notification limits:
- macOS notifications: ~200-300 chars visible
- Linux notify-send: ~100-200 chars comfortable
- Windows toasts: ~150-200 chars

200 chars balances detail vs readability across platforms.

### Q4: Why fail silently in plugin?

**A:** Notifications are non-critical - OpenCode should never be interrupted because a notification failed. Silent failures ensure robust user experience. Users can debug with `cly notify opencode status` if notifications aren't appearing.

### Q5: Why truncate in Go instead of TypeScript?

**A:** Centralize all configuration logic in one place (cly). Plugin stays simple and focused on message extraction. Config changes don't require plugin reinstall.

---

## Ready for Implementation

This draft provides:
- ✅ Complete architecture overview
- ✅ Detailed implementation plan (Phase 1-4)
- ✅ All file changes documented
- ✅ Configuration structure defined
- ✅ Installation workflow specified (single command!)
- ✅ Testing plan (manual)
- ✅ Edge cases considered
- ✅ Success criteria defined
- ✅ No environment variable complexity

**Key Improvement over v1:** Eliminated env var step completely - users just run `cly notify opencode install` and restart OpenCode.

**Next Steps:**
1. Review this draft
2. Clarify any questions or concerns
3. Proceed with implementation following this plan
