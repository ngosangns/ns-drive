/**
 * Subscribe to backend SSE (/api/v1/events). First frame is runtime:snapshot;
 * later frames update Pinia. Document changes arrive as state:changed.
 */
import { onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import { useFlowsStore } from '@/stores/flows'
import { useRemotesStore } from '@/stores/remotes'
import { useCanvasStore } from '@/stores/canvas'
import { shouldApplyLiveEvent } from '@/lib/runtimeSync'
import type { RuntimeSnapshot } from '@/api/types'

const SYNC_TOPICS = ['sync:started', 'sync:progress', 'sync:completed', 'sync:failed'] as const
const FLOW_RUN_TOPIC = 'flow:execution'
const LEGACY_FLOW_TOPIC = 'board:execution'

export type UseEventStreamOptions = {
  enabled?: Ref<boolean>
}

export function useEventStream(opts: UseEventStreamOptions = {}) {
  const flows = useFlowsStore()
  const remotes = useRemotesStore()
  const connected = ref(false)
  let es: EventSource | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let intentionalClose = false

  function applyFlowPayload(data: Record<string, unknown>) {
    const id = String(data.flow_id ?? data.board_id ?? '')
    const status = String(data.status ?? '')
    const opId = data.op_id
      ? String(data.op_id)
      : data.node_id
        ? String(data.node_id)
        : undefined
    let error: string | undefined
    if (data.error) error = String(data.error)
    else if (status === 'failed' && data.action) error = String(data.action)
    if (!id || !status) return
    flows.applyRunEvent(id, status, opId || undefined, error)
  }

  function connect() {
    intentionalClose = false
    if (es) {
      es.close()
      es = null
    }
    let snapshotSeen = false
    const queued: Array<{ topic: string; data: Record<string, unknown> }> = []

    function handleLive(topic: string, data: Record<string, unknown>) {
      if (!shouldApplyLiveEvent(snapshotSeen, flows.runtimeRevision, data.revision)) {
        if (!snapshotSeen) queued.push({ topic, data })
        return
      }
      if ((SYNC_TOPICS as readonly string[]).includes(topic)) {
        flows.applySyncProgress(topic, data)
        return
      }
      if (topic === FLOW_RUN_TOPIC || topic === LEGACY_FLOW_TOPIC) {
        applyFlowPayload(data)
      }
    }

    es = new EventSource('/api/v1/events')
    es.onopen = () => {
      connected.value = true
    }
    es.onerror = () => {
      connected.value = false
      es?.close()
      es = null
      if (intentionalClose) return
      if (reconnectTimer) clearTimeout(reconnectTimer)
      reconnectTimer = setTimeout(connect, 2_000)
    }

    es.addEventListener('runtime:snapshot', (ev) => {
      let data: RuntimeSnapshot = {}
      try {
        data = JSON.parse((ev as MessageEvent).data) as RuntimeSnapshot
      } catch {
        return
      }
      flows.hydrateRuntime(data)
      snapshotSeen = true
      const pending = queued.splice(0)
      for (const q of pending) handleLive(q.topic, q.data)
    })

    es.addEventListener('state:changed', (ev) => {
      let data: Record<string, unknown> = {}
      try {
        data = JSON.parse((ev as MessageEvent).data) as Record<string, unknown>
      } catch {
        return
      }
      const domain = String(data.domain ?? '')
      if (domain === 'remotes') {
        void remotes.load()
        return
      }
      if (domain === 'flows' || domain === '') {
        try {
          if (useCanvasStore().dirty) return
        } catch {
          /* pinia not ready */
        }
        void flows.load()
      }
    })

    for (const topic of SYNC_TOPICS) {
      es.addEventListener(topic, (ev) => {
        let data: Record<string, unknown> = {}
        try {
          data = JSON.parse((ev as MessageEvent).data) as Record<string, unknown>
        } catch {
          return
        }
        handleLive(topic, data)
      })
    }
    for (const topic of [FLOW_RUN_TOPIC, LEGACY_FLOW_TOPIC]) {
      es.addEventListener(topic, (ev) => {
        let data: Record<string, unknown> = {}
        try {
          data = JSON.parse((ev as MessageEvent).data) as Record<string, unknown>
        } catch {
          return
        }
        handleLive(topic, data)
      })
    }
  }

  function disconnect() {
    intentionalClose = true
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    es?.close()
    es = null
    connected.value = false
  }

  if (opts.enabled) {
    watch(
      opts.enabled,
      (on) => {
        if (on) connect()
        else disconnect()
      },
      { immediate: true },
    )
  } else {
    onMounted(() => {
      connect()
    })
  }

  onUnmounted(() => {
    disconnect()
  })

  return { connected, connect, disconnect }
}
