package googlex

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/oauth2"

	"github.com/NSXBet/nsx-cli/shared/config"
	"github.com/NSXBet/nsx-cli/shared/interact"
)

type authServer struct {
	server      *http.Server
	AuthCode    chan string
	state       string
	oauthConfig *oauth2.Config
}

var (
	googleClientID     string
	googleClientSecret string
)

func ClientID() string {
	if googleClientID == "" {
		googleClientID = os.Getenv("GOOGLE_CLIENT_ID")
	}
	interact.Debug("googleClientID: %s", googleClientID)
	return googleClientID
}

func ClientSecret() string {
	if googleClientSecret == "" {
		return os.Getenv("GOOGLE_CLIENT_SECRET")
	}
	return googleClientSecret
}

var tokenPath = filepath.Join(config.BaseFolder(), "token.json")

func TokenPath() string {
	return tokenPath
}

func EnsureConfigDir() error {
	return os.MkdirAll(config.BaseFolder(), 0o700)
}

func NewAuthServer(config *oauth2.Config, state string) *authServer {
	auth := &authServer{
		AuthCode:    make(chan string, 1),
		state:       state,
		oauthConfig: config,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", auth.handleCallback)

	auth.server = &http.Server{
		Addr:    ":0", // Let the OS choose an available port
		Handler: mux,
	}

	return auth
}

func (a *authServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	receivedState := r.URL.Query().Get("state")
	if receivedState != a.state {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Code not found", http.StatusBadRequest)
		return
	}

	html := `
		<html>
			<body style="display: flex; justify-content: center; align-items: center; height: 100vh; font-family: sans-serif;">
				<div style="text-align: center;">
					<h1 style="color:rgb(10, 50, 112);">Authorization Successful!</h1>
					<p>You can close this window and return to the terminal.</p>
				</div>
			</body>
		</html>
	`
	w.Header().Set("Content-Type", "text/html")
	if _, err := fmt.Fprint(w, html); err != nil {
		interact.Error("Failed to write HTML: %v", err)
	}
	a.AuthCode <- code
}

func (a *authServer) Start() (string, error) {
	// Start server
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", fmt.Errorf("failed to listen: %v", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)
	a.oauthConfig.RedirectURL = redirectURL

	go func() {
		if err := a.server.Serve(listener); err != nil {
			if err != http.ErrServerClosed {
				interact.Error("Failed to serve: %v", err)
				os.Exit(1)
			}
		}
	}()
	return redirectURL, nil
}

func (a *authServer) Shutdown(ctx context.Context) {
	if err := a.server.Shutdown(ctx); err != nil {
		interact.Error("Failed to shutdown: %v", err)
		os.Exit(1)
	}
}

func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func SaveToken(token *oauth2.Token) error {
	f, err := os.OpenFile(tokenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %v", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			interact.Error("Failed to close file: %v", err)
		}
	}()

	return json.NewEncoder(f).Encode(token)
}
