Output .agents/tmp/pi-tree-one-per-ws.gif

Set Shell "bash"
Set FontSize 14
Set Width 1400
Set Height 700
Set Padding 20

Type "/tmp/cly-test pt"
Enter
Sleep 3s

# Should show only 1 session per workspace now
# Open history to see latest was auto-updated
Type "h"
Sleep 2s

# Close history
Type "h"
Sleep 2s

Type "q"
Sleep 500ms
