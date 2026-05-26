package dotfiles

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

type Mapping struct {
	Source      string
	Destination string
	IsDir       bool
	LineNum     int
}

// CacheEntry models a single `@cache` directive. The whole rest-of-line
// after `@cache ` is the command; sha256(Command) is the identity used to
// decide whether to skip the run on subsequent invocations.
//
// Two identical `@cache` lines hash to the same value and are de-facto
// idempotent — the second occurrence becomes a cache hit on the first's
// lock entry, so the command runs once per unique command text.
type CacheEntry struct {
	Command string
	LineNum int
}

type OpMapping struct {
	Source      string
	Destination string
	Account     string
	// IsReference is true when Source is a raw 1Password secret reference
	// (e.g. op://Vault/Item/field). In that case the destination is written
	// with `op read --out-file`. Otherwise Source is a template file path and
	// `op inject` is used.
	IsReference bool
	LineNum     int
}

type Install struct {
	URL     string
	LineNum int
}

type Config struct {
	BaseDir         string
	Mappings        []Mapping
	InstallCommands []string
	CacheEntries    []CacheEntry
	Installs        []Install
	OpMappings      []OpMapping
	Errors          []string
}

func ParseConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer file.Close()

	baseDir := filepath.Dir(configPath)
	cfg := &Config{BaseDir: baseDir}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "!"):
			cmd := strings.TrimSpace(line[1:])
			cfg.InstallCommands = append(cfg.InstallCommands, cmd)
		case strings.HasPrefix(line, "@install "):
			url := strings.TrimSpace(strings.TrimPrefix(line, "@install "))
			if url != "" {
				cfg.Installs = append(cfg.Installs, Install{URL: url, LineNum: lineNum})
			}
		case strings.HasPrefix(line, "@op "):
			if err := parseOpLine(cfg, line, lineNum, baseDir); err != nil {
				cfg.Errors = append(cfg.Errors, fmt.Sprintf("line %d: %s", lineNum, err.Error()))
			}
		case strings.HasPrefix(line, "@"):
			if err := parseJobLine(cfg, line, lineNum); err != nil {
				cfg.Errors = append(cfg.Errors, fmt.Sprintf("line %d: %s", lineNum, err.Error()))
			}
		default:
			if err := parseMappingLine(cfg, line, lineNum, baseDir); err != nil {
				cfg.Errors = append(cfg.Errors, fmt.Sprintf("line %d: %s", lineNum, err.Error()))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return cfg, nil
}

// legacyCacheNameForm matches the removed `@cache <name> -- <command>`
// grammar. We only flag it when the remainder of the line begins with a
// bareword identifier followed by ` -- ` so that legitimate `--flag`
// arguments inside a real command are not misclassified.
var legacyCacheNameForm = regexp.MustCompile(`^[A-Za-z0-9_.-]+ -- `)

func parseJobLine(cfg *Config, line string, lineNum int) error {
	// Hard-fail removed directives loudly; users must migrate.
	switch {
	case strings.HasPrefix(line, "@startup ") || line == "@startup":
		return fmt.Errorf("@startup is removed; migrate background processes to process-compose.yaml")
	case strings.HasPrefix(line, "@interval ") || line == "@interval":
		return fmt.Errorf("@interval is removed; migrate scheduled tasks to process-compose.yaml")
	case strings.HasPrefix(line, "@once ") || line == "@once":
		return fmt.Errorf("@once is removed; use @cache instead")
	}

	if !strings.HasPrefix(line, "@cache ") && line != "@cache" {
		fields := strings.Fields(line)
		name := "@"
		if len(fields) > 0 {
			name = fields[0]
		}
		return fmt.Errorf("unknown directive %q", name)
	}

	command := strings.TrimSpace(strings.TrimPrefix(line, "@cache"))
	if command == "" {
		return fmt.Errorf("@cache requires a command")
	}

	if legacyCacheNameForm.MatchString(command) {
		return fmt.Errorf("@cache no longer takes a name; use '@cache <command>' (the command itself is the identity)")
	}

	cfg.CacheEntries = append(cfg.CacheEntries, CacheEntry{
		Command: command,
		LineNum: lineNum,
	})
	return nil
}

// ParseMapping parses a "source -> destination" string into a Mapping.
// Source paths starting with ./ are resolved relative to the dotfiles directory.
func ParseMapping(line string) (Mapping, error) {
	parts := strings.Split(line, "->")
	if len(parts) != 2 {
		return Mapping{}, fmt.Errorf("invalid format, expected 'source -> destination'")
	}

	source := strings.TrimSpace(parts[0])
	destination := strings.TrimSpace(parts[1])

	if source == "" || destination == "" {
		return Mapping{}, fmt.Errorf("source or destination is empty")
	}

	isDir := strings.HasSuffix(source, "/")
	if isDir {
		source = strings.TrimSuffix(source, "/")
		destination = strings.TrimSuffix(destination, "/")
	}

	// Resolve source relative to dotfiles dir
	cfg := pkgconfig.Get()
	dotfilesDir := "~/DotFiles"
	if cfg != nil && cfg.App.DotFilesDir != "" {
		dotfilesDir = cfg.App.DotFilesDir
	}
	baseDir := expandTilde(dotfilesDir)

	source = resolvePath(source, baseDir)
	destination = expandTilde(destination)

	return Mapping{
		Source:      source,
		Destination: destination,
		IsDir:       isDir,
	}, nil
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

	// Glob support: ./path/* -> ~/dest/ expands to individual file symlinks
	if strings.Contains(source, "*") {
		matches, err := filepath.Glob(source)
		if err != nil {
			return fmt.Errorf("invalid glob pattern: %w", err)
		}
		if len(matches) == 0 {
			cfg.Errors = append(cfg.Errors, fmt.Sprintf("line %d: glob pattern matched no files: %s", lineNum, source))
			return nil
		}
		for _, match := range matches {
			name := filepath.Base(match)
			info, statErr := os.Stat(match)
			matchIsDir := statErr == nil && info.IsDir()
			cfg.Mappings = append(cfg.Mappings, Mapping{
				Source:      match,
				Destination: filepath.Join(destination, name),
				IsDir:       matchIsDir,
				LineNum:     lineNum,
			})
		}
		return nil
	}

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

func parseOpLine(cfg *Config, line string, lineNum int, baseDir string) error {
	// @op [account=...] ./source -> ~/destination
	rest := strings.TrimPrefix(line, "@op ")
	rest = strings.TrimSpace(rest)

	var account string
	if strings.HasPrefix(rest, "account=") {
		spaceIdx := strings.IndexByte(rest, ' ')
		if spaceIdx == -1 {
			return fmt.Errorf("@op requires source -> destination")
		}
		account = strings.TrimPrefix(rest[:spaceIdx], "account=")
		rest = strings.TrimSpace(rest[spaceIdx+1:])
	}

	parts := strings.SplitN(rest, "->", 2)
	if len(parts) != 2 {
		return fmt.Errorf("@op requires source -> destination")
	}

	source := strings.TrimSpace(parts[0])
	destination := strings.TrimSpace(parts[1])

	if source == "" || destination == "" {
		return fmt.Errorf("@op source or destination is empty")
	}

	// Strip surrounding quotes on the source (useful for op:// refs with spaces).
	source = strings.Trim(source, "\"'")

	isRef := strings.HasPrefix(source, "op://")
	if !isRef {
		source = resolvePath(source, baseDir)
	}
	destination = expandTilde(destination)

	cfg.OpMappings = append(cfg.OpMappings, OpMapping{
		Source:      source,
		Destination: destination,
		Account:     account,
		IsReference: isRef,
		LineNum:     lineNum,
	})

	return nil
}
