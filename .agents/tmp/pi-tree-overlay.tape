Output .agents/tmp/pi-tree-overlay.gif

Set Shell "bash"
Set FontSize 13
Set Width 1600
Set Height 800
Set Padding 10

# Start pi-tree TUI
Type "/tmp/cly-test pt"
Enter
Sleep 3s

# Open history — should overlay ON TOP of tree
Type "h"
Sleep 3s

# Close history — tree should be exactly the same
Type "h"
Sleep 2s

# Quit
Type "q"
Sleep 500ms
