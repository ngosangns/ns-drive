// Package rclone provides a shell-out wrapper around the rclone binary.
//
// Phase 2: implements sync, bisync, copy, move, check, list, mkdir, purge,
// delete, about, and remote CRUD via exec.Command("rclone", ...). Progress is
// reported via a simple stats channel populated from rclone's --stats output.
//
// The wrapper does not depend on the rclone Go library — keeping go.mod minimal
// and ensuring any rclone version installed on the host works.
package rclone

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Action represents a sync direction.
type Action string

const (
	ActionPull     Action = "pull"
	ActionPush     Action = "push"
	ActionBi       Action = "bi"
	ActionBiResync Action = "bi-resync"
	ActionCopy     Action = "copy"
	ActionMove     Action = "move"
	ActionCheck    Action = "check"
	ActionDryRun   Action = "dry-run"
)

// Client wraps the rclone binary and config path.
type Client struct {
	mu        sync.Mutex
	rcloneBin string
	config    string
	logger    *slog.Logger
}

// Options configures the Client.
type Options struct {
	// BinaryPath is the absolute path to rclone. Defaults to "rclone" in PATH.
	BinaryPath string
	// ConfigPath is the rclone.conf path. Defaults to ~/.config/gn-drive/rclone.conf.
	ConfigPath string
	// Logger is the structured logger to use. Defaults to slog.Default().
	Logger *slog.Logger
}

// New creates a new rclone Client.
func New(opts Options) (*Client, error) {
	bin := opts.BinaryPath
	if bin == "" {
		p, err := exec.LookPath("rclone")
		if err != nil {
			return nil, fmt.Errorf("rclone: binary not found in PATH: %w", err)
		}
		bin = p
	} else {
		// Try LookPath first (handles "rclone" on PATH). Fall back to Stat
		// for absolute paths that LookPath can't resolve.
		if p, err := exec.LookPath(bin); err == nil {
			bin = p
		} else if _, err := os.Stat(bin); err != nil {
			return nil, fmt.Errorf("rclone: binary not found at %s: %w", bin, err)
		}
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		rcloneBin: bin,
		config:    opts.ConfigPath,
		logger:    logger,
	}, nil
}

// Binary returns the resolved rclone binary path.
func (c *Client) Binary() string { return c.rcloneBin }

// ConfigPath returns the rclone.conf path used by this client.
func (c *Client) ConfigPath() string { return c.config }

// Version returns the rclone version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	out, err := c.run(ctx, nil, "version")
	if err != nil {
		return "", err
	}
	// First line: "rclone v1.74.2"
	first := strings.SplitN(string(out), "\n", 2)[0]
	return strings.TrimSpace(first), nil
}

// FileTransfer is one file's live status (Wails FileTransferInfo).
