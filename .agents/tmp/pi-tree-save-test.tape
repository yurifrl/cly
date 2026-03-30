Output .agents/tmp/pi-tree-save-test.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 700
Set Padding 20

# Start pi-tree TUI
Type "/tmp/cly-test pt"
Enter
Sleep 3s

# Open history, pick v1
Type "h"
Sleep 1s
Type "k"
Sleep 500ms
Type "k"
Sleep 500ms

# Enter to load v1 tree
Enter
Sleep 2s

# Save creates a new version of the viewed snapshot
Type "s"
Sleep 2s

# Open history again to verify new version was created
Type "h"
Sleep 2s

# Close history — tree stays on selected version
Type "h"
Sleep 2s

# Quit
Type "q"
Sleep 500ms
