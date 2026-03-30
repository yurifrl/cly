Output .agents/tmp/pi-tree-test.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 700
Set Padding 20

# Start the pi-tree TUI (will fall back to snapshots since no cmux in VHS)
Type "/tmp/cly-test pt"
Enter
Sleep 3s

# Open history panel with 'h'
Type "h"
Sleep 2s

# Navigate up through versions
Type "k"
Sleep 1s
Type "k"
Sleep 1s

# Select a history version (enter) — should load that version's tree
Enter
Sleep 2s

# Open history again
Type "h"
Sleep 2s

# Navigate down
Type "j"
Sleep 1s

# Close history with h — should keep selected version's tree
Type "h"
Sleep 2s

# Quit
Type "q"
Sleep 1s
