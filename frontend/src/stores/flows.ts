import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { useApi } from '@/composables/useApi'
import type { Flow, FlowOpSyncStatus, Operation, RuntimeSnapshot } from '@/api/types'
import { normalizeFlowAction } from '@/constants/forms'
import {
  applyRuntimeSnapshot,
  progressEventToOpStatus,
  syncSnapEqual,
  type RuntimeLogEntry,
} from '@/lib/runtimeSync'

function newId(): string {
  return crypto.randomUUID()
}

/** Wails createEmptyOperation: action lives in sync_config and top-level column. */
export function emptyOperation(): Operation {
  return {
    id: newId(),
    source_remote: '',
    source_path: '/',
    target_remote: '',
    target_path: '/out',
    action: 'push',
    sync_config: { action: 'push' },
    is_expanded: true,
    sort_order: 0,
    status: 'idle',
  }
}

/**
 * Effective flow action: column first, then sync_config.action.
 * Always normalized to push | bi | bi-resync (pull not allowed on flows).
 */
export function resolveOpAction(op: Operation): string {
  const a = (op.action || '').trim()
  if (a) return normalizeFlowAction(a)
  const sc = op.sync_config as Record<string, unknown> | null | undefined
  if (sc && typeof sc.action === 'string' && sc.action.trim()) {
    return normalizeFlowAction(sc.action.trim())
  }
  return 'push'
}

/** Keep action column and sync_config.action aligned before persist. */
export function withSyncedAction(op: Operation, action?: string): Operation {
  const a = normalizeFlowAction(action ?? resolveOpAction(op))
  const prev = (op.sync_config && typeof op.sync_config === 'object' ? op.sync_config : {}) as Record<
    string,
    unknown
  >
  return {
    ...op,
    action: a,
    sync_config: { ...prev, action: a },
  }
}

export function emptyFlow(): Flow {
  return {
    id: newId(),
    name: '',
    is_collapsed: false,
    schedule_enabled: false,
    enabled: false,
    operations: [],
    status: 'idle',
  }
}

export type FlowRunLogEntry = RuntimeLogEntry

export const useFlowsStore = defineStore('flows', () => {
  const api = useApi()
  const items = ref<Flow[]>([])
  /** Runtime status from execute/SSE keyed by flow id. */
  const runStatus = ref<Record<string, string>>({})
  /** Timeline of the latest run per flow (for status panel). */
  const runLog = ref<Record<string, FlowRunLogEntry[]>>({})
  const lastError = ref<Record<string, string>>({})
  /**
   * Live rclone stats for the active op of each flow (Wails SyncStatus).
   * Keyed by flow id. busyKey on backend is `${flowId}:${opId}`.
   */
  const opSyncStatus = ref<Record<string, FlowOpSyncStatus>>({})
  const runtimeRevision = ref(-1)

  const runningFlowIds = computed(() => {
    const s = new Set<string>()
    for (const [id, st] of Object.entries(runStatus.value)) {
      if (st === 'running' || st === 'cancelling') s.add(id)
    }
    for (const f of items.value) {
      if (f.status === 'running' || f.status === 'cancelling') s.add(f.id)
    }
    return s
  })

  function isFlowRunning(id: string): boolean {
    return runningFlowIds.value.has(id)
  }

  function flowStatusOf(id: string): string {
    return runStatus.value[id] || items.value.find((f) => f.id === id)?.status || 'idle'
  }

  function logOf(id: string): FlowRunLogEntry[] {
    return runLog.value[id] ?? []
  }

  function setRunStatus(id: string, status: string) {
    // Replace object so Pinia/Vue always notify dependents.
    runStatus.value = { ...runStatus.value, [id]: status }
  }

  function appendLog(id: string, entry: FlowRunLogEntry) {
    const prev = runLog.value[id] ?? []
    // Cap length so the panel stays readable.
    const next = [...prev, entry].slice(-40)
    runLog.value = { ...runLog.value, [id]: next }
  }

  function clearRunUi(id: string) {
    runLog.value = { ...runLog.value, [id]: [] }
    const { [id]: _drop, ...rest } = lastError.value
    lastError.value = rest
    const { [id]: _s, ...syncRest } = opSyncStatus.value
    opSyncStatus.value = syncRest
  }

  function activeSyncOf(flowId: string): FlowOpSyncStatus | null {
    return opSyncStatus.value[flowId] ?? null
  }

  /** Map syncengine SSE (profile_id = flowId:opId) onto Wails-style status. */
  function applySyncProgress(topic: string, data: Record<string, unknown>) {
    const prev = (() => {
      const profile = String(data.profile_id ?? '')
      const colon = profile.indexOf(':')
      if (colon <= 0) return undefined
      return opSyncStatus.value[profile.slice(0, colon)]
    })()
    const snap = progressEventToOpStatus(topic, data, prev)
    if (!snap) return
    const { flow_id: flowId, op_id: opId, status } = snap

    if (prev && syncSnapEqual(prev, snap)) return
    opSyncStatus.value = { ...opSyncStatus.value, [flowId]: snap }

    const prevOpStatus = items.value
      .find((f) => f.id === flowId)
      ?.operations?.find((o) => o.id === opId)?.status
    const lifecycleChanged =
      !prev || prev.status !== status || prev.op_id !== opId || prevOpStatus !== status
    if (!lifecycleChanged) return

    if (status === 'running') applyRunEvent(flowId, 'running', opId)
    else if (status === 'failed') applyRunEvent(flowId, 'failed', opId, snap.error_message)
    else if (status === 'completed') applyRunEvent(flowId, 'completed', opId)
    else if (status === 'cancelled') applyRunEvent(flowId, 'cancelled', opId)
  }

  function hydrateRuntime(snap: RuntimeSnapshot) {
    const rev = Number(snap.revision ?? 0)
    if (runtimeRevision.value >= 0 && rev < runtimeRevision.value) return
    const empty = !(snap.flows ?? []).length
    const locallyRunning = Object.values(runStatus.value).some(
      (s) => s === 'running' || s === 'cancelling',
    )
    if (empty && locallyRunning && rev <= Math.max(runtimeRevision.value, 0)) return
    const next = applyRuntimeSnapshot(snap, items.value, opSyncStatus.value)
    runtimeRevision.value = next.revision
    runStatus.value = next.runStatus
    lastError.value = next.lastError
    runLog.value = next.runLog
    opSyncStatus.value = next.opSyncStatus
    if (items.value.length) items.value = next.items
  }

  async function pullRuntime() {
    try {
      const snap = await api.get<RuntimeSnapshot>('/api/v1/runtime')
      hydrateRuntime(snap)
    } catch {
      /* locked / offline */
    }
  }

  async function load() {
    const list = (await api.get<Flow[]>('/api/v1/flows')) ?? []
    hydrate(list)
    await pullRuntime()
  }

  function hydrate(list: Flow[] | null | undefined) {
    items.value = (list ?? []).map((f) => ({
      ...f,
      operations: (f.operations ?? []).map((op) => withSyncedAction(op)),
      schedule_enabled: f.schedule_enabled ?? f.enabled ?? false,
      enabled: f.schedule_enabled ?? f.enabled ?? false,
      schedule_cron: f.schedule_cron || f.cron_expr || '',
      cron_expr: f.cron_expr || f.schedule_cron || '',
      status: runStatus.value[f.id] || f.status || 'idle',
      last_error: lastError.value[f.id] || f.last_error,
    }))
  }

  async function save(f: Flow) {
    const body: Flow = {
      ...f,
      schedule_enabled: f.schedule_enabled ?? f.enabled,
      enabled: f.schedule_enabled ?? f.enabled,
      cron_expr: f.cron_expr || f.schedule_cron,
      schedule_cron: f.schedule_cron || f.cron_expr,
      operations: (f.operations ?? []).map((op, i) => {
        const synced = withSyncedAction(op)
        return {
          ...synced,
          flow_id: f.id,
          sort_order: op.sort_order ?? i,
        }
      }),
    }
    const exists = items.value.some((x) => x.id === f.id)
    if (exists) {
      await api.put(`/api/v1/flows/${encodeURIComponent(f.id)}`, body)
    } else {
      await api.post('/api/v1/flows', body)
    }
    await load()
  }

  async function add(f: Flow) {
    await save(f)
  }

  async function remove(id: string) {
    await api.del(`/api/v1/flows/${encodeURIComponent(id)}`)
    const next = { ...runStatus.value }
    delete next[id]
    runStatus.value = next
    const nextLog = { ...runLog.value }
    delete nextLog[id]
    runLog.value = nextLog
    const nextErr = { ...lastError.value }
    delete nextErr[id]
    lastError.value = nextErr
    await load()
  }

  async function execute(id: string) {
    clearRunUi(id)
    setRunStatus(id, 'running')
    const idx = items.value.findIndex((f) => f.id === id)
    if (idx >= 0) {
      const ops = (items.value[idx].operations ?? []).map((op) => ({ ...op, status: 'idle' as const }))
      items.value = items.value.map((x, i) =>
        i === idx ? { ...x, status: 'running', operations: ops, last_error: undefined } : x,
      )
    }
    appendLog(id, { at: Date.now(), status: 'running', label: 'Flow' })
    try {
      await api.post(`/api/v1/flows/${encodeURIComponent(id)}/execute`)
    } catch (e) {
      setRunStatus(id, 'failed')
      const msg = e instanceof Error ? e.message : String(e)
      lastError.value = { ...lastError.value, [id]: msg }
      appendLog(id, { at: Date.now(), status: 'failed', error: msg, label: 'Flow' })
      if (idx >= 0) {
        items.value = items.value.map((x, i) =>
          i === idx ? { ...x, status: 'failed', last_error: msg } : x,
        )
      }
      throw e
    }
  }

  async function stop(id: string) {
    setRunStatus(id, 'cancelling')
    appendLog(id, { at: Date.now(), status: 'cancelling', label: 'Flow' })
    const idx = items.value.findIndex((f) => f.id === id)
    if (idx >= 0) {
      items.value = items.value.map((x, i) => (i === idx ? { ...x, status: 'cancelling' } : x))
    }
    await api.post(`/api/v1/flows/${encodeURIComponent(id)}/stop`)
  }

  /**
   * Apply a flow / operation runtime event from SSE or a runtime snapshot.
   * Operation-level "completed" must NOT mark the whole flow completed —
   * only flow-level events (no opId) set the terminal flow status.
   */
  function applyRunEvent(flowId: string, status: string, opId?: string, error?: string) {
    if (!flowId || !status) return
    const idx = items.value.findIndex((f) => f.id === flowId)

    // Skip duplicate lifecycle events (progress ticks used to re-fire "running").
    {
      const curFlow = runStatus.value[flowId] || (idx >= 0 ? items.value[idx].status : '') || 'idle'
      if (opId && idx >= 0) {
        const curOp = items.value[idx].operations?.find((o) => o.id === opId)?.status
        const flowOk =
          status === 'running' || status === 'cancelling'
            ? curFlow === 'running' || curFlow === 'cancelling'
            : status === 'completed'
              ? curFlow === 'running' || curFlow === 'cancelling' || curFlow === status
              : curFlow === status
        // For op-level "running"/"completed": skip if op already has that status
        // and flow lifecycle doesn't need updating.
        if (curOp === status && !error) {
          if (status === 'running' && (curFlow === 'running' || curFlow === 'cancelling')) return
          if (status === 'completed' && flowOk) return
          if (status === 'failed' || status === 'cancelled') {
            if (curFlow === status) return
          }
        }
      } else if (!opId && curFlow === status && !error) {
        return
      }
    }

    // Resolve op label for log.
    let label = 'Flow'
    if (opId && idx >= 0) {
      const ops = items.value[idx].operations ?? []
      const oi = ops.findIndex((o) => o.id === opId)
      label = oi >= 0 ? `Op #${oi + 1}` : `Op ${opId.slice(0, 8)}`
    } else if (opId) {
      label = `Op ${opId.slice(0, 8)}`
    }

    // Deduplicate consecutive identical log lines.
    const prevLog = runLog.value[flowId] ?? []
    const last = prevLog[prevLog.length - 1]
    if (!last || last.status !== status || last.opId !== opId || last.error !== error) {
      appendLog(flowId, {
        at: Date.now(),
        status,
        opId,
        error: error || undefined,
        label,
      })
    }

    // Don't store cancel/kill noise as lastError (UI shows "Stopped" instead).
    if (error && status !== 'cancelled' && status !== 'cancelling') {
      const lower = error.toLowerCase()
      if (
        !lower.includes('signal: killed') &&
        !lower.includes('context canceled') &&
        !lower.includes('task cancelled')
      ) {
        lastError.value = { ...lastError.value, [flowId]: error }
      }
    }
    if (status === 'cancelled' || status === 'cancelling') {
      const { [flowId]: _drop, ...rest } = lastError.value
      lastError.value = rest
      // Clear raw error on op sync snapshot too.
      const snap = opSyncStatus.value[flowId]
      if (snap?.error_message) {
        opSyncStatus.value = {
          ...opSyncStatus.value,
          [flowId]: { ...snap, error_message: undefined, status: 'cancelled' },
        }
      }
    }

    if (idx < 0) {
      if (!opId) {
        setRunStatus(flowId, status)
      } else if (status === 'running' || status === 'cancelling') {
        setRunStatus(flowId, status)
      } else if (status === 'failed' || status === 'cancelled') {
        setRunStatus(flowId, status)
      }
      return
    }

    let flowStatus = runStatus.value[flowId] || items.value[idx].status || 'idle'
    const ops = [...(items.value[idx].operations ?? [])]

    if (opId) {
      for (let i = 0; i < ops.length; i++) {
        if (ops[i].id !== opId) continue
        ops[i] = {
          ...ops[i],
          status,
          ...(error ? { last_error: error } : {}),
        }
        break
      }
      if (status === 'running') flowStatus = 'running'
      else if (status === 'cancelling') flowStatus = 'cancelling'
      else if (status === 'failed' || status === 'cancelled') flowStatus = status
    } else {
      flowStatus = status
      if (status === 'completed' || status === 'failed' || status === 'cancelled') {
        for (let i = 0; i < ops.length; i++) {
          if (ops[i].status === 'running' || ops[i].status === 'cancelling') {
            ops[i] = {
              ...ops[i],
              status: status === 'completed' ? 'completed' : status,
            }
          }
        }
      }
    }

    setRunStatus(flowId, flowStatus)
    items.value = items.value.map((x, i) =>
      i === idx
        ? {
            ...x,
            status: flowStatus,
            operations: ops,
            ...(error && !opId ? { last_error: error } : {}),
          }
        : x,
    )

  }

  return {
    items,
    runStatus,
    runLog,
    lastError,
    opSyncStatus,
    runningFlowIds,
    isFlowRunning,
    flowStatusOf,
    logOf,
    activeSyncOf,
    hydrate,
    load,
    save,
    add,
    remove,
    execute,
    stop,
    hydrateRuntime,
    pullRuntime,
    runtimeRevision,
    applyRunEvent,
    applySyncProgress,
    emptyFlow,
    emptyOperation,
    error: api.error,
    loading: api.loading,
  }
})
