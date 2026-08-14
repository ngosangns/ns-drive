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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type jsonLogStats struct {
	Bytes          int64    `json:"bytes"`
	TotalBytes     int64    `json:"totalBytes"`
	Transfers      int64    `json:"transfers"`
	TotalTransfers int64    `json:"totalTransfers"`
	Checks         int64    `json:"checks"`
	TotalChecks    int64    `json:"totalChecks"`
	Deletes        int64    `json:"deletes"`
	Renames        int64    `json:"renames"`
	Errors         int64    `json:"errors"`
	Speed          float64  `json:"speed"`
	Eta            *float64 `json:"eta"`
	// Transferring is present on some rclone versions during active transfers.
	Transferring []jsonTransferring `json:"transferring"`
}

type jsonTransferring struct {
	Name       string  `json:"name"`
	Size       int64   `json:"size"`
	Bytes      int64   `json:"bytes"`
	Percentage int     `json:"percentage"`
	Speed      float64 `json:"speed"`
}

type jsonLogLine struct {
	Level  string        `json:"level"`
	Stats  *jsonLogStats `json:"stats"`
	Object string        `json:"object"`
	Msg    string        `json:"msg"`
	Size   int64         `json:"size"`
}

// maxTrackedFiles caps the per-file list shipped to the UI (Wails uses 100).
const maxTrackedFiles = 150

// fileTransferTracker accumulates per-file status from CLI JSON logs.
type fileTransferTracker struct {
	byName map[string]*FileTransfer
	order  []string // insertion order for stable UI
}

func newFileTransferTracker() *fileTransferTracker {
	return &fileTransferTracker{byName: make(map[string]*FileTransfer)}
}

func (t *fileTransferTracker) upsert(ft FileTransfer) {
	if ft.Name == "" {
		return
	}
	if prev, ok := t.byName[ft.Name]; ok {
		// Don't demote completed/failed back to transferring unless still active.
		if (prev.Status == "completed" || prev.Status == "failed" || prev.Status == "checked") &&
			ft.Status == "transferring" {
			return
		}
		*prev = ft
		return
	}
	if len(t.order) >= maxTrackedFiles && ft.Status != "failed" {
		// Prefer keeping failures; drop oldest completed if full.
		t.evictOldestCompleted()
		if len(t.order) >= maxTrackedFiles {
			return
		}
	}
	cp := ft
	t.byName[ft.Name] = &cp
	t.order = append(t.order, ft.Name)
}

func (t *fileTransferTracker) evictOldestCompleted() {
	for i, name := range t.order {
		if ft := t.byName[name]; ft != nil && (ft.Status == "completed" || ft.Status == "checked") {
			delete(t.byName, name)
			t.order = append(t.order[:i], t.order[i+1:]...)
			return
		}
	}
}

// seedPending inserts listed files as status=pending without demoting names
// already known as transferring/completed/failed/checking.
func (t *fileTransferTracker) seedPending(entries []FileEntry) {
	for _, e := range entries {
		if e.IsDir {
			continue
		}
		name := strings.TrimSpace(e.Path)
		if name == "" {
			name = strings.TrimSpace(e.Name)
		}
		if name == "" {
			continue
		}
		if prev, ok := t.byName[name]; ok && prev != nil {
			// Keep live status; only fill size if still pending/unknown.
			if prev.Size == 0 && e.Size > 0 {
				prev.Size = e.Size
			}
			continue
		}
		t.upsert(FileTransfer{
			Name:     name,
			Size:     e.Size,
			Bytes:    0,
			Progress: 0,
			Status:   "pending",
		})
	}
}

func (t *fileTransferTracker) snapshot(totalFiles int64) []FileTransfer {
	out := make([]FileTransfer, 0, len(t.order)+1)
	active := 0
	completed := 0
	failed := 0
	pendingNamed := 0
	for _, name := range t.order {
		ft := t.byName[name]
		if ft == nil {
			continue
		}
		out = append(out, *ft)
		switch ft.Status {
		case "transferring", "checking":
			active++
		case "completed", "checked":
			completed++
		case "failed":
			failed++
		case "pending":
			pendingNamed++
		}
	}
	// Synthetic count only for files beyond named rows (cap/list miss).
	// When seedPending already listed names, pendingNamed covers them.
	if totalFiles > 0 {
		known := int64(completed + failed + active + pendingNamed)
		if pend := totalFiles - known; pend > 0 {
			out = append(out, FileTransfer{
				Name:     fmt.Sprintf("(%d pending)", pend),
				Status:   "pending",
				Progress: 0,
			})
		}
	}
	return out
}

// parseJSONStatsLine parses a single rclone --use-json-log line. If the line is
// a JSON object carrying a "stats" object, it populates s and returns true.
// Non-JSON or non-stats lines return false so the caller can fall back to the
// legacy text parser.
func parseJSONStatsLine(line string, s *Stats) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return false
	}
	var entry jsonLogLine
	if err := json.Unmarshal([]byte(line), &entry); err != nil || entry.Stats == nil {
		return false
	}
	st := entry.Stats
	s.Bytes = st.Bytes
	s.BytesTotal = st.TotalBytes
	s.Files = st.Transfers
	s.FilesTotal = st.TotalTransfers
	s.Transfers = st.Transfers
	s.Checks = st.Checks
	s.ChecksTotal = st.TotalChecks
	s.Deletes = st.Deletes
	s.Renames = st.Renames
	s.Errors = st.Errors
	s.Speed = st.Speed
	if st.Eta != nil {
		s.ETA = int64(*st.Eta)
	}
	if entry.Object != "" {
		s.CurrentFile = entry.Object
	}
	return true
}

// ingestJSONLogLine updates aggregate stats + per-file tracker from one log line.
func ingestJSONLogLine(line string, s *Stats, track *fileTransferTracker) {
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' || track == nil {
		return
	}
	var entry jsonLogLine
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return
	}

	// Active multi-file transfers from stats.transferring (when rclone provides it).
	if entry.Stats != nil && len(entry.Stats.Transferring) > 0 {
		for _, tr := range entry.Stats.Transferring {
			if tr.Name == "" {
				continue
			}
			s.CurrentFile = tr.Name
			track.upsert(FileTransfer{
				Name:     tr.Name,
				Size:     tr.Size,
				Bytes:    tr.Bytes,
				Progress: float64(tr.Percentage),
				Status:   "transferring",
				Speed:    tr.Speed,
			})
		}
	}

	if entry.Object == "" {
		return
	}
	s.CurrentFile = entry.Object
	msg := strings.ToLower(entry.Msg)
	level := strings.ToLower(entry.Level)

	ft := FileTransfer{Name: entry.Object, Size: entry.Size, Bytes: entry.Size}

	switch {
	case level == "error" || strings.Contains(msg, "error") || strings.Contains(msg, "failed"):
		ft.Status = "failed"
		ft.Error = entry.Msg
		ft.Progress = 0
	case strings.Contains(msg, "check"):
		if strings.Contains(msg, "ok") || strings.Contains(msg, "identical") {
			ft.Status = "checked"
			ft.Progress = 100
		} else {
			ft.Status = "checking"
		}
	case strings.Contains(msg, "copied") ||
		strings.Contains(msg, "moved") ||
		strings.Contains(msg, "updated") ||
		strings.Contains(msg, "multi-thread"):
		ft.Status = "completed"
		ft.Progress = 100
		if entry.Size > 0 {
			ft.Bytes = entry.Size
		}
	default:
		// Unknown object notice — treat as completed success path if info-level.
		if level == "info" || level == "notice" {
			ft.Status = "completed"
			ft.Progress = 100
		} else {
			return
		}
	}
	track.upsert(ft)
}

// updateStageFromLog reduces rclone's implementation-specific messages into
// safe, stable UI lifecycle markers. Raw messages can contain local paths and
// credentials, so only object names are exposed as optional detail.
func updateStageFromLog(line string, s *Stats) {
	line = strings.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' || s == nil {
		return
	}
	var entry jsonLogLine
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return
	}
	msg := strings.ToLower(entry.Msg)
	stage := ""
	switch {
	case (entry.Stats != nil && len(entry.Stats.Transferring) > 0) || strings.Contains(msg, "copied") || strings.Contains(msg, "transferred") || strings.Contains(msg, "moved"):
		stage = "transferring"
	case strings.Contains(msg, "checking") || strings.Contains(msg, "check"):
		stage = "checking"
	case strings.Contains(msg, "listing") || strings.Contains(msg, "list "):
		stage = "listing"
	case strings.Contains(msg, "retry") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "throttle"):
		stage = "retrying"
	case strings.Contains(msg, "starting") || strings.Contains(msg, "config file") || strings.Contains(msg, "using "):
		stage = "connecting"
	}
	if stage == "" {
		return
	}
	s.Stage = stage
	if entry.Object != "" {
		s.StageDetail = entry.Object
	} else {
		s.StageDetail = ""
	}
}

// parseStatsLine extracts progress numbers from an rclone --stats-one-line line.
// Format (approximate): "2025/01/15 10:00:00 INFO  : ... TRANSFER: 1.024k/2.048k ..."
func parseStatsLine(line string, s *Stats) {
	if !strings.Contains(line, "INFO") {
		return
	}
	// Look for "X/Y" patterns after TRANSFER / CHECK / etc.
	if i := strings.Index(line, "TRANSFER: "); i >= 0 {
		if a, b, ok := parseFraction(line[i:]); ok {
			s.Bytes = a
			s.BytesTotal = b
		}
	}
	if i := strings.Index(line, "CHECK: "); i >= 0 {
		if a, b, ok := parseFraction(line[i:]); ok {
			s.Checks = a
			s.ChecksTotal = b
		}
	}
	if i := strings.Index(line, "ERRORS: "); i >= 0 {
		if n, ok := parseInt(line[i:]); ok {
			s.Errors = n
		}
	}
	if i := strings.Index(line, "DELETED: "); i >= 0 {
		if n, ok := parseInt(line[i:]); ok {
			s.Deletes = n
		}
	}
}

func parseFraction(s string) (int64, int64, bool) {
	// Find "X/Y" where X and Y are size-suffixed numbers (e.g. "1.024k/2.048k").
	idx := strings.Index(s, " ")
	if idx < 0 {
		return 0, 0, false
	}
	rest := s[idx+1:]
	slash := strings.Index(rest, "/")
	if slash < 0 {
		return 0, 0, false
	}
	left := rest[:slash]
	// Take the next token after "/"
	rightAndMore := rest[slash+1:]
	space := strings.Index(rightAndMore, " ")
	var right string
	if space < 0 {
		right = rightAndMore
	} else {
		right = rightAndMore[:space]
	}
	return parseSize(left), parseSize(right), true
}

func parseInt(s string) (int64, bool) {
	idx := strings.Index(s, " ")
	if idx < 0 {
		return 0, false
	}
	rest := s[idx+1:]
	end := 0
	for end < len(rest) && (rest[end] >= '0' && rest[end] <= '9') {
		end++
	}
	if end == 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(rest[:end], 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseSize parses rclone size suffixes: "1.024k", "2M", "1G", "1024".
// Returns bytes.
func parseSize(s string) int64 {
	if s == "" {
		return 0
	}
	// Find first non-digit/dot character.
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
		i++
	}
	numStr := s[:i]
	suffix := strings.ToLower(s[i:])
	if numStr == "" {
		return 0
	}
	n, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	mult := float64(1)
	switch suffix {
	case "k", "kb":
		mult = 1024
	case "m", "mb":
		mult = 1024 * 1024
	case "g", "gb":
		mult = 1024 * 1024 * 1024
	case "t", "tb":
		mult = 1024 * 1024 * 1024 * 1024
	}
	return int64(n * mult)
}

func nowUnix() int64 {
	return timeNowFunc()
}

// --- File operations -------------------------------------------------------

// ListFiles returns files and directories at a path.
// remotePath may be "remote:path" or an absolute local filesystem path.
