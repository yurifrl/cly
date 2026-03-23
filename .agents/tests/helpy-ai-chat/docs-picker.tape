# VHS tape: Helpy docs picker with frontmatter descriptions
# Validates: docs load, descriptions from frontmatter show, navigation works

Output .agents/tests/helpy-ai-chat/docs-picker.gif

Require cly

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Padding 10
Set Theme "Dracula"
Set TypingSpeed 50ms

# Open docs picker
Type "cly helpy -d"
Enter
Sleep 3s

# Navigate down through docs to see descriptions
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 1s

# Type filter to narrow results
Type "gh-dash"
Sleep 1s

# Open the doc
Enter
Sleep 2s

# Scroll down to see content
Down 5
Sleep 1s

# Go back to picker
Escape
Sleep 1s

# Quit
Escape
Sleep 500ms
