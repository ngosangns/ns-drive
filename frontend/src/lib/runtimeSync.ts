import type {
  FileTransferInfo,
  Flow,
  FlowOpSyncStatus,
  RuntimeSnapshot,
} from '@/api/types'

export type RuntimeLogEntry = {
  at: number
  status: string
  opId?: string
  error?: string
  label?: string
}

export function parseTransfers(
  raw: unknown,
  prev?: FileTransferInfo[],
): FileTransferInfo[] | undefined {
  if (!Array.isArray(raw)) return prev
  const out: FileTransferInfo[] = []
  for (const item of raw) {
    if (!item || typeof item !== 'object') continue
    const o = item as Record<string, unknown>
    const name = String(o.name ?? '')
    if (!name) continue
    out.push({
      name,
      size: Number(o.size ?? 0),
      bytes: Number(o.bytes ?? 0),
      progress: Number(o.progress ?? 0),
      status: String(o.status ?? 'completed'),
      speed: o.speed != null ? Number(o.speed) : undefined,
      error: o.error ? String(o.error) : undefined,
    })
  }
  return out
}

export function syncSnapEqual(a: FlowOpSyncStatus, b: FlowOpSyncStatus): boolean {
  if (a.status !== b.status || a.op_id !== b.op_id) return false
  if (Math.round(a.progress) !== Math.round(b.progress)) return false
  if (a.files_transferred !== b.files_transferred) return false
  if (a.bytes_transferred !== b.bytes_transferred) return false
  if (a.checks !== b.checks || a.errors !== b.errors) return false
  if (a.speed_bps !== b.speed_bps) return false
  if (a.current_file !== b.current_file) return false
  const at = a.transfers ?? []
  const bt = b.transfers ?? []
  if (at.length !== bt.length) return false
  for (let i = 0; i < at.length; i++) {
    if (
      at[i].name !== bt[i].name ||
      at[i].status !== bt[i].status ||
      Math.round(at[i].progress) !== Math.round(bt[i].progress)
    ) {
      return false
    }
  }
  return true
}

/** True when a live SSE frame should be applied after a snapshot. */
export function shouldApplyLiveEvent(
  snapshotSeen: boolean,
  snapshotRev: number,
  eventRev: unknown,
): boolean {
  if (!snapshotSeen) return false
  const rev = typeof eventRev === 'number' ? eventRev : Number(eventRev ?? 0)
  if (!Number.isFinite(rev) || rev <= 0) return true
  return rev > snapshotRev
}

export function progressEventToOpStatus(
  topic: string,
  data: Record<string, unknown>,
  prev?: FlowOpSyncStatus,
): FlowOpSyncStatus | null {
  const profile = String(data.profile_id ?? '')
  const colon = profile.indexOf(':')
  if (colon <= 0) return null
  const flowId = profile.slice(0, colon)
  const opId = profile.slice(colon + 1)
  if (!flowId || !opId) return null

  const transferred = Number(data.transferred ?? data.bytes ?? prev?.bytes_transferred ?? 0)
  const total = Number(data.total ?? data.bytes_total ?? prev?.total_bytes ?? 0)
  const filesXfer = Number(data.files_transferred ?? prev?.files_transferred ?? 0)
  const filesTotal = Number(data.total_files ?? prev?.total_files ?? 0)
  let progress = 0
  if (total > 0) progress = Math.min(100, (transferred / total) * 100)
  else if (filesTotal > 0) progress = Math.min(100, (filesXfer / filesTotal) * 100)
  else if (topic === 'sync:completed') progress = 100
  else if (prev) progress = prev.progress

  let status: FlowOpSyncStatus['status'] = 'running'
  if (topic === 'sync:completed') {
    status = 'completed'
  } else if (topic === 'sync:failed') {
    const st = String(data.state ?? 'failed')
    status = st === 'cancelled' ? 'cancelled' : 'failed'
  } else if (String(data.state ?? '') === 'cancelled') {
    status = 'cancelled'
  } else if (String(data.state ?? '') === 'failed') {
    status = 'failed'
  } else if (String(data.state ?? '') === 'completed') {
    status = 'completed'
  }

  return {
    flow_id: flowId,
    op_id: opId,
    task_id: data.task_id ? String(data.task_id) : prev?.task_id,
    action: String(data.action ?? prev?.action ?? 'push'),
    status,
    progress,
    speed_bps: Number(data.bytes_per_sec ?? prev?.speed_bps ?? 0),
    eta_secs: Number(data.eta_secs ?? prev?.eta_secs ?? 0),
    files_transferred: Number(data.files_transferred ?? prev?.files_transferred ?? 0),
    total_files: Number(data.total_files ?? prev?.total_files ?? 0),
    bytes_transferred: transferred,
    total_bytes: total,
    current_file: String(data.current_file ?? prev?.current_file ?? ''),
    stage: data.stage ? String(data.stage) : prev?.stage,
    stage_detail: data.stage_detail ? String(data.stage_detail) : prev?.stage_detail,
    errors: Number(data.errors ?? prev?.errors ?? 0),
    checks: Number(data.checks ?? prev?.checks ?? 0),
    total_checks: Number(data.total_checks ?? prev?.total_checks ?? 0),
    deletes: Number(data.deletes ?? prev?.deletes ?? 0),
    renames: Number(data.renames ?? prev?.renames ?? 0),
    transfers: parseTransfers(data.transfers, prev?.transfers),
    error_message: data.error_message ? String(data.error_message) : prev?.error_message,
    updated_at: Date.now(),
  }
}

export type HydratedRuntime = {
  revision: number
  runStatus: Record<string, string>
  lastError: Record<string, string>
  runLog: Record<string, RuntimeLogEntry[]>
  opSyncStatus: Record<string, FlowOpSyncStatus>
  items: Flow[]
}

export function applyRuntimeSnapshot(
  snap: RuntimeSnapshot,
  items: Flow[],
  previousSync?: Record<string, FlowOpSyncStatus>,
): HydratedRuntime {
  const revision = Number(snap.revision ?? 0)
  const runStatus: Record<string, string> = {}
  const lastError: Record<string, string> = {}
  const runLog: Record<string, RuntimeLogEntry[]> = {}
  const opSyncStatus: Record<string, FlowOpSyncStatus> = {}
  const opByFlow = new Map<string, Map<string, { status: string; last_error?: string }>>()

  for (const f of snap.flows ?? []) {
    if (!f?.id) continue
    runStatus[f.id] = f.status || 'idle'
    if (f.last_error) lastError[f.id] = f.last_error
    runLog[f.id] = (f.log ?? []).map((line) => ({
      at: Number(line.at ?? 0),
      status: String(line.status ?? ''),
      opId: line.op_id || undefined,
      error: line.error || undefined,
      label: line.label || undefined,
    }))
    const opMap = new Map<string, { status: string; last_error?: string }>()
    for (const op of f.ops ?? []) {
      if (!op?.id) continue
      opMap.set(op.id, { status: op.status, last_error: op.last_error })
    }
    opByFlow.set(f.id, opMap)
    if (f.sync && typeof f.sync === 'object') {
      const st = progressEventToOpStatus('sync:progress', f.sync as Record<string, unknown>)
      if (st) opSyncStatus[f.id] = st
    }
  }

  // The runtime hub and the direct WebSocket event fan-out are asynchronous.
  // A snapshot can therefore describe a running flow before its matching
  // progress event has reached the hub. Do not erase richer live transfers
  // while that operation remains active.
  for (const [flowID, live] of Object.entries(previousSync ?? {})) {
    if (runStatus[flowID] !== 'running' && runStatus[flowID] !== 'cancelling') continue
    const snapshotSync = opSyncStatus[flowID]
    if (!snapshotSync || (live.transfers?.length ?? 0) > (snapshotSync.transfers?.length ?? 0)) {
      opSyncStatus[flowID] = live
    }
  }

  const nextItems = items.map((item) => {
    const ops = opByFlow.get(item.id)
    const runtime = runStatus[item.id]
    return {
      ...item,
      status: runtime || item.status || 'idle',
      last_error: lastError[item.id] || item.last_error,
      operations: (item.operations ?? []).map((op) => {
        const rt = ops?.get(op.id)
        return rt
          ? { ...op, status: rt.status, last_error: rt.last_error || op.last_error }
          : { ...op, status: op.status || 'idle' }
      }),
    }
  })

  return { revision, runStatus, lastError, runLog, opSyncStatus, items: nextItems }
}
