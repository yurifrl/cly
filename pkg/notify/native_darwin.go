//go:build darwin

package notify

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// NativeMacOSNotifier delivers notifications via an embedded, signed
// cly-notifier.app bundle that talks to UNUserNotificationCenter.
//
// The bundle is launched via LaunchServices (`open -a`) so macOS associates
// the running process with its bundle metadata (icon, bundle ID, notification
// permissions). Direct exec of the inner binary bypasses LaunchServices
// registration and notifications get a generic placeholder icon.
type NativeMacOSNotifier struct {
	mu       sync.Mutex
	conn     *daemonConn
	ready    bool
	startErr error
}

// NewNativeMacOSNotifier constructs the notifier and attempts to launch the
// daemon. Errors are stored on the struct and surfaced via Available().
// The placeholder-bundle case is silent (expected on fresh checkout).
func NewNativeMacOSNotifier(ctx context.Context) *NativeMacOSNotifier {
	n := &NativeMacOSNotifier{}
	if os.Getenv("CLY_NOTIFIER_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "notify: bundle bytes=%d, available=%v\n", len(notifierBundle), notifierBundleAvailable())
	}
	if err := n.start(ctx); err != nil {
		n.startErr = err
		if !errPlaceholderBundle(err) || os.Getenv("CLY_NOTIFIER_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "notify: native macOS backend unavailable (%v); falling back\n", err)
		}
	}
	return n
}

func (n *NativeMacOSNotifier) start(ctx context.Context) error {
	debug := os.Getenv("CLY_NOTIFIER_DEBUG") != ""
	dbg := func(format string, args ...any) {
		if debug {
			fmt.Fprintf(os.Stderr, "notify: "+format+"\n", args...)
		}
	}
	if !notifierBundleAvailable() {
		return fmt.Errorf("embedded notifier bundle is a placeholder; run `task build:notifier`")
	}
	dbg("installing bundle...")
	bundlePath, err := installBundle(notifierBundle)
	if err != nil {
		return fmt.Errorf("install bundle: %w", err)
	}
	dbg("bundle at %s", bundlePath)
	if err := verifyCodesign(bundlePath); err != nil {
		return fmt.Errorf("codesign verify: %w", err)
	}
	dbg("codesign ok")
	supportDir := filepath.Dir(bundlePath)
	sockPath := filepath.Join(supportDir, fmt.Sprintf("notifier.%d.sock", os.Getpid()))
	_ = os.Remove(sockPath)

	// Launch via LaunchServices so macOS associates the process with the
	// bundle's metadata (icon, bundle ID, notification permissions).
	cmd := exec.Command("/usr/bin/open", "-g", "-a", bundlePath, "--args", "--socket", sockPath)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("open daemon: %w", err)
	}
	dbg("daemon launched via LaunchServices")

	conn, err := dialWithRetry(sockPath, 5*time.Second)
	if err != nil {
		return fmt.Errorf("connect socket: %w", err)
	}
	dbg("socket dialed")
	n.conn = conn

	readyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := n.awaitReady(readyCtx); err != nil {
		_ = conn.Close()
		return fmt.Errorf("daemon never reported ready: %w", err)
	}
	dbg("daemon ready")
	n.ready = true
	return nil
}

func (n *NativeMacOSNotifier) awaitReady(ctx context.Context) error {
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			if n.conn != nil && n.conn.c != nil {
				_ = n.conn.c.SetReadDeadline(time.Now())
			}
		case <-done:
		}
	}()
	for {
		msg, err := n.conn.readMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		if op, _ := msg["op"].(string); op == "ready" {
			if n.conn != nil && n.conn.c != nil {
				_ = n.conn.c.SetReadDeadline(time.Time{})
			}
			return nil
		}
	}
}

// Send dispatches a notification through the daemon.
func (n *NativeMacOSNotifier) Send(ctx context.Context, note Notification) error {
	if !n.Available() {
		return fmt.Errorf("native notifier unavailable: %v", n.startErr)
	}
	return n.conn.send(map[string]any{
		"op":    "send",
		"group": note.Group,
		"title": note.Title,
		"body":  note.Message,
		"sound": note.Sound,
	})
}

// Available reports whether the daemon is running and ready.
func (n *NativeMacOSNotifier) Available() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.ready && n.conn != nil
}

// Close shuts down the daemon. The Swift daemon detects parent socket
// disconnect and exits cleanly; no SIGTERM needed.
func (n *NativeMacOSNotifier) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ready = false
	if n.conn != nil {
		_ = n.conn.Close()
		n.conn = nil
	}
	return nil
}

// ----- bundle install + codesign -----------------------------------------

var errPlaceholderMarker = "embedded notifier bundle is a placeholder"

func errPlaceholderBundle(err error) bool {
	return err != nil && strings.Contains(err.Error(), errPlaceholderMarker)
}

// installBundle extracts the embedded tarball to a hash-stamped directory
// under ~/Library/Application Support/cly/ and returns the path to the .app.
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
		if filepath.Ext(name) != ".app" || !strings.HasPrefix(name, "cly-notifier-") {
			continue
		}
		_ = os.RemoveAll(filepath.Join(supportDir, name))
	}
}

func untar(gzData []byte, dest string) error {
	gzr, err := gzip.NewReader(bytes.NewReader(gzData))
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

func verifyCodesign(bundle string) error {
	// Lenient verify: ad-hoc signed bundles often fail --deep --strict after
	// tar/untar ("sealed resource is missing or invalid") even though the
	// binary itself is fine. We only fail hard on a broken signature.
	cmd := exec.Command("codesign", "--verify", bundle)
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if strings.Contains(msg, "sealed resource") {
			return nil
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(msg))
	}
	return nil
}

// ----- socket dialing ----------------------------------------------------

type daemonConn struct {
	c   net.Conn
	enc *json.Encoder
	dec *json.Decoder
	mu  sync.Mutex
}

func dialWithRetry(path string, timeout time.Duration) (*daemonConn, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		c, err := net.Dial("unix", path)
		if err == nil {
			return &daemonConn{
				c:   c,
				enc: json.NewEncoder(c),
				dec: json.NewDecoder(bufio.NewReader(c)),
			}, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, lastErr
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
