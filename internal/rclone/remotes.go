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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Remote struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// ListRemotes returns all remotes in rclone.conf.
// Types are enriched from `rclone config dump` when available; dump failures
// leave Type empty so listremotes still succeeds.
func (c *Client) ListRemotes(ctx context.Context) ([]Remote, error) {
	// rclone listremotes (no --long flag; format: "remote:")
	// Exit 2 + Usage message when config is empty/missing — treat as zero remotes.
	out, err := c.run(ctx, nil, "listremotes", "--config", c.config)
	if err != nil {
		// Empty/missing config often exits non-zero with usage text on stderr
		// (now separated from stdout). Treat as zero remotes.
		msg := string(out) + err.Error()
		if strings.Contains(msg, "Usage:") || strings.Contains(msg, "Available commands:") {
			return nil, nil
		}
		return nil, err
	}
	typesByName := c.remoteTypesFromDump(ctx)
	var remotes []Remote
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		name := strings.TrimSuffix(strings.TrimSpace(line), ":")
		if name == "" {
			continue
		}
		remotes = append(remotes, Remote{Name: name, Type: typesByName[name]})
	}
	return remotes, nil
}

// remoteTypesFromDump maps remote name → type via `rclone config dump` JSON.
// Returns an empty map on any failure so callers can still list names.
func (c *Client) remoteTypesFromDump(ctx context.Context) map[string]string {
	out, err := c.run(ctx, nil, "config", "dump", "--config", c.config)
	if err != nil || len(out) == 0 {
		return map[string]string{}
	}
	// dump shape: { "name": { "type": "drive", ... }, ... }
	var dump map[string]map[string]any
	if err := json.Unmarshal(out, &dump); err != nil {
		return map[string]string{}
	}
	outMap := make(map[string]string, len(dump))
	for name, section := range dump {
		if section == nil {
			continue
		}
		if t, ok := section["type"].(string); ok {
			outMap[name] = t
		}
	}
	return outMap
}

// CreateRemote creates a new remote non-interactively.
// configKVs is a list of "key=value" pairs to pass to rclone config create.
func (c *Client) CreateRemote(ctx context.Context, name, remoteType string, configKVs []string) error {
	args := []string{"config", "create", name, remoteType}
	for _, kv := range configKVs {
		args = append(args, kv)
	}
	// Obscure password-like values (mega pass, sftp pass, …) the same way
	// rclone config does interactively.
	args = append(args, "--obscure", "--config", c.config)
	_, err := c.run(ctx, nil, args...)
	return err
}

// CreateRemoteVerified creates a remote then probes it (lsd). On probe
// failure the remote is deleted so an unauthenticated entry is never kept.
func (c *Client) CreateRemoteVerified(ctx context.Context, name, remoteType string, configKVs []string) error {
	if err := c.CreateRemote(ctx, name, remoteType, configKVs); err != nil {
		return err
	}
	if err := c.TestRemote(ctx, name); err != nil {
		_ = c.DeleteRemote(ctx, name)
		return fmt.Errorf("auth failed: %w", err)
	}
	return nil
}

// DeleteRemote removes a remote.
func (c *Client) DeleteRemote(ctx context.Context, name string) error {
	_, err := c.run(ctx, nil, "config", "delete", name, "--config", c.config)
	return err
}

// TestRemote verifies that the remote is reachable by listing its root.
func (c *Client) TestRemote(ctx context.Context, name string) error {
	_, err := c.run(ctx, nil, "lsd", name+":", "--config", c.config, "--max-depth", "1")
	return err
}

// --- internal -------------------------------------------------------------

func (c *Client) run(ctx context.Context, env []string, args ...string) ([]byte, error) {
	c.mu.Lock()
	cmd := exec.CommandContext(ctx, c.rcloneBin, args...)
	c.mu.Unlock()
	cmd.Env = append(os.Environ(), env...)
	// Keep stdout and stderr separate. CombinedOutput interleaves rclone NOTICE
	// lines into JSON (lsjson/about), which breaks parsing on local paths with
	// symlinks/sockets (e.g. "invalid character '/' after array element").
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.Bytes()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return out, fmt.Errorf("rclone %s: %w (%s)", strings.Join(args, " "), err, truncate(msg, 500))
	}
	return out, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
