package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/yurifrl/cly/pkg/style"
)

var gsyncJobs int

// perFolderWorkers is how many files a single folder uploads concurrently.
const perFolderWorkers = 8

// mtimeKey matches the metadata key gsutil uses, so objects previously synced
// by gsutil rsync are recognized as up-to-date instead of re-uploaded.
const mtimeKey = "goog-reserved-file-mtime"

// RegisterGsync adds the visual, parallel-per-folder sync command.
func RegisterGsync(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:     "gsync",
		Aliases: []string{"gs"},
		Short:   "Visually sync each source folder to GCS in parallel (native Go, no gsutil)",
		Long:    "Syncs every top-level folder under the configured source dir to GCS in parallel using the Google Cloud Storage SDK, with a live TUI: syncing folders on top, finished folders collapsed below, and a report written to /tmp.",
		RunE:    runGsync,
	}
	cmd.Flags().IntVarP(&gsyncJobs, "jobs", "j", 4, "Number of folders to sync in parallel")
	parent.AddCommand(cmd)
}

func runGsync(cmd *cobra.Command, args []string) error {
	bucket := getBucket()
	if bucket == "" {
		return fmt.Errorf("GCS bucket not configured. Set it in your config.yaml :\n\nmodules:\n  backup:\n    gcs_bucket: your-bucket-name")
	}

	workdir := getWorkdir()
	folders, looseFiles, err := listWorkdirFolders(workdir)
	if err != nil {
		return err
	}
	if len(folders) == 0 {
		return fmt.Errorf("no folders found under %s", workdir)
	}
	if gsyncJobs < 1 {
		gsyncJobs = 1
	}

	ctx := context.Background()
	if !hasADC() {
		fmt.Println(style.YellowStyle.Render("⚠️  No application-default credentials found. Launching login..."))
		if err := adcLogin(); err != nil {
			return fmt.Errorf("application-default login failed: %w", err)
		}
	}
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	defer client.Close()

	account := adcAccount()

	m := newGsyncModel(bucket, account, workdir, folders, looseFiles)
	p := tea.NewProgram(m)
	go orchestrate(ctx, p, client, bucket, workdir, folders, looseFiles, gsyncJobs)
	_, err = p.Run()
	return err
}

func listWorkdirFolders(workdir string) (folders []string, looseFiles []string, err error) {
	entries, err := os.ReadDir(workdir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read %s: %w", workdir, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			folders = append(folders, e.Name())
		} else {
			looseFiles = append(looseFiles, e.Name())
		}
	}
	sort.Strings(folders)
	return folders, looseFiles, nil
}

// gsyncIgnorePath is the user-editable, gitignore-style ignore file for gsync.
func gsyncIgnorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config/cly/gsyncignore")
}

// readGsyncIgnore returns the glob patterns from ~/.config/cly/gsyncignore,
// falling back to the built-in defaults when the file is missing or empty.
// Blank lines and lines starting with # are ignored.
func readGsyncIgnore() []string {
	data, err := os.ReadFile(gsyncIgnorePath())
	if err != nil {
		return defaultIgnoreGlobs()
	}
	var globs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		globs = append(globs, line)
	}
	if len(globs) == 0 {
		return defaultIgnoreGlobs()
	}
	return globs
}

// ignoreRegexp compiles the ignore globs into a single matcher applied to
// each path relative to the source dir.
func ignoreRegexp() *regexp.Regexp {
	globs := readGsyncIgnore()
	frags := make([]string, 0, len(globs))
	for _, g := range globs {
		frags = append(frags, globToRegex(g))
	}
	return regexp.MustCompile(strings.Join(frags, "|"))
}

// globToRegex converts one gitignore-style glob into a regex fragment.
// Semantics: `*`=any run except /, `**`=any run, `?`=one non-/ char,
// trailing `/`=directory (matches it and everything under it), leading `/`
// anchors to the source root, otherwise the pattern matches at any depth.
func globToRegex(g string) string {
	dirOnly := strings.HasSuffix(g, "/")
	g = strings.TrimSuffix(g, "/")
	anchored := strings.HasPrefix(g, "/")
	g = strings.TrimPrefix(g, "/")

	var b strings.Builder
	for i := 0; i < len(g); i++ {
		c := g[i]
		switch c {
		case '*':
			if i+1 < len(g) && g[i+1] == '*' {
				b.WriteString(".*")
				i++
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '[', ']', '{', '}', '^', '$', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}

	prefix := "(^|/)"
	if anchored {
		prefix = "^"
	}
	suffix := "(/|$)"
	if dirOnly {
		suffix = "/"
	}
	return prefix + b.String() + suffix
}

// defaultIgnoreGlobs is the built-in ignore set used when gsyncignore is absent.
func defaultIgnoreGlobs() []string {
	return []string{
		"node_modules/", "__pycache__/", "*.pyc", "*.pyo", "*.pyd",
		".DS_Store", "._.DS_Store", ".terraform/", ".terragrunt-cache/",
		".vscode/", ".idea/", ".npm/", ".yarn/",
		"venv/", ".venv/", "env/", ".env/", ".direnv/",
		"dist/", "build/", "target/", "bin/", "obj/", "out/",
		"*.bak", "*.bkp", "*.log", "*.tmp", "*.temp", "tmp/", "temp/",
		"*.swp", "*.swo", "*.swn", "*~", ".#*",
		".cache/", ".pytest_cache/", ".mypy_cache/", ".ruff_cache/",
		"coverage/", ".coverage", "htmlcov/", ".tox/", ".eggs/", "*.egg-info/",
		"*.class", "*.o", "*.so", "*.dylib", "*.dll", "*.exe",
	}
}

// ---- credentials -----------------------------------------------------------

func hasADC() bool {
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		return true
	}
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".config/gcloud/application_default_credentials.json"))
	return err == nil
}

func adcLogin() error {
	c := exec.Command("gcloud", "auth", "application-default", "login")
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func adcAccount() string {
	out, err := exec.Command("gcloud", "config", "get-value", "account").Output()
	if err != nil {
		return "application-default credentials"
	}
	return strings.TrimSpace(string(out))
}

// ---- messages --------------------------------------------------------------

type folderStartMsg int
type folderTotalMsg struct {
	idx   int
	total int
}
type folderLineMsg struct {
	idx  int
	op   operationType
	file string // relative path, only for uploads
}
type folderDoneMsg struct {
	idx int
	err bool
}
type allDoneMsg struct{ reportPath string }

type folderResult struct {
	name     string
	uploaded int
	skipped  int
	errors   int
	hadErr   bool
	errLines []string
}

// ---- orchestration ---------------------------------------------------------

func orchestrate(ctx context.Context, p *tea.Program, client *storage.Client, bucket, workdir string, folders, looseFiles []string, jobs int) {
	re := ignoreRegexp()
	bkt := client.Bucket(bucket)
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	results := make([]folderResult, len(folders))

	for i, name := range folders {
		wg.Add(1)
		go func(idx int, folder string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			p.Send(folderStartMsg(idx))
			res := syncFolder(ctx, p, idx, bkt, workdir, folder, re)
			res.name = folder
			results[idx] = res
			p.Send(folderDoneMsg{idx: idx, err: res.hadErr})
		}(i, name)
	}

	wg.Wait()
	reportPath := writeReport(bucket, workdir, results, looseFiles)
	p.Send(allDoneMsg{reportPath: reportPath})
}

type uploadJob struct {
	path string      // absolute local path
	rel  string      // path relative to workdir (== object name, e.g. "cly/main.go")
	info fs.FileInfo // local stat
}

// syncFolder walks one top-level folder, applies exclusions, then uploads
// changed files to GCS using a small worker pool.
func syncFolder(ctx context.Context, p *tea.Program, idx int, bkt *storage.BucketHandle, workdir, folder string, re *regexp.Regexp) folderResult {
	res := folderResult{}
	src := filepath.Join(workdir, folder)

	var jobsList []uploadJob
	var symlinkSkips int
	_ = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		rel, rerr := filepath.Rel(workdir, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if re.MatchString(rel + "/") {
				return filepath.SkipDir // don't descend into excluded dirs (e.g. node_modules)
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			symlinkSkips++ // match gsutil -e: symlinks are skipped
			return nil
		}
		if re.MatchString(rel) {
			return nil // excluded file
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		jobsList = append(jobsList, uploadJob{path: path, rel: rel, info: info})
		return nil
	})

	res.skipped = symlinkSkips
	p.Send(folderTotalMsg{idx: idx, total: len(jobsList)})

	var mu sync.Mutex
	wsem := make(chan struct{}, perFolderWorkers)
	var wg sync.WaitGroup
	for _, j := range jobsList {
		wg.Add(1)
		go func(j uploadJob) {
			defer wg.Done()
			wsem <- struct{}{}
			defer func() { <-wsem }()

			obj := bkt.Object(j.rel)
			display := strings.TrimPrefix(j.rel, folder+"/")

			if !needsUpload(ctx, obj, j.info) {
				mu.Lock()
				res.skipped++
				mu.Unlock()
				p.Send(folderLineMsg{idx: idx, op: opSkipped})
				return
			}
			if err := uploadFile(ctx, obj, j.path, j.info); err != nil {
				mu.Lock()
				res.errors++
				res.hadErr = true
				if len(res.errLines) < 20 {
					res.errLines = append(res.errLines, fmt.Sprintf("%s: %v", j.rel, err))
				}
				mu.Unlock()
				p.Send(folderLineMsg{idx: idx, op: opError})
				return
			}
			mu.Lock()
			res.uploaded++
			mu.Unlock()
			p.Send(folderLineMsg{idx: idx, op: opUpload, file: display})
		}(j)
	}
	wg.Wait()
	return res
}

// needsUpload reports whether the local file differs from the stored object,
// comparing size and mtime (rsync-style; interops with gsutil's mtime key).
func needsUpload(ctx context.Context, obj *storage.ObjectHandle, info fs.FileInfo) bool {
	attrs, err := obj.Attrs(ctx)
	if errors.Is(err, storage.ErrObjectNotExist) {
		return true
	}
	if err != nil {
		return true // on any lookup error, attempt the upload
	}
	if attrs.Size != info.Size() {
		return true
	}
	if m := attrs.Metadata[mtimeKey]; m != "" {
		if sec, e := strconv.ParseInt(m, 10, 64); e == nil {
			return sec != info.ModTime().Unix()
		}
	}
	return false // same size, no usable mtime -> treat as up-to-date
}

func uploadFile(ctx context.Context, obj *storage.ObjectHandle, path string, info fs.FileInfo) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := obj.NewWriter(ctx)
	w.Metadata = map[string]string{mtimeKey: strconv.FormatInt(info.ModTime().Unix(), 10)}
	if _, err := io.Copy(w, f); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

// ---- model -----------------------------------------------------------------

type folderState struct {
	name     string
	status   int
	total    int
	uploaded int
	skipped  int
	errors   int
	current  []string
}

const (
	stPending = iota
	stSyncing
	stDone
	stErrored
)

type gsyncModel struct {
	bucket     string
	account    string
	workdir    string
	folders    []*folderState
	looseFiles []string
	spinner    spinner.Model
	progress   progress.Model
	viewport   viewport.Model
	ready      bool
	width      int
	startedAt  time.Time
	finished   bool
	reportPath string

	totalUp   int
	totalSkip int
	totalErr  int

	recentDone []string // completed file paths (folder-prefixed), newest appended last
}

var (
	gsHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	gsLabel   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	gsName    = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	gsNameDim = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	gsFile    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	gsUpArrow = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
)

func newGsyncModel(bucket, account, workdir string, folders, looseFiles []string) gsyncModel {
	fs := make([]*folderState, len(folders))
	for i, n := range folders {
		fs[i] = &folderState{name: n}
	}
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	pr := progress.New(progress.WithDefaultBlend(), progress.WithWidth(30), progress.WithoutPercentage())
	return gsyncModel{
		bucket:     bucket,
		account:    account,
		workdir:    workdir,
		folders:    fs,
		looseFiles: looseFiles,
		spinner:    sp,
		progress:   pr,
		startedAt:  time.Now(),
	}
}

func (m gsyncModel) Init() tea.Cmd { return m.spinner.Tick }

func (m gsyncModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		vpHeight := msg.Height - m.chromeHeight()
		if vpHeight < 3 {
			vpHeight = 3
		}
		if !m.ready {
			m.viewport = viewport.New(viewport.WithWidth(msg.Width), viewport.WithHeight(vpHeight))
			m.ready = true
		} else {
			m.viewport.SetWidth(msg.Width)
			m.viewport.SetHeight(vpHeight)
		}
		m.progress.SetWidth(min(40, msg.Width-30))
		m.refreshViewport()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		m.refreshViewport()
		return m, cmd
	case folderStartMsg:
		m.folders[int(msg)].status = stSyncing
		m.refreshViewport()
	case folderTotalMsg:
		m.folders[msg.idx].total = msg.total
		m.refreshViewport()
	case folderLineMsg:
		f := m.folders[msg.idx]
		switch msg.op {
		case opUpload:
			f.uploaded++
			m.totalUp++
			if msg.file != "" {
				f.current = append(f.current, msg.file)
				if len(f.current) > 3 {
					f.current = f.current[len(f.current)-3:]
				}
				m.recentDone = append(m.recentDone, f.name+"/"+msg.file)
				if len(m.recentDone) > 300 {
					m.recentDone = m.recentDone[len(m.recentDone)-300:]
				}
			}
		case opSkipped:
			f.skipped++
			m.totalSkip++
		case opError:
			f.errors++
			m.totalErr++
		}
		m.refreshViewport()
	case folderDoneMsg:
		f := m.folders[msg.idx]
		f.current = nil
		if msg.err || f.errors > 0 {
			f.status = stErrored
		} else {
			f.status = stDone
		}
		m.refreshViewport()
	case allDoneMsg:
		m.finished = true
		m.reportPath = msg.reportPath
		m.refreshViewport()
		return m, nil
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m gsyncModel) chromeHeight() int { return 6 }

func (m *gsyncModel) refreshViewport() {
	if !m.ready {
		return
	}
	m.viewport.SetContent(m.bodyContent())
}

func (m gsyncModel) doneCount() int {
	n := 0
	for _, f := range m.folders {
		if f.status == stDone || f.status == stErrored {
			n++
		}
	}
	return n
}

func (m gsyncModel) bodyContent() string {
	var b strings.Builder

	var active, done []*folderState
	for _, f := range m.folders {
		switch f.status {
		case stSyncing:
			active = append(active, f)
		case stDone, stErrored:
			done = append(done, f)
		}
	}
	pending := len(m.folders) - len(active) - len(done)

	b.WriteString(gsLabel.Render("SYNCING") + "\n")
	if len(active) == 0 && pending > 0 {
		b.WriteString(gsNameDim.Render(fmt.Sprintf("  %d folder(s) queued…", pending)) + "\n")
	}
	for _, f := range active {
		prog := ""
		if f.total > 0 {
			prog = gsNameDim.Render(fmt.Sprintf("[%d/%d]", f.uploaded+f.skipped, f.total)) + " "
		} else {
			prog = gsNameDim.Render("[scanning…]") + " "
		}
		fmt.Fprintf(&b, "  %s %s  %s%s\n",
			m.spinner.View(),
			gsName.Render(f.name+"/"),
			prog,
			counts(f))
		for _, cf := range f.current {
			fmt.Fprintf(&b, "      %s %s\n", gsUpArrow.Render("↑"), gsFile.Render(truncate(f.name+"/"+cf, m.width-8)))
		}
	}
	if pending > 0 && len(active) > 0 {
		b.WriteString(gsNameDim.Render(fmt.Sprintf("  + %d queued", pending)) + "\n")
	}

	b.WriteString("\n" + gsLabel.Render("DONE") + "\n")
	if len(done) == 0 && len(m.recentDone) == 0 {
		b.WriteString(gsNameDim.Render("  (nothing finished yet)") + "\n")
	}
	for _, f := range done {
		icon := style.GreenStyle.Render("✓")
		if f.status == stErrored {
			icon = style.RedStyle.Render("✗")
		}
		fmt.Fprintf(&b, "  %s %s  %s\n", icon, gsNameDim.Render(f.name+"/"), counts(f))
	}
	// live stream of completed files, newest first
	for i := len(m.recentDone) - 1; i >= 0; i-- {
		fmt.Fprintf(&b, "  %s %s\n", style.GreenStyle.Render("↑"), gsFile.Render(truncate(m.recentDone[i], m.width-4)))
	}
	return b.String()
}

func counts(f *folderState) string {
	parts := []string{style.BlueStyle.Render(fmt.Sprintf("%d↑", f.uploaded))}
	if f.skipped > 0 {
		parts = append(parts, style.YellowStyle.Render(fmt.Sprintf("%d⏭", f.skipped)))
	}
	if f.errors > 0 {
		parts = append(parts, style.RedStyle.Render(fmt.Sprintf("%d✗", f.errors)))
	}
	return strings.Join(parts, "  ")
}

func (m gsyncModel) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if !m.ready {
		v.SetContent("\n  Initializing gsync…")
		return v
	}

	title := gsHeader.Render("gsync")
	head := fmt.Sprintf("%s  %s → gs://%s\n%s %s",
		title,
		gsNameDim.Render(m.workdir),
		m.bucket,
		style.GreenStyle.Render("✓"),
		gsNameDim.Render(m.account))

	totalFolders := len(m.folders)
	discovered, processed := 0, m.totalUp+m.totalSkip+m.totalErr
	for _, f := range m.folders {
		discovered += f.total
	}
	pct := 0.0
	if discovered > 0 {
		pct = float64(processed) / float64(discovered)
		if pct > 1 {
			pct = 1
		}
	}
	phase := m.spinner.View() + " syncing"
	if m.finished {
		phase = style.GreenStyle.Render("✓ done")
	}
	bar := fmt.Sprintf("%s  %s  %s  %s  %s",
		phase,
		m.progress.ViewAs(pct),
		gsNameDim.Render(fmt.Sprintf("%d/%d files", processed, discovered)),
		gsNameDim.Render(fmt.Sprintf("%d/%d folders", m.doneCount(), totalFolders)),
		gsNameDim.Render(elapsed(m.startedAt)))

	foot := gsNameDim.Render("↑↓/mouse scroll · q quit")
	if m.finished && m.reportPath != "" {
		foot = gsNameDim.Render("↑↓ scroll · q quit · report → " + m.reportPath)
	} else {
		foot += gsNameDim.Render(fmt.Sprintf("   ·   %d↑ %d⏭ %d✗", m.totalUp, m.totalSkip, m.totalErr))
	}

	v.SetContent(head + "\n" + bar + "\n\n" + m.viewport.View() + "\n" + foot)
	return v
}

func elapsed(start time.Time) string {
	return time.Since(start).Round(time.Second).String()
}

func truncate(s string, max int) string {
	if max < 4 || len(s) <= max {
		return s
	}
	return "…" + s[len(s)-(max-1):]
}

// ---- report ----------------------------------------------------------------

func writeReport(bucket, workdir string, results []folderResult, looseFiles []string) string {
	ts := time.Now().Format("2006-01-02-1504")
	path := filepath.Join(os.TempDir(), "gsync-"+ts+".md")
	f, err := os.Create(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var totUp, totSkip, totErr int
	for _, r := range results {
		totUp += r.uploaded
		totSkip += r.skipped
		totErr += r.errors
	}

	fmt.Fprintf(f, "# gsync report — %s\n\n", time.Now().Format(time.RFC1123))
	fmt.Fprintf(f, "- Source: `%s`\n- Bucket: `gs://%s`\n- Folders: %d\n", workdir, bucket, len(results))
	fmt.Fprintf(f, "- Totals: %d uploaded · %d skipped · %d errors\n\n", totUp, totSkip, totErr)

	fmt.Fprintf(f, "## Per-folder\n\n| folder | uploaded | skipped | errors |\n|---|---:|---:|---:|\n")
	for _, r := range results {
		fmt.Fprintf(f, "| %s/ | %d | %d | %d |\n", r.name, r.uploaded, r.skipped, r.errors)
	}

	hasErrors := false
	for _, r := range results {
		if len(r.errLines) > 0 {
			hasErrors = true
			break
		}
	}
	if hasErrors {
		fmt.Fprintf(f, "\n## Errors\n\n")
		for _, r := range results {
			if len(r.errLines) == 0 {
				continue
			}
			fmt.Fprintf(f, "### %s/\n\n```\n", r.name)
			for _, l := range r.errLines {
				fmt.Fprintf(f, "%s\n", l)
			}
			fmt.Fprintf(f, "```\n\n")
		}
	}

	if len(looseFiles) > 0 {
		fmt.Fprintf(f, "\n## Skipped: loose top-level files\n\nFiles directly under `%s` are not synced (only folders are). Move them into a folder to include them:\n\n", workdir)
		for _, name := range looseFiles {
			fmt.Fprintf(f, "- `%s`\n", name)
		}
	}

	fmt.Fprintf(f, "\n## Skipped: ignore globs\n\nThese `~/.config/cly/gsyncignore` patterns were excluded and never uploaded:\n\n")
	for _, p := range readGsyncIgnore() {
		fmt.Fprintf(f, "- `%s`\n", p)
	}
	fmt.Fprintf(f, "\n## Notes\n\n")
	fmt.Fprintf(f, "- \"skipped\" counts symlinks (not followed) plus files already up-to-date in the bucket (matching size + mtime).\n")
	fmt.Fprintf(f, "- Uploads set the `%s` metadata key so later runs skip unchanged files.\n", mtimeKey)
	return path
}
