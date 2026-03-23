# VHS tape: Helpy AI chat panel with streaming
# Validates: Ctrl+A opens chat, typing sends message, AI streams response
# Requires: ANTHROPIC_API_KEY set in environment

Output .agents/tests/helpy-ai-chat/ai-chat.gif

Require cly

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Padding 10
Set Theme "Dracula"
Set TypingSpeed 50ms

# Open a specific doc with frontmatter
Type "cly helpy -f ~/DotFiles/docs/gh-dash-enhance-diffnav-quick.md"
Enter
Sleep 2s

# Toggle AI chat panel with Ctrl+A
Ctrl+A
Sleep 1s

# Type a question about the doc
Type "What keyboard shortcuts does gh-dash use?"
Sleep 500ms

# Send the message
Enter

# Wait for AI to stream response
Sleep 15s

# Scroll up in chat to see full response
Up 3
Sleep 2s

# Close chat
Escape
Sleep 1s

# Verify viewport is back to full height
Sleep 1s

# Quit
Type "q"
Sleep 500ms
