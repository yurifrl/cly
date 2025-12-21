# Implementation Tasks

## 1. Dependencies
- [x] 1.1 Install textarea: `go get github.com/charmbracelet/bubbles/textarea`
- [x] 1.2 Run `go mod tidy`

## 2. Demo Namespace Structure
- [x] 2.1 Create `modules/demo/` directory
- [x] 2.2 Write `modules/demo/cmd.go` (parent namespace command)
- [x] 2.3 Create `modules/demo/chat/` subdirectory

## 3. Chat Subcommand
- [x] 3.1 Write `modules/demo/chat/cmd.go` with Register function
- [x] 3.2 Write `modules/demo/chat/chat.go` with Bubbletea implementation
- [x] 3.3 Adapt reference code: model struct with viewport, textarea, messages
- [x] 3.4 Implement Init() returning textarea.Blink
- [x] 3.5 Implement Update() handling WindowSizeMsg, KeyMsg (Enter, Esc, Ctrl+C)
- [x] 3.6 Implement View() combining viewport and textarea with gap
- [x] 3.7 Style messages with sender coloring

## 4. Namespace Registration
- [x] 4.1 Register chat subcommand with demo parent in demo/cmd.go init()
- [x] 4.2 Add demo import to `cmd/root.go`
- [x] 4.3 Add demo.Register(RootCmd) in `cmd/root.go` init()

## 5. Testing & Validation
- [x] 5.1 Test: `go run main.go --help` shows demo command
- [x] 5.2 Test: `go run main.go demo --help` shows chat subcommand
- [x] 5.3 Test: `go run main.go demo chat` launches interactive chat
- [x] 5.4 Test: Type message and press Enter adds to viewport
- [x] 5.5 Test: Viewport scrolls with multiple messages
- [x] 5.6 Test: Esc/Ctrl+C exits gracefully
- [x] 5.7 Test: `go build` succeeds

## 6. Documentation
- [x] 6.1 Verify namespace pattern: parent command with subcommands
- [x] 6.2 Confirm chat demo matches reference implementation behavior

## Dependencies
- Section 1 must complete before section 2
- Section 2 must complete before section 3
- Section 3 must complete before section 4
- Section 4 must complete before section 5

## Parallel Work
- Tasks 3.2-3.7 can be written in same file (chat.go)
