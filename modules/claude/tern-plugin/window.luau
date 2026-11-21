-- Show the Anthropic icon for Claude Code panes on tabs, pane rows and parked
-- cards. `c` is the shorthand for `cly claude` (and `headroom wrap claude`
-- runs underneath it), so Tern would otherwise show the shell icon.
for _, program in ipairs({ "c", "claude", "headroom" }) do
    tern.chrome.program_icon(program, "provider-anthropic")
end
