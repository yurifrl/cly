package dotfiles

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Mapping struct {
	Source      string
	Destination string
	IsDir       bool
	LineNum     int
}

type Config struct {
	BaseDir         string
	Mappings        []Mapping
	InstallCommands []string
	Errors          []string
}

func ParseConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	baseDir := filepath.Dir(configPath)
	cfg := &Config{
		BaseDir: baseDir,
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "!") {
			cmd := strings.TrimSpace(line[1:])
			cfg.InstallCommands = append(cfg.InstallCommands, cmd)
			continue
		}

		if err := parseMappingLine(cfg, line, lineNum, baseDir); err != nil {
			cfg.Errors = append(cfg.Errors, fmt.Sprintf("line %d: %s", lineNum, err.Error()))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return cfg, nil
}

func parseMappingLine(cfg *Config, line string, lineNum int, baseDir string) error {
	parts := strings.Split(line, "->")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format, expected 'source -> destination'")
	}

	source := strings.TrimSpace(parts[0])
	destination := strings.TrimSpace(parts[1])

	if source == "" || destination == "" {
		return fmt.Errorf("source or destination is empty")
	}

	isDir := strings.HasSuffix(source, "/")
	if isDir {
		source = strings.TrimSuffix(source, "/")
		destination = strings.TrimSuffix(destination, "/")
	}

	source = resolvePath(source, baseDir)
	destination = expandTilde(destination)

	cfg.Mappings = append(cfg.Mappings, Mapping{
		Source:      source,
		Destination: destination,
		IsDir:       isDir,
		LineNum:     lineNum,
	})

	return nil
}

func resolvePath(path, baseDir string) string {
	if strings.HasPrefix(path, "./") {
		path = path[2:]
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return path
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
