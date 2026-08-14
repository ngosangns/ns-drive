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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileTransfer struct {
	Name     string  `json:"name"`
	Size     int64   `json:"size"`
	Bytes    int64   `json:"bytes"`
	Progress float64 `json:"progress"` // 0-100
	Status   string  `json:"status"`   // transferring | completed | failed | checking | checked | pending
	Speed    float64 `json:"speed,omitempty"`
	Error    string  `json:"error,omitempty"`
}

// Stats describes progress during a sync operation.
type Stats struct {
	Bytes       int64   `json:"bytes"`
	BytesTotal  int64   `json:"bytes_total"`
	Files       int64   `json:"files"`
	FilesTotal  int64   `json:"files_total"`
	Transfers   int64   `json:"transfers"`
	Errors      int64   `json:"errors"`
	Checks      int64   `json:"checks"`
	ChecksTotal int64   `json:"checks_total"`
	Deletes     int64   `json:"deletes"`
	Renames     int64   `json:"renames"`
	Speed       float64 `json:"speed_bps"`
	ETA         int64   `json:"eta_secs"`
	CurrentFile string  `json:"current_file,omitempty"`
	// Stage is a coarse rclone lifecycle marker derived from its structured log.
	// It lets the UI explain work before byte-transfer stats exist.
	Stage       string `json:"stage,omitempty"`
	StageDetail string `json:"stage_detail,omitempty"`
	LastUpdate  int64  `json:"last_update_unix"`
	// FileTransfers is the per-file list for the status panel (capped).
	FileTransfers []FileTransfer `json:"file_transfers,omitempty"`
}

// SyncResult is the outcome of a sync operation.
type SyncResult struct {
	Stats     Stats
	StartedAt int64
	EndedAt   int64
	ExitCode  int
	Stderr    string
}

// SyncConfig is the per-operation configuration.
type SyncConfig struct {
	Action       Action
	Source       string // remote:path or local path
	SourceRemote string
	SourcePath   string
	Dest         string
	DestRemote   string
	DestPath     string
	// Resync forces a bisync resync.
	Resync bool
	// Profile is the optional profile to apply flags from.
	Profile *ProfileFlags
	// StatsInterval is how often to emit stats. Default: 1s.
	StatsInterval string
}

// ProfileFlags are the rclone flags a profile / flow SyncConfig can set.
// Mirrors Wails SyncConfig fields relevant to the CLI shell-out.
type ProfileFlags struct {
	Bandwidth          string
	Transfers          int
	Checkers           int
	TpsLimit           float64
	MinAge             string
	MaxAge             string
	MinSize            string
	MaxSize            string
	ExcludeIfPresent   string
	MaxDelete          int
	DryRun             bool
	UseListR           bool
	NoUnicodeNormalize bool

	// Filters (Wails includedPaths / excludedPaths)
	Includes []string
	Excludes []string

	// Performance
	MultiThreadStreams int
	BufferSize         string
	Retries            int
	LowLevelRetries    int
	MaxDuration        string
	RetriesSleep       string
	ConnTimeout        string
	IoTimeout          string
	OrderBy            string
	CheckFirst         bool

	// Safety / comparison
	Immutable           bool
	MaxTransfer         string
	MaxDeleteSize       string
	Suffix              string
	SuffixKeepExtension bool
	BackupDir           string
	SizeOnly            bool
	UpdateMode          bool
	IgnoreExisting      bool
	DeleteExcluded      bool
	MaxDepth            int

	// Sync (push)
	DeleteTiming string // before|during|after

	// Bisync
	ConflictResolve string
	ConflictLoser   string
	ConflictSuffix  string
	Resilient       bool
	MaxLock         string
	CheckAccess     bool
}

// --- Sync / BiSync / Copy / Move / Check ----------------------------------

// Sync runs the configured action. It streams progress via onProgress (may be nil).
func (c *Client) Sync(ctx context.Context, cfg SyncConfig, onProgress func(Stats)) (*SyncResult, error) {
	args, cleanup, err := c.buildArgs(cfg)
	if err != nil {
		return nil, err
	}
	if cleanup != "" {
		defer os.Remove(cleanup)
	}
	// Seed the Pending tab with real file names from the data source while
	// rclone runs. Without this the UI only sees transferring/completed names
	// (or a synthetic "(N pending)" count) and the Pending tab looks empty.
	seedPath := pendingSeedPath(cfg)
	return c.execute(ctx, args, onProgress, seedPath)
}

// pendingSeedPath picks which endpoint to list for the Pending file tab.
// Push/copy/move list Source; pull lists Dest (truth side that feeds Source).
func pendingSeedPath(cfg SyncConfig) string {
	src, dst := cfg.Source, cfg.Dest
	if src == "" || dst == "" {
		if cfg.SourceRemote != "" && cfg.SourcePath != "" {
			src = cfg.SourceRemote + ":" + cfg.SourcePath
		}
		if cfg.DestRemote != "" && cfg.DestPath != "" {
			dst = cfg.DestRemote + ":" + cfg.DestPath
		}
	}
	switch cfg.Action {
	case ActionPull:
		return dst
	default:
		return src
	}
}

func (c *Client) buildArgs(cfg SyncConfig) (args []string, cleanup string, err error) {
	src, dst, err := c.resolveEndpoints(cfg)
	if err != nil {
		return nil, "", err
	}

	interval := cfg.StatsInterval
	if interval == "" {
		interval = "1s"
	}

	// --use-json-log makes rclone emit periodic stats as a structured JSON
	// object (parsed by parseJSONStatsLine), which is far more robust than
	// scraping the human-readable one-line text. The text parser remains as a
	// fallback for older rclone builds / non-JSON lines.
	base := []string{"--config", c.config, "--stats", interval, "--use-json-log", "-v"}

	switch cfg.Action {
	case ActionPull:
		// Pull: one-way Dest → Source. Callers keep From/Source and To/Dest as
		// fixed path slots; pull reverses data flow so Dest is the truth and
		// Source is updated (e.g. From=local, To=remote → download remote→local).
		args = append([]string{"sync", dst, src, "--update"}, base...)
	case ActionPush:
		// Push: one-way Source → Dest (e.g. From=local, To=remote → upload).
		args = append([]string{"sync", src, dst, "--update"}, base...)
	case ActionBi:
		// Incremental bidirectional sync. bisync relies on the listings stored
		// in its workdir by a previous run; it must NOT pass --resync on every
		// run (that re-establishes the baseline and can clobber concurrent
		// changes / delete data). A brand-new pair must be primed once with
		// ActionBiResync; until then rclone bisync exits with a clear
		// "cannot find prior listing — run with --resync" error, which is the
		// safe behaviour.
		args = append([]string{"bisync", src, dst}, base...)
	case ActionBiResync:
		// Establish (or rebuild) the bisync baseline. --force permits large
		// deltas that bisync would otherwise refuse.
		args = append([]string{"bisync", src, dst, "--resync", "--force"}, base...)
	case ActionCopy:
		args = append([]string{"copy", src, dst}, base...)
	case ActionMove:
		args = append([]string{"move", src, dst}, base...)
	case ActionCheck:
		args = append([]string{"check", src, dst}, base...)
	case ActionDryRun:
		args = append([]string{"sync", src, dst, "--dry-run", "--update"}, base...)
	default:
		return nil, "", fmt.Errorf("rclone: unknown action %q", cfg.Action)
	}

	if cfg.Profile != nil {
		args = append(args, profileToFlags(cfg.Profile)...)
	}
	return args, cleanup, nil
}

func (c *Client) resolveEndpoints(cfg SyncConfig) (src, dst string, err error) {
	if cfg.Source != "" && cfg.Dest != "" {
		return cfg.Source, cfg.Dest, nil
	}
	if cfg.SourceRemote == "" || cfg.SourcePath == "" || cfg.DestRemote == "" || cfg.DestPath == "" {
		return "", "", errors.New("rclone: SyncConfig requires Source+Dest or SourceRemote+SourcePath+DestRemote+DestPath")
	}
	return cfg.SourceRemote + ":" + cfg.SourcePath, cfg.DestRemote + ":" + cfg.DestPath, nil
}

func profileToFlags(p *ProfileFlags) []string {
	if p == nil {
		return nil
	}
	var f []string
	if p.Bandwidth != "" {
		f = append(f, "--bwlimit", p.Bandwidth)
	}
	if p.Transfers > 0 {
		f = append(f, "--transfers", strconv.Itoa(p.Transfers))
	}
	if p.Checkers > 0 {
		f = append(f, "--checkers", strconv.Itoa(p.Checkers))
	}
	if p.TpsLimit > 0 {
		f = append(f, "--tpslimit", strconv.FormatFloat(p.TpsLimit, 'f', -1, 64))
	}
	if p.MinAge != "" {
		f = append(f, "--min-age", p.MinAge)
	}
	if p.MaxAge != "" {
		f = append(f, "--max-age", p.MaxAge)
	}
	if p.MinSize != "" {
		f = append(f, "--min-size", p.MinSize)
	}
	if p.MaxSize != "" {
		f = append(f, "--max-size", p.MaxSize)
	}
	if p.ExcludeIfPresent != "" {
		f = append(f, "--exclude-if-present", p.ExcludeIfPresent)
	}
	if p.MaxDelete > 0 {
		f = append(f, "--max-delete", strconv.Itoa(p.MaxDelete))
	}
	if p.DryRun {
		f = append(f, "--dry-run")
	}
	if p.NoUnicodeNormalize {
		f = append(f, "--no-unicode-normalization")
	}
	for _, inc := range p.Includes {
		if s := strings.TrimSpace(inc); s != "" {
			f = append(f, "--include", s)
		}
	}
	for _, exc := range p.Excludes {
		if s := strings.TrimSpace(exc); s != "" {
			f = append(f, "--exclude", s)
		}
	}
	if p.MultiThreadStreams > 0 {
		f = append(f, "--multi-thread-streams", strconv.Itoa(p.MultiThreadStreams))
	}
	if p.BufferSize != "" {
		f = append(f, "--buffer-size", p.BufferSize)
	}
	if p.Retries > 0 {
		f = append(f, "--retries", strconv.Itoa(p.Retries))
	}
	if p.LowLevelRetries > 0 {
		f = append(f, "--low-level-retries", strconv.Itoa(p.LowLevelRetries))
	}
	if p.MaxDuration != "" {
		f = append(f, "--max-duration", p.MaxDuration)
	}
	if p.RetriesSleep != "" {
		f = append(f, "--retries-sleep", p.RetriesSleep)
	}
	if p.ConnTimeout != "" {
		f = append(f, "--contimeout", p.ConnTimeout)
	}
	if p.IoTimeout != "" {
		f = append(f, "--timeout", p.IoTimeout)
	}
	if p.OrderBy != "" {
		f = append(f, "--order-by", p.OrderBy)
	}
	if p.CheckFirst {
		f = append(f, "--check-first")
	}
	if p.Immutable {
		f = append(f, "--immutable")
	}
	if p.MaxTransfer != "" {
		f = append(f, "--max-transfer", p.MaxTransfer)
	}
	if p.MaxDeleteSize != "" {
		f = append(f, "--max-delete-size", p.MaxDeleteSize)
	}
	if p.Suffix != "" {
		f = append(f, "--suffix", p.Suffix)
	}
	if p.SuffixKeepExtension {
		f = append(f, "--suffix-keep-extension")
	}
	if p.BackupDir != "" {
		f = append(f, "--backup-dir", p.BackupDir)
	}
	if p.SizeOnly {
		f = append(f, "--size-only")
	}
	if p.UpdateMode {
		f = append(f, "--update")
	}
	if p.IgnoreExisting {
		f = append(f, "--ignore-existing")
	}
	if p.DeleteExcluded {
		f = append(f, "--delete-excluded")
	}
	if p.MaxDepth > 0 {
		f = append(f, "--max-depth", strconv.Itoa(p.MaxDepth))
	}
	switch strings.ToLower(strings.TrimSpace(p.DeleteTiming)) {
	case "before":
		f = append(f, "--delete-before")
	case "after":
		f = append(f, "--delete-after")
	case "during":
		f = append(f, "--delete-during")
	}
	if p.ConflictResolve != "" {
		f = append(f, "--conflict-resolve", p.ConflictResolve)
	}
	if p.ConflictLoser != "" {
		f = append(f, "--conflict-loser", p.ConflictLoser)
	}
	if p.ConflictSuffix != "" {
		f = append(f, "--conflict-suffix", p.ConflictSuffix)
	}
	if p.Resilient {
		f = append(f, "--resilient")
	}
	if p.MaxLock != "" {
		f = append(f, "--max-lock", p.MaxLock)
	}
	if p.CheckAccess {
		f = append(f, "--check-access")
	}
	return f
}

// execute runs rclone with the given args and parses --stats-one-line output.
// execCmd is the subset of *exec.Cmd used by execute. It exists so tests
// can inject a stub to exercise the StdoutPipe/StderrPipe/Start error paths.
type execCmd interface {
	StdoutPipe() (io.ReadCloser, error)
	StderrPipe() (io.ReadCloser, error)
	Start() error
	Wait() error
}

// newExecCommand is overridable for tests; defaults to exec.CommandContext.
var newExecCommand = func(ctx context.Context, name string, args ...string) execCmd {
	return exec.CommandContext(ctx, name, args...)
}

func (c *Client) execute(ctx context.Context, args []string, onProgress func(Stats), seedPath string) (*SyncResult, error) {
	c.mu.Lock()
	cmd := newExecCommand(ctx, c.rcloneBin, args...)
	c.mu.Unlock()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("rclone: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("rclone: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("rclone: start: %w", err)
	}

	result := &SyncResult{StartedAt: nowUnix(), ExitCode: -1}

	// rclone --use-json-log writes progress/stats to STDERR (not stdout).
	// Older text --stats lines may appear on either stream. Parse both.
	// fileTrack accumulates per-file status from object lines + stats.transferring
	// so the UI can show processing / completed / failed / pending (Wails tabs).
	var (
		statsMu   sync.Mutex
		stats     Stats
		fileTrack = newFileTransferTracker()
		stderrBuf strings.Builder
		wg        sync.WaitGroup
	)

	emitProgress := func() {
		// Caller must hold statsMu. snap is a value copy; FileTransfers is
		// deep-copied so onProgress can retain the slice after unlock.
		stats.FileTransfers = fileTrack.snapshot(stats.FilesTotal)
		snap := stats
		snap.LastUpdate = nowUnix()
		if n := len(stats.FileTransfers); n > 0 {
			snap.FileTransfers = make([]FileTransfer, n)
			copy(snap.FileTransfers, stats.FileTransfers)
		} else {
			snap.FileTransfers = nil
		}
		if onProgress != nil {
			onProgress(snap)
		}
	}

	// rclone may spend a long time authenticating and listing before its first
	// periodic stats line. Emit an immediate lifecycle marker for that gap.
	stats.Stage = "starting"
	stats.StageDetail = ""
	emitProgress()

	// Concurrent seed: list source files as pending so the Pending tab has
	// real names before transfers start. Cap + timeout keep large trees from
	// stalling the run. Failures are non-fatal (log-only). Own WaitGroup so a
	// slow list cannot delay stream drain beyond cancel-after-exit.
	var seedWG sync.WaitGroup
	seedCtx, seedCancel := context.WithCancel(ctx)
	defer seedCancel()
	if seedPath != "" {
		seedWG.Add(1)
		go func() {
			defer seedWG.Done()
			listCtx, cancel := context.WithTimeout(seedCtx, 20*time.Second)
			defer cancel()
			entries, err := c.listFilesForPending(listCtx, seedPath, maxTrackedFiles)
			if err != nil || len(entries) == 0 {
				return
			}
			statsMu.Lock()
			fileTrack.seedPending(entries)
			emitProgress()
			statsMu.Unlock()
		}()
	}

	consume := func(r io.Reader, capture *strings.Builder) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			if capture != nil {
				capture.WriteString(line)
				capture.WriteByte('\n')
			}
			statsMu.Lock()
			// Prefer structured JSON stats; fall back to text TRANSFER lines.
			if !parseJSONStatsLine(line, &stats) {
				parseStatsLine(line, &stats)
			}
			updateStageFromLog(line, &stats)
			// Always try per-file event extraction (object lines + transferring[]).
			ingestJSONLogLine(line, &stats, fileTrack)
			emitProgress()
			statsMu.Unlock()
		}
	}

	wg.Add(2)
	go consume(stdout, nil)
	go consume(stderr, &stderrBuf)
	wg.Wait()
	// Stop pending seed once rclone streams end so Wait is not blocked by
	// a long remote listing after the transfer already finished.
	seedCancel()
	seedWG.Wait()

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Stderr = stderrBuf.String()
		result.EndedAt = nowUnix()
		statsMu.Lock()
		stats.FileTransfers = fileTrack.snapshot(stats.FilesTotal)
		result.Stats = stats
		statsMu.Unlock()
		return result, fmt.Errorf("rclone: %w (stderr: %s)", err, truncate(stderrBuf.String(), 500))
	}

	result.ExitCode = 0
	result.EndedAt = nowUnix()
	statsMu.Lock()
	stats.FileTransfers = fileTrack.snapshot(stats.FilesTotal)
	result.Stats = stats
	statsMu.Unlock()
	return result, nil
}

// jsonLogStats mirrors the "stats" object rclone emits under --use-json-log.
// Field names match rclone's JSON keys.
