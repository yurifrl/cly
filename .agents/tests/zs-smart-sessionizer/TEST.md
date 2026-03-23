# Test: ZS Smart Sessionizer Parity

## Context
Validates that `cly zs` matches the behavior of the original `zellij-smart-sessionizer` bash script for the key flows:
- outside Zellij: pick from sessions + zoxide dirs, attach to an existing session or create a new session with layout
- inside Zellij: pick from zoxide dirs, create a new tab with layout and focus it

The test uses the original reference script plus a deterministic fake environment (`zellij`, `zoxide`, `fzf`, `sk`) so both implementations receive the exact same inputs.

## Workflow

### 1. Build cly
```bash
cd /Users/yuri/Workdir/Yuri/cly
GOFLAGS= go build -o dist/cly .
```

### 2. Compare outside-Zellij flow
```bash
bash .agents/tests/zs-smart-sessionizer/test-run.sh outside
```

### 3. Compare outside-Zellij attach flow
```bash
bash .agents/tests/zs-smart-sessionizer/test-run.sh attach
```

### 4. Compare inside-Zellij flow
```bash
bash .agents/tests/zs-smart-sessionizer/test-run.sh inside
```

### 5. Visual run via VHS
```bash
vhs .agents/tests/zs-smart-sessionizer/outside-create.tape
vhs .agents/tests/zs-smart-sessionizer/inside-tab.tape
```

## Assertions

| Step | What to check | How |
|------|--------------|-----|
| 2 | Outside create flow picker input matches reference | script diffs `picker-1.txt` and `picker-2.txt` |
| 2 | Outside create flow zellij command matches reference | script diffs `zellij.log` |
| 3 | Outside attach flow picker input matches reference | script diffs `picker-1.txt` |
| 3 | Outside attach flow zellij command matches reference | script diffs `zellij.log` |
| 4 | Inside flow picker input matches reference | script diffs `picker-1.txt` and `picker-2.txt` |
| 4 | Inside flow zellij commands match reference | script diffs `zellij.log` |
| 5 | VHS output shows parity pass output | generated gif shows `PASS` summary |

## Commands
```bash
bash .agents/tests/zs-smart-sessionizer/test-run.sh outside
bash .agents/tests/zs-smart-sessionizer/test-run.sh attach
bash .agents/tests/zs-smart-sessionizer/test-run.sh inside
vhs .agents/tests/zs-smart-sessionizer/outside-create.tape
vhs .agents/tests/zs-smart-sessionizer/inside-tab.tape
```

## Cleanup
Delete generated media if desired:
```bash
rm -f .agents/tests/zs-smart-sessionizer/*.gif .agents/tests/zs-smart-sessionizer/*.mp4
```
