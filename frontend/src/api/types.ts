// TypeScript types mirroring the Go API JSON contracts.
// These match the structs in internal/store/models.go and the DTOs returned
// by internal/api/handlers.

export interface Profile {
  name: string
  from: string
  to: string
  /** Default sync action: push | bi | bi-resync */
  direction?: 'push' | 'bi' | 'bi-resync' | string
  included_paths?: string[] | null
  excluded_paths?: string[] | null
  bandwidth: number
  parallel: number
  backup_path?: string
  cache_path?: string
  min_size?: string
  max_size?: string
  filter_from_file?: string
  exclude_if_present?: string
  use_regex?: boolean
  max_age?: string
  min_age?: string
  max_depth?: number | null
  delete_excluded?: boolean
  max_delete?: number | null
  immutable?: boolean
  conflict_resolution?: string
  dry_run?: boolean
  max_transfer?: string
  max_delete_size?: string
  suffix?: string
  suffix_keep_extension?: boolean
  multi_thread_streams?: number | null
  buffer_size?: string
  retries?: number | null
  low_level_retries?: number | null
  max_duration?: string
  check_first?: boolean
  order_by?: string
  retries_sleep?: string
  tps_limit?: number | null
  conn_timeout?: string
  io_timeout?: string
  size_only?: boolean
  update_mode?: boolean
  ignore_existing?: boolean
  delete_timing?: string
  resilient?: boolean
  max_lock?: string
  check_access?: boolean
  conflict_loser?: string
  conflict_suffix?: string
  fast_list?: boolean
}

export interface Remote {
  name: string
  type: string
  description?: string
}

/**
 * Wails-aligned units:
 * - Flow: container, ops run sequentially
 * - Operation: one source→target sync step inside a flow
 * Profile is only rclone option bag / legacy CRUD, not a workspace unit.
 */
export interface Operation {
  id: string
  flow_id?: string
  source_remote: string
  source_path: string
  target_remote: string
  target_path: string
  action: string
  sync_config?: Record<string, unknown> | null
  is_expanded?: boolean
  sort_order?: number
  /** Runtime only */
  status?: string
  last_error?: string
}

export interface Flow {
  id: string
  name: string
  is_collapsed?: boolean
  schedule_enabled?: boolean
  enabled?: boolean
  schedule_cron?: string
  cron_expr?: string
  sort_order?: number
  operations?: Operation[]
  /** Location nodes + viewport for the workspace canvas. Ignored by execute. */
  canvas_json?: {
    viewport?: { x: number; y: number; zoom: number }
    nodes?: Array<{
      id: string
      remote: string
      path: string
      label?: string
      x: number
      y: number
    }>
  } | Record<string, unknown> | null
  created_at?: string
  updated_at?: string
  /** Runtime only */
  status?: string
  last_error?: string
}

/** Matches syncengine.TaskSnapshot (stats nested). */
export interface SyncTaskStats {
  bytes?: number
  bytes_total?: number
  files?: number
  files_total?: number
  transfers?: number
  errors?: number
  checks?: number
  checks_total?: number
  deletes?: number
  renames?: number
  speed_bps?: number
  eta_secs?: number
  current_file?: string
  last_update_unix?: number
}

/** Wails FileTransferInfo — one row in the file tabs. */
export interface FileTransferInfo {
  name: string
  size: number
  bytes: number
  progress: number
  /** transferring | completed | failed | checking | checked | pending */
  status: string
  speed?: number
  error?: string
}

/**
 * Wails SyncStatus-shaped snapshot for the flow run status panel
 * (desktop operation-logs-panel / sync-status).
 */
export interface FlowOpSyncStatus {
  flow_id: string
  op_id: string
  task_id?: string
  action: string
  status: 'running' | 'completed' | 'failed' | 'cancelled' | 'stopped'
  progress: number // 0-100
  speed_bps: number
  eta_secs: number
  files_transferred: number
  total_files: number
  bytes_transferred: number
  total_bytes: number
  current_file: string
	stage?: 'starting' | 'connecting' | 'listing' | 'checking' | 'transferring' | 'retrying' | string
	stage_detail?: string
  errors: number
  checks: number
  total_checks: number
  deletes: number
  renames: number
  /** Per-file list for Syncing / Complete / Failed / Pending tabs. */
  transfers?: FileTransferInfo[]
  error_message?: string
  updated_at: number
}

/** Backend RuntimeSnapshotEvent — hydrate after reload / SSE connect. */
export interface RuntimeOpState {
  id: string
  status: string
  last_error?: string
}

export interface RuntimeLogLine {
  at: number
  status: string
  op_id?: string
  error?: string
  label?: string
}

export interface RuntimeFlowState {
  id: string
  status: string
  last_error?: string
  ops?: RuntimeOpState[]
  sync?: Record<string, unknown> | null
  log?: RuntimeLogLine[]
}

export interface RuntimeSnapshot {
  revision?: number
  type?: string
  flows?: RuntimeFlowState[]
}

/** Canonical projection sent by the backend state WebSocket on every connect. */
export interface StateSnapshot {
  flows: Flow[]
  remotes: Remote[]
  settings: Record<string, string>
  runtime: RuntimeSnapshot
}

export interface SyncTask {
  id: string
  name: string
  action: string
  status: string
  stats?: SyncTaskStats
  started_at?: string
  ended_at?: string
  // convenience / legacy flat fields (optional)
  transferred?: number
  total?: number
  bytes_per_sec?: number
  errors?: number
  files_transferred?: number
  total_files?: number
  error_message?: string
}

export interface AppStatus {
  setup: boolean
  /** Process crypto unlocked AND (when setup) a valid web session cookie. */
  unlocked: boolean
  /** Valid gn-drive-session cookie present (minted on unlock or /status resume). */
  session?: boolean
  version: string
  lockout?: {
    failed_attempts: number
    locked_until: string
    is_locked: boolean
    retry_after_secs: number
  }
}

export interface FileEntry {
  name: string
  size: number
  mime_type: string
  is_dir: boolean
  mod_time: string
  path: string
  id: string
}

export interface AppSettings {
  theme?: string
  notifications_enabled?: string
  debug_mode?: string
  minimize_to_tray?: string
  start_at_login?: string
  [k: string]: string | undefined
}
