Output .agents/tmp/pi-tree-v4.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 700
Set Padding 20

# Start pi-tree TUI
Type "/tmp/cly-test pt"
Enter
Sleep 3s

# Open history
Type "h"
Sleep 1s

# Navigate to v4 (13 sessions) — need to go up from current position
# Current is at latest (v18), need to go up many times
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 200ms
Type "k"
Sleep 1s

# Should be on v4 now — press enter to load
Enter
Sleep 3s

# Quit
Type "q"
Sleep 500ms
