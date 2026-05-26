// fork.go forks a pi session JSONL file: copy bytes verbatim, then
// rewrite every occurrence of the source's session id to a fresh
// UUID v7. Atomic via target.tmp + rename.
//
// Why bytes.ReplaceAll instead of a JSON re-encoder: pi session ids
// are 122-bit UUIDs whose byte sequence cannot collide with anything
// else in the file. Replacing them as a literal substring is correct
// and preserves all formatting / unknown fields pi may add later.
package piwrap

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/google/uuid"
)

// sessionIDRegex extracts the top-level "sessionId":"<uuid>" from the
// first JSONL line. Tolerates whitespace around the colon.
var sessionIDRegex = regexp.MustCompile(`"sessionId"\s*:\s*"([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})"`)

// forkHeaderScanLimit caps how much of the source we scan for the
// sessionId (first JSONL entry should be well under this).
const forkHeaderScanLimit = 64 * 1024

// forkSession copies src to dst with a fresh top-level session id.
// Atomic on POSIX via dst.tmp + rename.
func forkSession(src, dst string) error {
	srcBytes, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	oldID, err := extractSessionID(srcBytes)
	if err != nil {
		return err
	}

	newID := uuid.Must(uuid.NewV7()).String()
	if newID == oldID {
		// Statistically impossible; regenerate once and fail hard if
		// the universe is broken.
		newID = uuid.Must(uuid.NewV7()).String()
		if newID == oldID {
			return errors.New("uuid generator returned the source id twice")
		}
	}

	patched := bytes.ReplaceAll(srcBytes, []byte(oldID), []byte(newID))

	tmp := dst + ".tmp"
	if err := writeAtomic(tmp, dst, patched); err != nil {
		return err
	}
	return nil
}

// extractSessionID scans the first forkHeaderScanLimit bytes of buf
// for "sessionId":"<uuid>" and returns the captured uuid.
func extractSessionID(buf []byte) (string, error) {
	scan := buf
	if len(scan) > forkHeaderScanLimit {
		scan = scan[:forkHeaderScanLimit]
	}
	m := sessionIDRegex.FindSubmatch(scan)
	if m == nil {
		return "", errors.New("no_session_id_in_source")
	}
	return string(m[1]), nil
}

// writeAtomic writes buf to tmp, fsyncs, then renames to final.
// Removes tmp on any failure path.
func writeAtomic(tmp, final string, buf []byte) error {
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open tmp: %w", err)
	}
	if _, err := f.Write(buf); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("fsync tmp: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename tmp: %w", err)
	}
	return nil
}

// copyFile is exposed for tests — performs a streaming copy. Not
// used by forkSession (which reads in full to do bytes.ReplaceAll).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
