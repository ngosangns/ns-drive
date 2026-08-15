import assert from 'node:assert/strict'
import { afterEach, beforeEach, describe, it } from 'node:test'
import { createPinia, setActivePinia } from 'pinia'
import { effectScope, ref } from 'vue'
import { useEventStream } from './useEventStream'

class MockWebSocket {
  static instances: MockWebSocket[] = []

  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null

  constructor(readonly url: string) {
    MockWebSocket.instances.push(this)
  }

  close() {
    this.onclose?.()
  }
}

let scope: ReturnType<typeof effectScope> | null = null
const originalWebSocket = globalThis.WebSocket

beforeEach(() => {
  MockWebSocket.instances = []
  globalThis.window = {
    location: { protocol: 'http:', host: 'localhost:8080' },
  } as unknown as Window & typeof globalThis
  globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket
  setActivePinia(createPinia())
})

afterEach(() => {
  scope?.stop()
  scope = null
  globalThis.WebSocket = originalWebSocket
  delete (globalThis as unknown as { window?: Window }).window
})

describe('useEventStream connection state', () => {
  it('marks the stream disconnected immediately when the socket errors', () => {
    const enabled = ref(true)
    scope = effectScope()
    const result = scope.run(() => useEventStream({ enabled }))

    assert.ok(result)
    const socket = MockWebSocket.instances[0]
    socket.onopen?.()
    assert.equal(result.state.value, 'connected')

    socket.onerror?.()
    assert.equal(result.state.value, 'disconnected')
    assert.equal(result.connected.value, false)
    result.disconnect()
  })
})
