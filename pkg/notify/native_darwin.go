//go:build darwin

package notify

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// NativeMacOSNotifier delivers notifications via an embedded, signed
// cly-notifier.app bundle that talks to UNUserNotificationCenter. The
// bundle is extracted to ~/Library/Application Support/cly/, codesign-
// verified, and spawned as a long-lived daemon. Action button clicks are
// streamed back via Events().
//
// On any setup failure (placeholder bundle, codesign mismatch, daemon
// crash before ready) Available() returns false so the caller can fall
// back to another backend.
type NativeMacOSNotifier struct {
	mu        sync.Mutex
	cmd       *exec.Cmd
	conn      *daemonConn
	events    chan ActionEvent
	ready     bool
	authorized bool
	startErr  error
}

// NewNativeMacOSNotifier constructs the notifier and attempts to spawn the
// daemon. Errors during setup are stored on the struct and surfaced via
// Available(). The placeholder-bundle case is silent; real failures log to
// stderr.
func NewNativeMacOSNotifier(ctx context.Context) *NativeMacOSNotifier {
	n := &NativeMacOSNotifier{
		events: make(chan ActionEvent, 16),
	}
	if err := n.start(ctx); err != nil {
		n.startErr = err
		if !errPlaceholderBundle(err) {
			fmt.Fprintf(os.Stderr, "notify: native macOS backend unavailable (%v); falling back\n", err)
		}
	}
	return n
}

var errPlaceholderMarker = "embedded notifier bundle is a placeholder"

func errPlaceholderBundle(err error) bool {
	return err != nil && containsString(err.Error(), errPlaceholderMarker)
}

func containsString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func (n *NativeMacOSNotifier) start(ctx context.Context) error {
	if !notifierBundleAvailable() {
		return fmt.Errorf("embedded notifier bundle is a placeholder; run `task build:notifier` or `go generate ./pkg/notify/...`")
	}
	bundlePath, err := installBundle(notifierBundle)
	if err != nil {
		return fmt.Errorf("install bundle: %w", err)
	}
	if err := verifyCodesign(bundlePath); err != nil {
		return fmt.Errorf("codesign verify: %w", err)
	}
	supportDir := filepath.Dir(bundlePath)
	sockPath := filepath.Join(supportDir, fmt.Sprintf("notifier.%d.sock", os.Getpid()))
	_ = os.Remove(sockPath)

	binPath := filepath.Join(bundlePath, "Contents", "MacOS", "cly-notifier")
	cmd := exec.Command(binPath, "--socket", sockPath)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn daemon: %w", err)
	}
	n.cmd = cmd

	// The daemon creates the socket asynchronously after start.
	conn, err := dialWithRetry(sockPath, 3*time.Second)
	if err != nil {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return fmt.Errorf("connect socket: %w", err)
	}
	n.conn = conn

	// Wait for the daemon's "ready" message.
	readyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := n.awaitReady(readyCtx); err != nil {
		_ = conn.Close()
		_ = cmd.Process.Signal(syscall.SIGTERM)
		return fmt.Errorf("daemon never reported ready: %w", err)
	}
	go n.readLoop()
	n.ready = true
	return nil
}

func (n *NativeMacOSNotifier) awaitReady(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		msg, err := n.conn.readMessage()
		if err != nil {
			return err
		}
		if op, _ := msg["op"].(string); op == "ready" {
			n.authorized, _ = msg["authorized"].(bool)
			return nil
		}
	}
}

func (n *NativeMacOSNotifier) readLoop() {
	for {
		msg, err := n.conn.readMessage()
		if err != nil {
			close(n.events)
			return
		}
		op, _ := msg["op"].(string)
		switch op {
		case "action":
			group, _ := msg["group"].(string)
			id, _ := msg["id"].(string)
			n.events <- ActionEvent{Group: group, ActionID: id}
		case "error":
			fmt.Fprintf(os.Stderr, "notify: daemon error: %v\n", msg["message"])
		}
	}
}

// Send dispatches a notification through the daemon.
func (n *NativeMacOSNotifier) Send(ctx context.Context, note Notification) error {
	if !n.Available() {
		return fmt.Errorf("native notifier unavailable: %v", n.startErr)
	}
	actions := make([]map[string]string, 0, len(note.Actions))
	for _, a := range note.Actions {
		actions = append(actions, map[string]string{"id": a.ID, "title": a.Title})
	}
	payload := map[string]any{
		"op":      "send",
		"group":   note.Group,
		"title":   note.Title,
		"body":    note.Message,
		"sound":   note.Sound,
		"actions": actions,
	}
	return n.conn.send(payload)
}

// Available reports whether the daemon is running and ready.
func (n *NativeMacOSNotifier) Available() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.ready && n.conn != nil
}

// Events returns the stream of user action clicks.
func (n *NativeMacOSNotifier) Events() <-chan ActionEvent {
	return n.events
}

// Close shuts down the daemon. Safe to call multiple times.
func (n *NativeMacOSNotifier) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ready = false
	if n.conn != nil {
		_ = n.conn.Close()
		n.conn = nil
	}
	if n.cmd != nil && n.cmd.Process != nil {
		_ = n.cmd.Process.Signal(syscall.SIGTERM)
		go func(c *exec.Cmd) { _ = c.Wait() }(n.cmd)
		n.cmd = nil
	}
	return nil
}

// ----- bundle install + codesign -----------------------------------------

// installBundle extracts the embedded tarball to a hash-stamped directory
// under ~/Library/Application Support/cly/ and returns the path to the .app.
// Identical hashes are reused; older versions are garbage-collected.
func installBundle(data []byte) (string, error) {
	supportDir, err := userSupportDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(supportDir, 0o755); err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:8])
	target := filepath.Join(supportDir, "cly-notifier-"+hash+".app")
	if _, err := os.Stat(target); err == nil {
		go gcOldBundles(supportDir, target)
		return target, nil
	}
	tmp := target + ".extract"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return "", err
	}
	if err := untar(data, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	// The tarball contains a top-level cly-notifier.app/. Move it into place.
	src := filepath.Join(tmp, "cly-notifier.app")
	if _, err := os.Stat(src); err != nil {
		_ = os.RemoveAll(tmp)
		return "", fmt.Errorf("tarball missing cly-notifier.app: %w", err)
	}
	if err := os.Rename(src, target); err != nil {
		_ = os.RemoveAll(tmp)
		return "", err
	}
	_ = os.RemoveAll(tmp)
	go gcOldBundles(supportDir, target)
	return target, nil
}

func userSupportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "cly"), nil
}

func gcOldBundles(supportDir, keep string) {
	entries, err := os.ReadDir(supportDir)
	if err != nil {
		return
	}
	keepBase := filepath.Base(keep)
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || name == keepBase {
			continue
		}
		if filepath.Ext(name) != ".app" || len(name) < len("cly-notifier-") || name[:len("cly-notifier-")] != "cly-notifier-" {
			continue
		}
		_ = os.RemoveAll(filepath.Join(supportDir, name))
	}
}

func untar(gzData []byte, dest string) error {
	gzr, err := gzip.NewReader(newByteReader(gzData))
	if err != nil {
		return err
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		path := filepath.Join(dest, h.Name)
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		case tar.TypeSymlink:
			_ = os.Symlink(h.Linkname, path)
		}
	}
}

// minimal io.Reader over []byte without alloc.
type byteReader struct {
	b []byte
	i int
}

func newByteReader(b []byte) *byteReader { return &byteReader{b: b} }
func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func verifyCodesign(bundle string) error {
	cmd := exec.Command("codesign", "--verify", "--deep", "--strict", bundle)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// ----- socket dialing ----------------------------------------------------

type daemonConn struct {
	c   *os.File // duplicate of the socket fd as an *os.File for blocking I/O
	enc *json.Encoder
	dec *json.Decoder
	mu  sync.Mutex
}

func dialWithRetry(path string, timeout time.Duration) (*daemonConn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		fd, err := socketDial(path)
		if err == nil {
			f := os.NewFile(uintptr(fd), path)
			return &daemonConn{
				c:   f,
				enc: json.NewEncoder(f),
				dec: json.NewDecoder(bufio.NewReader(f)),
			}, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
}

func socketDial(path string) (int, error) {
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return 0, err
	}
	addr := &syscall.SockaddrUnix{Name: path}
	if err := syscall.Connect(fd, addr); err != nil {
		syscall.Close(fd)
		return 0, err
	}
	return fd, nil
}

func (c *daemonConn) send(payload map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enc.Encode(payload)
}

func (c *daemonConn) readMessage() (map[string]any, error) {
	var m map[string]any
	if err := c.dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *daemonConn) Close() error {
	if c.c == nil {
		return nil
	}
	err := c.c.Close()
	c.c = nil
	return err
}
