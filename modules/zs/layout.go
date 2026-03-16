package zs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var layoutDirPattern = regexp.MustCompile(`LAYOUT DIR[^\"]*\"([^\"]+)\"`)

type layoutItem struct {
	Name    string
	Path    string
	Display string
}

func ResolveLayout(explicit string, noLayout, tabOnly bool, targetName string) (string, error) {
	if noLayout {
		debugf("layout resolution: no-layout => default")
		return "default", nil
	}
	if explicit != "" {
		debugf("layout resolution: explicit=%q", explicit)
		return explicit, nil
	}

	layouts, err := ListLayouts(tabOnly)
	if err != nil {
		return "", err
	}
	debugf("layout candidates count=%d sample=%s", len(layouts), summarizeStrings(layouts, 10))
	if len(layouts) == 0 {
		debugf("layout resolution: no layouts found => default")
		return "default", nil
	}

	return SelectLayout(layouts, targetName, tabOnly)
}

func ListLayouts(tabOnly bool) ([]string, error) {
	dir, err := DetectLayoutDir()
	if err != nil {
		debugf("layout dir detection failed: %v", err)
		return nil, nil
	}

	var layouts []string
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".kdl" {
			return nil
		}
		layouts = append(layouts, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk layouts: %w", err)
	}

	if tabOnly {
		layouts, err = filterTabLayouts(layouts)
		if err != nil {
			return nil, err
		}
	}

	sort.Strings(layouts)
	return layouts, nil
}

func DetectLayoutDir() (string, error) {
	debugf("running: zellij setup --check")
	out, err := exec.Command("zellij", "setup", "--check").CombinedOutput()
	if err != nil {
		debugf("zellij setup --check failed output=%q", string(out))
		return "", fmt.Errorf("zellij setup --check: %w", err)
	}

	match := layoutDirPattern.FindStringSubmatch(string(out))
	if len(match) < 2 {
		return "", fmt.Errorf("could not determine zellij layout directory")
	}
	debugf("layout dir=%q", match[1])
	return match[1], nil
}

func SelectLayout(layouts []string, targetName string, tabOnly bool) (string, error) {
	commandName, err := fuzzyCommand()
	if err != nil {
		return "", err
	}

	items := buildLayoutItems(layouts)
	preview, err := layoutPreviewCommand()
	if err != nil {
		return "", err
	}

	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s\t%s", item.Path, item.Display))
	}

	header := fmt.Sprintf("Select layout for %q", targetName)
	input := strings.Join(lines, "\n")
	if tabOnly {
		input = "\t\n" + input
	}
	if input != "" {
		input += "\n"
	}
	debugf("layout picker header=%q items=%d sample=%s", header, len(items), summarizeStrings(lines, 10))

	cmd := exec.Command(commandName,
		"--with-nth=2",
		"--delimiter=\t",
		"--reverse",
		"--header", header,
		"--preview", preview,
	)
	cmd.Stdin = strings.NewReader(input)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code := exitErr.ExitCode()
			debugf("layout picker exit code=%d", code)
			if code == 1 || code == 130 {
				return "", ErrSelectionCanceled
			}
		}
		return "", fmt.Errorf("layout picker failed: %w", err)
	}

	selected := strings.TrimSpace(stdout.String())
	debugf("layout picker selected=%q", selected)
	if selected == "" {
		return "", ErrSelectionCanceled
	}

	parts := strings.SplitN(selected, "\t", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected layout picker output: %q", selected)
	}

	selectedPath := parts[0]
	for _, item := range items {
		if item.Path == selectedPath {
			return item.Path, nil
		}
	}

	return "", fmt.Errorf("unexpected layout selection: %q", selectedPath)
}

func buildLayoutItems(layouts []string) []layoutItem {
	items := make([]layoutItem, 0, len(layouts)+1)
	for _, layout := range layouts {
		name := filepath.Base(layout)
		name = strings.TrimSuffix(name, filepath.Ext(name))
		items = append(items, layoutItem{
			Name:    name,
			Path:    layout,
			Display: name,
		})
	}
	items = append(items, layoutItem{
		Name:    "default",
		Path:    "default",
		Display: "default",
	})
	return items
}

func filterTabLayouts(layouts []string) ([]string, error) {
	filtered := make([]string, 0, len(layouts))
	for _, layout := range layouts {
		content, err := os.ReadFile(layout)
		if err != nil {
			return nil, fmt.Errorf("read layout %s: %w", layout, err)
		}
		if !strings.Contains(string(content), "tab ") {
			filtered = append(filtered, layout)
		}
	}
	return filtered, nil
}

func layoutPreviewCommand() (string, error) {
	if _, err := lookPath("bat"); err == nil {
		return `sh -c 'if [ "$1" = "default" ]; then printf "%s\n" "default"; else bat --color=always --style=numbers --wrap=auto --paging=never "$1"; fi' sh {2}`,
			nil
	}
	return `sh -c 'if [ "$1" = "default" ]; then printf "%s\n" "default"; else cat "$1"; fi' sh {2}`,
		nil
}
