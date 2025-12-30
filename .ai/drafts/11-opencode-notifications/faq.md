# OpenCode Notifications - FAQ

## Installation & Setup

### Q: Do I need npm to install this?
**A:** No. The plugin files are embedded in the cly binary. Just run `cly notify opencode install`.

### Q: Where does the plugin get installed?
**A:** `~/.config/opencode/plugin/cly-notify.ts` - OpenCode auto-loads plugins from this directory.

### Q: Why do I need to set environment variables?
**A:** OpenCode plugins run in a sandbox with no filesystem access. Environment variables are the standard way for plugins to receive configuration.

### Q: Can I skip the env var step?
**A:** No. Without env vars, the plugin won't know it's enabled or how to behave. Use `cly notify opencode env >> ~/.zshrc` to set them up once.

### Q: Do I need to restart OpenCode?
**A:** Yes, after installation. OpenCode loads plugins at startup.

## Configuration

### Q: How do I change the notification title?
**A:** Edit `~/.config/cly/config.yaml`:
```yaml
modules:
  notify:
    hooks:
      opencode:
        title: "Your Custom Title"
```
Then regenerate env vars: `cly notify opencode env >> ~/.zshrc`

### Q: How do I disable notifications?
**A:** Set `enabled: false` in config.yaml, or export `CLY_NOTIFY_OPENCODE_ENABLED=false`

### Q: How do I only notify for main sessions (skip subagents)?
**A:** Set `filter_subagents: true` in config.yaml, then regenerate env vars.

### Q: How do I show longer messages?
**A:** Increase `message_length` in config.yaml:
```yaml
opencode:
  message_length: 500  # default is 200
```
Then regenerate env vars.

### Q: Can I use a different sound?
**A:** Yes. Available sounds: Glass, Blow, Submarine, Ping, Pop, Purr, Frog, Basso, Hero, Morse
```yaml
opencode:
  sound: "Blow"
```

### Q: Can I disable sound but keep notifications?
**A:** Yes:
```yaml
modules:
  notify:
    sound: false  # Disables sound globally
```

## Troubleshooting

### Q: Notifications not appearing?
**A:** Run `cly notify opencode status` to diagnose:
1. Check plugin installed
2. Check cly in PATH
3. Check env vars set
4. Manually test: `cly notify hook opencode --message "test"`

### Q: Plugin installed but no notifications from OpenCode?
**A:** Check:
1. Did you restart OpenCode after install?
2. Are env vars loaded? Run `echo $CLY_NOTIFY_OPENCODE_ENABLED`
3. Is config enabled? Check `CLY_NOTIFY_ENABLED` and `CLY_NOTIFY_OPENCODE_ENABLED`
4. Check OpenCode console for errors

### Q: Getting "cly: command not found" errors?
**A:** The plugin can't find cly binary. Solutions:
1. Make sure cly is in PATH: `which cly`
2. Set CLY_PATH explicitly: `export CLY_PATH=/path/to/cly`
3. Reinstall cly: `go install github.com/yurifrl/cly@latest`

### Q: Messages are cut off?
**A:** Increase `message_length` in config.yaml. Default is 200 chars.

### Q: Getting notifications from subagent sessions?
**A:** Set `filter_subagents: true` to only notify for main sessions.

### Q: How do I see what message OpenCode sent?
**A:** Check terminal where OpenCode is running - plugin logs errors (but not successes to avoid spam).

## Uninstallation

### Q: How do I uninstall?
**A:** 
```bash
cly notify opencode uninstall
```
Then manually remove env vars from shell profile (~/.zshrc).

### Q: Will uninstalling break my cly setup?
**A:** No. Uninstalling only removes the OpenCode plugin. All other cly notify features continue working.

### Q: Can I reinstall?
**A:** Yes, run `cly notify opencode install` again. It will overwrite existing files.

## Advanced

### Q: Can I customize the message format?
**A:** Not currently. The plugin extracts the last assistant message automatically. Future enhancement could add templates.

### Q: Can I notify to Slack/Discord?
**A:** Not directly. Current implementation uses desktop notifications only. Future enhancement could add webhooks.

### Q: Can I run different commands instead of cly notify?
**A:** Yes. Edit the plugin file at `~/.config/opencode/plugin/cly-notify.ts` and change the command in the `$` call.

### Q: Does this work on Linux/Windows?
**A:** macOS: Yes (tested)
Linux: Yes (uses notify-send via beeep)
Windows: Yes (uses Windows toast notifications via beeep)

### Q: Can I have multiple plugins?
**A:** Yes. OpenCode loads all `.ts` files from `~/.config/opencode/plugin/`. Name them differently to avoid conflicts.

### Q: Where can I see the plugin source?
**A:** After installation: `cat ~/.config/opencode/plugin/cly-notify.ts`
Or in cly source: `modules/notify/opencode/plugin/index.ts`

## Performance

### Q: Will this slow down OpenCode?
**A:** No. The plugin only runs on session.idle events (when OpenCode finishes a task). Notification sending is async.

### Q: Does this use a lot of API calls?
**A:** Minimal. Only 2 API calls per notification:
1. `session.get` - check if subagent
2. `session.messages` - fetch last message

### Q: Can I rate-limit notifications?
**A:** Not currently. But session.idle events only fire when work completes, so notifications are naturally spaced out.

## Integration

### Q: Does this work with Zellij?
**A:** Yes. If `use_zellij_status: true` or `use_zellij_notify: true`, notifications also appear in Zellij.

### Q: Does this work with other cly notify hooks?
**A:** Yes. The opencode hook is independent. You can have notification, stop, and opencode hooks all enabled.

### Q: Can I use this with Claude Code hooks?
**A:** Yes. OpenCode and Claude Code are separate. Both can send notifications via cly.

### Q: Does this work in Docker/remote environments?
**A:** Desktop notifications require local OS notification system. In remote/Docker, the cly command runs but notifications may not display. Consider adding remote notification backends (ntfy.sh, webhooks) in future.

## Development

### Q: Can I modify the plugin?
**A:** Yes. Edit `~/.config/opencode/plugin/cly-notify.ts`. Changes take effect after restarting OpenCode.

### Q: How do I debug the plugin?
**A:** Add console.log statements in the plugin:
```typescript
console.log("Debug:", messageText)
```
Output appears in OpenCode terminal.

### Q: Can I test the plugin without OpenCode?
**A:** Not easily. The plugin uses OpenCode's SDK which requires OpenCode runtime. Test with `cly notify hook opencode --message "test"` instead.

### Q: Where are plugin logs?
**A:** OpenCode logs to stdout. Check the terminal where you ran `opencode`.

## Comparison

### Q: How is this different from Claude Code hooks?
**A:** 
- **Claude Code hooks**: Use ~/.claude/settings.json, triggered by Claude-specific events
- **OpenCode plugin**: Uses OpenCode plugin API, triggered by session.idle events
- Both use the same cly notification infrastructure

### Q: Should I use both?
**A:** If you use both OpenCode and Claude Code, yes. They're independent systems.

### Q: Which is better?
**A:** Neither - they serve different tools. Use the one for your AI tool.
