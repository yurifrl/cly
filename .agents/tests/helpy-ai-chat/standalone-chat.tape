# VHS tape: Helpy standalone AI chat (--chat flag)
# Validates: chat opens, typing sends message, AI streams response
# Requires: ANTHROPIC_API_KEY set in environment

Output .agents/tests/helpy-ai-chat/standalone-chat.gif

Require cly

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Padding 10
Set Theme "Dracula"
Set TypingSpeed 50ms

# Open standalone AI chat with a specific doc as context
Type "cly helpy --chat -f ~/DotFiles/docs/gh-dash-enhance-diffnav-quick.md"
Enter
Sleep 2s

# Type a question about the doc
Type "What are the main keyboard shortcuts?"
Sleep 500ms

# Send the message
Enter

# Wait for AI to stream response
Sleep 15s

# Quit
Escape
Sleep 500ms
