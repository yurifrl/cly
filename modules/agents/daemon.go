package agents

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// Daemon wraps the sync engine with a Unix socket server.
type Daemon struct {
	syncFn   func() (*ReconcileResult, error)
	listener net.Listener
	sockPath string
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// DaemonCommand is a request from client to daemon.
type DaemonCommand struct {
	Cmd string `json:"cmd"`
}

// DaemonResponse is a response from daemon to client.
type DaemonResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// NewDaemon creates a daemon that runs syncFn on file changes.
func NewDaemon(sockPath string, syncFn func() (*ReconcileResult, error)) *Daemon {
	return &Daemon{
		syncFn:   syncFn,
		sockPath: sockPath,
	}
}

// Run starts the daemon. Blocks until ctx is cancelled.
func (d *Daemon) Run(ctx context.Context) error {
	ctx, d.cancel = context.WithCancel(ctx)

	if err := cleanStaleSocket(d.sockPath); err != nil {
		return err
	}

	l, err := net.Listen("unix", d.sockPath)
	if err != nil {
		return err
	}
	d.listener = l

	defer os.Remove(d.sockPath)
	defer d.listener.Close()

	d.wg.Add(1)
	go d.acceptLoop(ctx)

	<-ctx.Done()
	d.listener.Close()
	d.wg.Wait()
	return nil
}

func (d *Daemon) acceptLoop(ctx context.Context) {
	defer d.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if ul, ok := d.listener.(*net.UnixListener); ok {
			ul.SetDeadline(time.Now().Add(500 * time.Millisecond))
		}

		conn, err := d.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		go d.handleConn(conn)
	}
}

func (d *Daemon) handleConn(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		d.writeResponse(conn, DaemonResponse{OK: false, Error: "read error"})
		return
	}

	var cmd DaemonCommand
	if err := json.Unmarshal(line, &cmd); err != nil {
		d.writeResponse(conn, DaemonResponse{OK: false, Error: "invalid json"})
		return
	}

	switch cmd.Cmd {
	case "status":
		d.writeResponse(conn, DaemonResponse{
			OK:   true,
			Data: map[string]interface{}{"running": true},
		})
	case "sync":
		result, err := d.syncFn()
		if err != nil {
			d.writeResponse(conn, DaemonResponse{OK: false, Error: err.Error()})
			return
		}
		d.writeResponse(conn, DaemonResponse{
			OK:   true,
			Data: map[string]interface{}{"written": result.Written, "skipped": result.Skipped},
		})
	case "stop":
		d.writeResponse(conn, DaemonResponse{OK: true})
		if d.cancel != nil {
			d.cancel()
		}
	default:
		d.writeResponse(conn, DaemonResponse{OK: false, Error: "unknown command"})
	}
}

func (d *Daemon) writeResponse(conn net.Conn, resp DaemonResponse) {
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	conn.Write(data)
}

// DaemonClient communicates with a running daemon via Unix socket.
type DaemonClient struct {
	sockPath string
}

// NewDaemonClient creates a client for the given socket path.
func NewDaemonClient(sockPath string) *DaemonClient {
	return &DaemonClient{sockPath: sockPath}
}

// IsRunning checks if a daemon is listening on the socket.
func (c *DaemonClient) IsRunning() bool {
	conn, err := net.Dial("unix", c.sockPath)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Status queries the daemon status.
func (c *DaemonClient) Status() (*DaemonResponse, error) {
	return c.send("status")
}

// Sync triggers a sync via the daemon.
func (c *DaemonClient) Sync() (*DaemonResponse, error) {
	return c.send("sync")
}

// Stop tells the daemon to shut down.
func (c *DaemonClient) Stop() error {
	_, err := c.send("stop")
	return err
}

func (c *DaemonClient) send(cmd string) (*DaemonResponse, error) {
	conn, err := net.Dial("unix", c.sockPath)
	if err != nil {
		return nil, fmt.Errorf("daemon not running: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	data, _ := json.Marshal(DaemonCommand{Cmd: cmd})
	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	var resp DaemonResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func cleanStaleSocket(path string) error {
	conn, err := net.Dial("unix", path)
	if err == nil {
		conn.Close()
		return fmt.Errorf("daemon already running at %s", path)
	}

	if _, statErr := os.Stat(path); statErr == nil {
		return os.Remove(path)
	}
	return nil
}
