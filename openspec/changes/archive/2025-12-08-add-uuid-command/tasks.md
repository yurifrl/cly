# Implementation Tasks

## 1. Dependencies
- [x] 1.1 Install Bubbles: `go get github.com/charmbracelet/bubbles`
- [x] 1.2 Install Bubbletea: `go get github.com/charmbracelet/bubbletea`
- [x] 1.3 Install google/uuid: `go get github.com/google/uuid`
- [x] 1.4 Run `go mod tidy`

## 2. Module Structure
- [x] 2.1 Create `modules/uuid/` directory
- [x] 2.2 Write `modules/uuid/cmd.go` with Register function
- [x] 2.3 Write `modules/uuid/uuid.go` with Bubbletea implementation

## 3. Bubbletea Implementation
- [x] 3.1 Define item type implementing list.Item interface
- [x] 3.2 Create model struct with list, choice, quitting, generated fields
- [x] 3.3 Implement initialModel() with 3 UUID options
- [x] 3.4 Implement Init() method
- [x] 3.5 Implement Update() with keyboard handling (q, enter)
- [x] 3.6 Implement View() showing list or generated UUID
- [x] 3.7 Add UUID generation logic for v4, v7, and multiple options

## 4. Module Registration
- [x] 4.1 Add uuid import to `cmd/root.go`
- [x] 4.2 Add init() function calling uuid.Register(RootCmd)

## 5. Testing & Validation
- [x] 5.1 Test: `go run main.go --help` shows uuid in command list
- [x] 5.2 Test: `go run main.go uuid` displays interactive list
- [x] 5.3 Test: Arrow keys navigate list options
- [x] 5.4 Test: Enter generates and displays UUID
- [x] 5.5 Test: 'q' cancels without generating
- [x] 5.6 Test: All three options (v4, v7, multiple) work correctly
- [x] 5.7 Test: `go build` succeeds

## 6. Documentation
- [x] 6.1 Verify directory structure matches docs/02-first-module.md
- [x] 6.2 Confirm success criteria from docs/02-first-module.md

## Dependencies
- Section 1 must complete before section 2
- Section 2 must complete before section 3
- Section 3 must complete before section 4
- Section 4 must complete before section 5

## Parallel Work
- Tasks 3.2-3.7 can be written in parallel (same file)
