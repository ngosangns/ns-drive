/**
 * Keep Pinia hydrated from the backend-owned WebSocket projection. REST writes
 * remain authoritative only after the backend persists and broadcasts a new
 * snapshot, so reconnecting or reloading never depends on browser state.
 */
import {
  getCurrentInstance,
  onMounted,
  onUnmounted,
  ref,
  watch,
  type Ref,
} from 'vue'
import { useFlowsStore } from '@/stores/flows'
import { useRemotesStore } from '@/stores/remotes'
import { isFlowTopic, isSyncTopic, parseFlowPayload } from '@/lib/sseRuntime'
import type { StateSnapshot } from '@/api/types'

export type UseEventStreamOptions = { enabled?: Ref<boolean> }

export type EventStreamState = 'connecting' | 'connected' | 'disconnected'

type StateFrame = {
  type?: string
  snapshot?: StateSnapshot
  topic?: string
  data?: unknown
}

function stateSocketURL(): string {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/api/v1/state`
}

function parseFrame(raw: string): StateFrame | null {
  try {
    const data = JSON.parse(raw) as StateFrame
    return data && typeof data === 'object' ? data : null
  } catch {
    return null
  }
}

export function useEventStream(opts: UseEventStreamOptions = {}) {
  const flows = useFlowsStore()
  const remotes = useRemotesStore()
  const connected = ref(false)
  const state = ref<EventStreamState>('connecting')
  let socket: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let intentionalClose = false
  let reconnectDelay = 500

  function hydrate(snapshot: StateSnapshot) {
    flows.hydrate(snapshot.flows)
    remotes.hydrate(snapshot.remotes)
    flows.hydrateRuntime(snapshot.runtime)
  }

  function applyRuntimeEvent(topic: string, raw: unknown) {
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return
    const data = raw as Record<string, unknown>
    if (isSyncTopic(topic)) {
      flows.applySyncProgress(topic, data)
      return
    }
    if (isFlowTopic(topic)) {
      const event = parseFlowPayload(data)
      if (event) flows.applyRunEvent(event.id, event.status, event.opId, event.error)
    }
  }

  function scheduleReconnect() {
    if (intentionalClose || reconnectTimer) return
    const wait = reconnectDelay
    reconnectDelay = Math.min(reconnectDelay * 2, 5_000)
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, wait)
  }

  function connect() {
    intentionalClose = false
    state.value = 'connecting'
    socket?.close()
    socket = new WebSocket(stateSocketURL())
    socket.onopen = () => {
      connected.value = true
      state.value = 'connected'
      reconnectDelay = 500
    }
    socket.onmessage = (event) => {
      const frame = parseFrame(String(event.data))
      if (frame?.type === 'state.snapshot' && frame.snapshot) hydrate(frame.snapshot)
      if (frame?.type === 'runtime.event' && frame.topic) applyRuntimeEvent(frame.topic, frame.data)
    }
    socket.onerror = () => {
      connected.value = false
      state.value = 'disconnected'
      socket?.close()
    }
    socket.onclose = () => {
      connected.value = false
      state.value = 'disconnected'
      socket = null
      scheduleReconnect()
    }
  }

  function disconnect() {
    intentionalClose = true
    if (reconnectTimer) clearTimeout(reconnectTimer)
    reconnectTimer = null
    socket?.close()
    socket = null
    connected.value = false
    state.value = 'disconnected'
  }

  if (opts.enabled) {
    watch(opts.enabled, (on) => (on ? connect() : disconnect()), { immediate: true })
  } else {
    onMounted(connect)
  }
  if (getCurrentInstance()) onUnmounted(disconnect)

  return { connected, state, connect, disconnect }
}
