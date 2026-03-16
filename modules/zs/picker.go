package zs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrSelectionCanceled = errors.New("selection canceled")

const (
	selectionKindSession = "session"
	selectionKindDir     = "dir"
)

type Selection struct {
	Kind  string
	Value string
}

type pickerItem struct {
	Label string
	Value string
	Kind  string
}

var lookPath = exec.LookPath

func ensureDependencies() error {
	for _, binary := range []string{"zellij", "zoxide"} {
		if _, err := lookPath(binary); err != nil {
			return fmt.Errorf("%s not found in PATH", binary)
		}
		debugf("found dependency=%s", binary)
	}

	command, err := fuzzyCommand()
	if err != nil {
		return err
	}
	debugf("using fuzzy command=%s", command)

	return nil
}

func SelectSessionTarget(sessions, dirs []string) (Selection, error) {
	items := buildSessionItems(sessions, dirs)
	if len(items) == 0 {
		return Selection{}, fmt.Errorf("no zellij sessions or zoxide directories found")
	}
	return runPicker(items, "Select session:")
}

func SelectDirectory(dirs []string, header string) (Selection, error) {
	items := buildDirectoryItems(dirs)
	if len(items) == 0 {
		return Selection{}, fmt.Errorf("no zoxide directories found")
	}
	return runPicker(items, header)
}

func buildSessionItems(sessions, dirs []string) []pickerItem {
	items := make([]pickerItem, 0, len(sessions)+len(dirs))

	for _, session := range sessions {
		session = strings.TrimSpace(session)
		if session == "" {
			continue
		}
		items = append(items, pickerItem{Label: session, Value: session, Kind: selectionKindSession})
	}

	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		items = append(items, pickerItem{Label: shortenHome(dir), Value: dir, Kind: selectionKindDir})
	}

	return items
}

func buildDirectoryItems(dirs []string) []pickerItem {
	items := make([]pickerItem, 0, len(dirs))

	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		items = append(items, pickerItem{Label: shortenHome(dir), Value: dir, Kind: selectionKindDir})
	}

	return items
}

func runPicker(items []pickerItem, header string) (Selection, error) {
	commandName, err := fuzzyCommand()
	if err != nil {
		return Selection{}, err
	}

	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, item.Label)
	}

	input := strings.Join(lines, "\n")
	if input != "" {
		input += "\n"
	}
	debugf("picker header=%q items=%d sample=%s", header, len(items), summarizeStrings(lines, 10))

	cmd := exec.Command(commandName, "--reverse", "--header", header)
	cmd.Stdin = strings.NewReader(input)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			debugf("picker exit code=%d", code)
			if code == 1 || code == 130 {
				return Selection{}, ErrSelectionCanceled
			}
		}
		return Selection{}, fmt.Errorf("picker failed: %w", err)
	}

	selected := strings.TrimSpace(stdout.String())
	debugf("picker selected=%q", selected)
	if selected == "" {
		return Selection{}, ErrSelectionCanceled
	}

	for _, item := range items {
		if item.Label == selected {
			return Selection{Kind: item.Kind, Value: item.Value}, nil
		}
	}

	return Selection{}, fmt.Errorf("unexpected picker output: %q", selected)
}

func fuzzyCommand() (string, error) {
	for _, candidate := range []string{"fzf", "sk"} {
		if _, err := lookPath(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("fzf or sk not found in PATH")
}

func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}

	cleanHome := filepath.Clean(home)
	cleanPath := filepath.Clean(path)
	if cleanPath == cleanHome {
		return "~"
	}

	prefix := cleanHome + string(filepath.Separator)
	if strings.HasPrefix(cleanPath, prefix) {
		return "~" + string(filepath.Separator) + strings.TrimPrefix(cleanPath, prefix)
	}

	return path
}
