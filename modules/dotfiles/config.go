package dotfiles

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

type Mapping struct {
	Source      string
	Destination string
	IsDir       bool
	LineNum     int
}

type JobRun string

const (
	JobRunStartup  JobRun = "startup"
	JobRunInterval JobRun = "interval"
	JobRunOnce     JobRun = "once"
	JobRunCache    JobRun = "cache"
)

type Job struct {
	Name      string
	Run       JobRun
	Command   string
	Every     string
	Check     string // reserved for future use
	KeepAlive bool
	LineNum   int
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
	Jobs            []Job
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

func parseJobLine(cfg *Config, line string, lineNum int) error {
	parts := strings.SplitN(line, " -- ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid job format, expected '@startup name -- command'")
	}

	meta := parseKVTokens(parts[0])
	if len(meta) < 2 {
		return fmt.Errorf("job name is required")
	}

	job := Job{
		Name:    meta[1],
		Command: strings.TrimSpace(parts[1]),
		LineNum: lineNum,
	}

	if job.Command == "" {
		return fmt.Errorf("job command is empty")
	}
	if !isValidJobName(job.Name) {
		return fmt.Errorf("invalid job name %q", job.Name)
	}
	if hasJobName(cfg.Jobs, job.Name) {
		return fmt.Errorf("duplicate job name %q", job.Name)
	}

	switch meta[0] {
	case "@startup":
		job.Run = JobRunStartup
		for _, token := range meta[2:] {
			if token == "keepalive" {
				job.KeepAlive = true
				continue
			}
			return fmt.Errorf("unknown startup option %q", token)
		}
	case "@interval":
		if len(meta) < 3 {
			return fmt.Errorf("interval job requires every=<duration>")
		}
		if len(meta) > 3 {
			return fmt.Errorf("interval job only supports '@interval name every=<duration> -- command'")
		}
		if !strings.HasPrefix(meta[2], "every=") {
			return fmt.Errorf("interval job requires every=<duration>")
		}
		job.Run = JobRunInterval
		job.Every = strings.TrimPrefix(meta[2], "every=")
		if job.Every == "" {
			return fmt.Errorf("interval job requires every=<duration>")
		}
	case "@once":
		if len(meta) > 2 {
			return fmt.Errorf("once job only supports '@once name -- command'")
		}
		job.Run = JobRunOnce
	case "@cache":
		job.Run = JobRunCache
		// Re-split parts[0] with plain Fields so quoting doesn't collapse the
		// check into a single token. For @cache we treat the whole text between
		// `@cache` and ` -- ` as the check expression.
		fields := strings.Fields(parts[0])
		if len(fields) < 2 {
			return fmt.Errorf("@cache requires at least a name")
		}
		job.Name = fields[1]
		if len(fields) > 2 {
			// Multi-word check: the first word doubles as the job name AND is
			// included in the shell check (e.g. `@cache foo -v` -> sh -c 'foo -v').
			job.Check = strings.Join(fields[1:], " ")
		}
	default:
		return fmt.Errorf("unknown job directive %q", meta[0])
	}

	cfg.Jobs = append(cfg.Jobs, job)
	return nil
}

// parseKVTokens splits a whitespace-separated header like
//   @cache hermes check="gh enhance -v"
// into tokens (`@cache`, `hermes`, `check=gh enhance -v`) respecting
// double-quoted values. Quotes are consumed and the inside is kept verbatim.
func parseKVTokens(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if (c == ' ' || c == '\t') && !inQuote {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func hasJobName(jobs []Job, name string) bool {
	for _, job := range jobs {
		if job.Name == name {
			return true
		}
	}
	return false
}

func isValidJobName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '-', '_', '.':
			continue
		default:
			return false
		}
	}
	return true
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
