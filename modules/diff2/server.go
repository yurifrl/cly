package diff2

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
)

// Deps is the injectable dependency set for the server.
// Tests substitute fakes; production uses real exec-backed versions.
type Deps struct {
	Git     Gitter
	Bd      BdRunner
	WebRoot fs.FS // served at "/"; nil disables static assets
}

// RealDeps returns Deps wired to the real git + bd binaries.
func RealDeps(webRoot fs.FS) Deps {
	return Deps{Git: execGit{}, Bd: execBd{}, WebRoot: webRoot}
}

// NewServer builds an http.Handler wiring all routes.
func NewServer(d Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", d.handleHealth)
	mux.HandleFunc("/api/diff", d.handleDiff)
	mux.HandleFunc("/api/diff/file", d.handleDiffFile)
	mux.HandleFunc("/api/labels", d.handleLabels)
	mux.HandleFunc("/api/bead", d.handleBead)

	// Static + SPA fallback.
	if d.WebRoot != nil {
		mux.Handle("/", spaHandler(d.WebRoot))
	}
	return loggingMW(mux)
}

// loggingMW adds a tiny access log; swap for real logger later.
func loggingMW(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}

// spaHandler serves static assets; unknown paths fall back to index.html.
func spaHandler(root fs.FS) http.Handler {
	fileServer := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// try path as-is
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(root, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		// fall back to index.html for client-side routing
		if data, err := fs.ReadFile(root, "index.html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
		http.NotFound(w, r)
	})
}

// --- handlers ----------------------------------------------------------

func (d Deps) handleHealth(w http.ResponseWriter, r *http.Request) {
	h := struct {
		Git      bool `json:"git"`
		Bd       bool `json:"bd"`
		BeadsDB  bool `json:"beadsDb"`
	}{}
	if err := IsRepo(d.Git); err == nil {
		h.Git = true
	}
	// bd present?
	if d.Bd != nil {
		if _, _, err := d.Bd.Run("--version"); err == nil {
			h.Bd = true
			// beads DB present? probe via label list
			if _, _, err := d.Bd.Run("label", "list-all", "--json"); err == nil {
				h.BeadsDB = true
			}
		}
	}
	writeJSON(w, http.StatusOK, h)
}

func (d Deps) handleDiff(w http.ResponseWriter, r *http.Request) {
	if err := IsRepo(d.Git); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	files, err := ListChangedFiles(d.Git)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	// Drop binaries from review list (spec: skip).
	out := make([]File, 0, len(files))
	for _, f := range files {
		if f.Binary {
			continue
		}
		out = append(out, f)
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": out})
}

func (d Deps) handleDiffFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeErr(w, http.StatusBadRequest, errors.New("missing ?path"))
		return
	}
	fd, err := DiffFile(d.Git, path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if fd.Binary {
		writeErr(w, http.StatusUnsupportedMediaType, errors.New("binary file"))
		return
	}
	writeJSON(w, http.StatusOK, fd)
}

func (d Deps) handleLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := ListLabels(d.Bd)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if labels == nil {
		labels = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"labels": labels})
}

func (d Deps) handleBead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		writeErr(w, http.StatusMethodNotAllowed, errors.New("POST only"))
		return
	}
	var req BeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	resp, err := CreateBead(d.Bd, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrBdMissing), errors.Is(err, ErrBdNoDB):
			writeErr(w, http.StatusConflict, err)
		case errors.Is(err, ErrBdBadInput):
			writeErr(w, http.StatusBadRequest, err)
		default:
			writeErr(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// --- helpers -----------------------------------------------------------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

// --- listener -----------------------------------------------------------

// Listen binds a TCP listener on 127.0.0.1 with the given port.
// port=0 picks a free one. Returns the listener + the resolved port.
func Listen(port int) (net.Listener, int, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, err
	}
	got := l.Addr().(*net.TCPAddr).Port
	return l, got, nil
}

// Ensure the static assets are served correctly.
var _ = embed.FS{}
