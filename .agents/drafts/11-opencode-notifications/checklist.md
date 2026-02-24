# Implementation Checklist

## Phase 1: TypeScript Plugin

### Create: `modules/notify/opencode/plugin/index.ts`
- [ ] Import Plugin type from @opencode-ai/plugin
- [ ] Export ClyNotifyPlugin function
- [ ] Listen to session.idle events
- [ ] Check env vars: CLY_NOTIFY_ENABLED, CLY_NOTIFY_OPENCODE_ENABLED
- [ ] Read config: filter_subagents, message_length from env
- [ ] Fetch session.get to check for parentID (subagent)
- [ ] Fetch session.messages to get last message
- [ ] Filter for assistant role messages
- [ ] Extract text parts only
- [ ] Clean text: remove code blocks (```), inline code (`), normalize spaces
- [ ] Truncate to max length, add "..." if truncated
- [ ] Execute: `cly notify hook opencode --message "{text}"`
- [ ] Fail silently on errors

### Create: `modules/notify/opencode/plugin/package.json`
```json
{
  "name": "cly-notify-opencode-plugin",
  "version": "1.0.0",
  "type": "module",
  "dependencies": {
    "@opencode-ai/plugin": "latest"
  }
}
```

### Create: `modules/notify/opencode/plugin/tsconfig.json`
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "esModuleInterop": true
  }
}
```

## Phase 2: Go Installation

### Create: `modules/notify/opencode/install.go`
- [ ] Add `//go:embed plugin/*` directive
- [ ] Embed pluginFiles variable
- [ ] Implement Install() function:
  - [ ] Extract files to ~/.config/opencode/plugin/
  - [ ] Rename index.ts to cly-notify.ts
  - [ ] Print installation success + next steps
- [ ] Implement Uninstall() function:
  - [ ] Remove cly-notify.ts
  - [ ] Print uninstall success
- [ ] Implement Status() function:
  - [ ] Check plugin file exists
  - [ ] Check cly in PATH
  - [ ] Check env vars set
  - [ ] Print status report
- [ ] Implement Env() function:
  - [ ] Load config from pkgconfig.Get()
  - [ ] Find cly binary path with exec.LookPath
  - [ ] Print export statements

### Create: `modules/notify/opencode/cmd.go`
- [ ] Create Register() function
- [ ] Add opencode parent command
- [ ] Add install subcommand → calls Install()
- [ ] Add uninstall subcommand → calls Uninstall()
- [ ] Add status subcommand → calls Status()
- [ ] Add env subcommand → calls Env()

## Phase 3: Hook Handler

### Modify: `modules/notify/cmd.go`
- [ ] Import opencode package
- [ ] In createHookCmd(), add --message flag:
  - [ ] Add `var messageOverride string`
  - [ ] Add `cmd.Flags().StringVar(&messageOverride, ...)`
  - [ ] Check if messageOverride != "", override hookConfig.Message
- [ ] In Register(), add: `opencode.Register(notifyCmd)`

## Phase 4: Configuration

### Modify: `modules/config/config.yaml`
- [ ] Add opencode hook under modules.notify.hooks:
  ```yaml
  opencode:
    enabled: true
    title: "✅ OpenCode Complete"
    message: "Task completed"
    sound: "Glass"
    filter_subagents: false
    message_length: 200
  ```

### Modify: `pkg/config/config.go`
- [ ] Add fields to HookConfig struct:
  - [ ] `FilterSubagents bool`
  - [ ] `MessageLength int`
- [ ] Update GetNotify() to parse new fields:
  - [ ] Parse filter_subagents
  - [ ] Parse message_length

## Testing Checklist

### Installation
- [ ] Run: `cly notify opencode install`
- [ ] Verify: Plugin file at ~/.config/opencode/plugin/cly-notify.ts
- [ ] Run: `cly notify opencode status` (before env)
- [ ] Verify: Shows plugin installed, env vars not set
- [ ] Run: `cly notify opencode env`
- [ ] Verify: Outputs valid export statements
- [ ] Run: `source <(cly notify opencode env)`
- [ ] Run: `cly notify opencode status` (after env)
- [ ] Verify: All env vars shown as set

### Notifications
- [ ] Run: `cly notify hook opencode --message "Test"`
- [ ] Verify: Desktop notification appears with "Test"
- [ ] Test with long message (500+ chars)
- [ ] Verify: Truncated to configured length

### OpenCode Integration
- [ ] Restart OpenCode
- [ ] Start a task, wait for idle
- [ ] Verify: Notification appears with extracted message
- [ ] Verify: Message is cleaned (no code blocks)
- [ ] Verify: Message truncated appropriately

### Subagent Filtering
- [ ] Set filter_subagents: true in config
- [ ] Regenerate env: `cly notify opencode env >> ~/.zshrc`
- [ ] Restart shell and OpenCode
- [ ] Trigger subagent task
- [ ] Verify: No notification for subagent
- [ ] Trigger main session task
- [ ] Verify: Notification for main session

### Configuration
- [ ] Test with enabled: false
- [ ] Test with different sounds (Glass, Blow, Submarine)
- [ ] Test with different message_length values (50, 200, 500)
- [ ] Test with sound globally disabled

### Uninstall
- [ ] Run: `cly notify opencode uninstall`
- [ ] Verify: Plugin file removed
- [ ] Run: `cly notify opencode status`
- [ ] Verify: Shows plugin not installed

## Build Verification

After implementation:
- [ ] Run: `go build`
- [ ] Verify: No compilation errors
- [ ] Run: `go run . notify opencode --help`
- [ ] Verify: Commands listed correctly
- [ ] Run: `go run . notify hook --help`
- [ ] Verify: --message flag appears

## Documentation

- [ ] Update README.md with OpenCode notifications section
- [ ] Add example configuration
- [ ] Document installation steps
- [ ] Add troubleshooting section

## Edge Cases to Test

- [ ] Plugin file already exists (reinstall)
- [ ] OpenCode config directory doesn't exist (install creates it)
- [ ] cly not in PATH (plugin fails gracefully)
- [ ] Env vars not set (plugin disables itself)
- [ ] Invalid message_length in config (defaults to 200)
- [ ] Invalid filter_subagents in config (defaults to false)
- [ ] OpenCode API errors (plugin fails silently)
- [ ] Message extraction errors (uses default message)

## Success Criteria

All these must work:
- [ ] ✅ Install with single command
- [ ] ✅ Status shows clear information
- [ ] ✅ Env generation produces valid exports
- [ ] ✅ Direct hook test sends notification
- [ ] ✅ OpenCode integration sends notification on idle
- [ ] ✅ Configuration options all work
- [ ] ✅ Subagent filtering works
- [ ] ✅ Message truncation works
- [ ] ✅ Uninstall cleans up properly
- [ ] ✅ No errors in build
- [ ] ✅ No OpenCode interruptions on failures
