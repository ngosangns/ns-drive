/**
 * Subscribe to backend SSE (/api/v1/events). First frame is runtime:snapshot;
 * later frames update Pinia. Document changes arrive as state:changed.
 */
import { onMounted, onUnmounted, ref, watch, type Ref } from 'vue'
import { useFlowsStore } from '@/stores/flows'
import { useRemotesStore } from '@/stores/remotes'
import { useCanvasStore } from '@/stores/canvas'
import {
  FLOW_RUN_TOPIC,
  LEGACY_FLOW_TOPIC,
  SYNC_TOPICS,
  acceptLive,
  isFlowTopic,
  isSyncTopic,
  parseFlowPayload,
  parseJsonObject,
  parseSnapshot,
  shouldReloadFlows,
  shouldReloadRemotes,
} from '@/lib/sseRuntime'

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
    const parsed = parseFlowPayload(data)
    if (!parsed) return
    flows.applyRunEvent(parsed.id, parsed.status, parsed.opId, parsed.error)
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
      if (!acceptLive(snapshotSeen, Number(flows.runtimeRevision ?? 0), data)) {
        if (!snapshotSeen) queued.push({ topic, data })
        return
      }
      if (isSyncTopic(topic)) {
        flows.applySyncProgress(topic, data)
        return
      }
      if (isFlowTopic(topic)) applyFlowPayload(data)
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
      const data = parseSnapshot((ev as MessageEvent).data)
      if (!data) return
      flows.hydrateRuntime(data)
      snapshotSeen = true
      const pending = queued.splice(0)
      for (const q of pending) handleLive(q.topic, q.data)
    })

    es.addEventListener('state:changed', (ev) => {
      const data = parseJsonObject((ev as MessageEvent).data)
      if (!data) return
      const domain = String(data.domain ?? '')
      if (shouldReloadRemotes(domain)) {
        void remotes.load()
        return
      }
      let dirty = false
      try {
        dirty = !!useCanvasStore().dirty
      } catch {
        /* pinia not ready */
      }
      if (shouldReloadFlows(domain, dirty)) void flows.load()
    })

    for (const topic of [...SYNC_TOPICS, FLOW_RUN_TOPIC, LEGACY_FLOW_TOPIC]) {
      es.addEventListener(topic, (ev) => {
        const data = parseJsonObject((ev as MessageEvent).data)
        if (!data) return
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
