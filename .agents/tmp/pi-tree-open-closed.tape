Output .agents/tmp/pi-tree-open-closed.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 700
Set Padding 20

# Start pi-tree TUI - live view shows open workspaces bright, closed dim
Type "/tmp/cly-test pt"
Enter
Sleep 3s

# Open history to see an old snapshot - closed sessions will be dimmed
Type "h"
Sleep 2s

# Pick an older version
Type "k"
Sleep 500ms
Type "k"
Sleep 500ms
Enter
Sleep 3s

# Quit
Type "q"
Sleep 500ms
