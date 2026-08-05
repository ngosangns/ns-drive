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
	"errors"
	"fmt"
	"strings"
)

func (c *Client) ListFiles(ctx context.Context, remotePath string) ([]FileEntry, error) {
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return nil, errors.New("rclone: path is required")
	}
	// Absolute local paths are valid for rclone without a remote section.
	// Named remotes must use "name:path" form.
	if !strings.Contains(remotePath, ":") && !strings.HasPrefix(remotePath, "/") {
		return nil, errors.New("rclone: path must be absolute (/path) or remote:path (e.g. \"gdrive:/folder\")")
	}
	// One level only for browser UX. Quiet log level keeps NOTICE (symlinks,
	// sockets) on stderr; run() only returns stdout so JSON stays clean.
	out, err := c.run(ctx, nil,
		"--log-level", "ERROR",
		"lsjson", remotePath,
		"--config", c.config,
		"--max-depth", "1",
	)
	if err != nil {
		return nil, err
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return []FileEntry{}, nil
	}
	var entries []FileEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("rclone: parse lsjson: %w", err)
	}
	return entries, nil
}

// listFilesForPending recursively lists files (not dirs) for the Pending tab seed.
// limit caps how many names we ship to the UI (same budget as maxTrackedFiles).
func (c *Client) listFilesForPending(ctx context.Context, remotePath string, limit int) ([]FileEntry, error) {
	remotePath = strings.TrimSpace(remotePath)
	if remotePath == "" {
		return nil, errors.New("rclone: path is required")
	}
	if !strings.Contains(remotePath, ":") && !strings.HasPrefix(remotePath, "/") {
		return nil, errors.New("rclone: path must be absolute (/path) or remote:path")
	}
	if limit <= 0 {
		limit = maxTrackedFiles
	}
	out, err := c.run(ctx, nil,
		"--log-level", "ERROR",
		"lsjson", remotePath,
		"--config", c.config,
		"--recursive",
		"--files-only",
	)
	if err != nil {
		return nil, err
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		return []FileEntry{}, nil
	}
	var entries []FileEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("rclone: parse lsjson: %w", err)
	}
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

// Mkdir creates a directory on a remote.
func (c *Client) Mkdir(ctx context.Context, remotePath string) error {
	_, err := c.run(ctx, nil, "mkdir", remotePath, "--config", c.config)
	return err
}

// Purge removes a directory and all its contents.
func (c *Client) Purge(ctx context.Context, remotePath string) error {
	_, err := c.run(ctx, nil, "purge", remotePath, "--config", c.config)
	return err
}

// DeleteFile deletes a single file.
func (c *Client) DeleteFile(ctx context.Context, remotePath string) error {
	_, err := c.run(ctx, nil, "deletefile", remotePath, "--config", c.config)
	return err
}

// About returns quota info for a remote (no path).
func (c *Client) About(ctx context.Context, remoteName string) (*QuotaInfo, error) {
	out, err := c.run(ctx, nil, "about", remoteName+":", "--config", c.config, "--json")
	if err != nil {
		return nil, err
	}
	var a struct {
		Used  int64 `json:"used"`
		Total int64 `json:"total"`
		Free  int64 `json:"free"`
	}
	if err := json.Unmarshal(out, &a); err != nil {
		return nil, fmt.Errorf("rclone: parse about: %w", err)
	}
	return &QuotaInfo{Used: a.Used, Total: a.Total, Free: a.Free}, nil
}

// --- Remotes CRUD ---------------------------------------------------------

// Remote describes an rclone remote.
