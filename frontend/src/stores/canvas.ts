import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Flow, Operation } from '@/api/types'
import {
  connectError,
  fromGraph,
  locationKey,
  newNodeId,
  toGraph,
  type FlowGraph,
  type GraphNode,
} from '@/lib/flowGraph'
import { emptyFlow, emptyOperation, useFlowsStore, withSyncedAction } from '@/stores/flows'

export type CanvasSelection =
  | { kind: 'flow' }
  | { kind: 'node'; id: string }
  | { kind: 'edge'; id: string }

export const useCanvasStore = defineStore('canvas', () => {
  const flows = useFlowsStore()
  const activeFlowId = ref<string | null>(null)
  const selection = ref<CanvasSelection | null>(null)
  const dirty = ref(false)
  let persistTimer: ReturnType<typeof setTimeout> | null = null
  const PERSIST_DEBOUNCE_MS = 400

  function schedulePersist() {
    dirty.value = true
    if (persistTimer) clearTimeout(persistTimer)
    persistTimer = setTimeout(() => {
      persistTimer = null
      void persist()
    }, PERSIST_DEBOUNCE_MS)
  }

  const activeFlow = computed(() => flows.items.find((f) => f.id === activeFlowId.value) ?? null)

  const graph = computed<FlowGraph>(() =>
    activeFlow.value
      ? toGraph(activeFlow.value)
      : { nodes: [], edges: [], viewport: { x: 0, y: 0, zoom: 1 } },
  )

  function selectFlow(id: string | null) {
    activeFlowId.value = id
    selection.value = id ? { kind: 'flow' } : null
    dirty.value = false
    if (id) {
      const g = toGraph(flows.items.find((f) => f.id === id) ?? { id, name: '', operations: [] })
      if (g.edges.length === 1) selection.value = { kind: 'edge', id: g.edges[0].id }
    }
  }

  function select(next: CanvasSelection | null) {
    selection.value = next
  }

  function writeLocal(updated: Flow) {
    flows.items = flows.items.map((f) => (f.id === updated.id ? updated : f))
  }

  function flowLocked(flow = activeFlow.value) {
    return !!flow && flows.isFlowRunning(flow.id)
  }

  async function persist(next?: FlowGraph, flow = activeFlow.value) {
    if (persistTimer) {
      clearTimeout(persistTimer)
      persistTimer = null
    }
    if (!flow || flowLocked(flow)) return
    const g = next ?? toGraph(flow)
    const { operations, canvas_json } = fromGraph(g, flow.operations)
    const updated: Flow = { ...flow, operations, canvas_json }
    writeLocal(updated)
    await flows.save(updated)
    dirty.value = false
  }

  function patchActive(mutate: (g: FlowGraph, flow: Flow) => void, saveNow = false) {
    const flow = activeFlow.value
    if (!flow || flowLocked(flow)) return
    const g = toGraph(flow)
    mutate(g, flow)
    const { operations, canvas_json } = fromGraph(g, flow.operations)
    writeLocal({ ...flow, operations, canvas_json })
    dirty.value = true
    if (saveNow) void persist(g, flow)
  }

  async function addFlow(name: string) {
    const f = emptyFlow()
    f.name = name
    f.operations = [emptyOperation()]
    const g = toGraph(f)
    const { operations, canvas_json } = fromGraph(g, f.operations)
    f.operations = operations
    f.canvas_json = canvas_json
    await flows.save(f)
    selectFlow(f.id)
    return f
  }

  async function addLocation(remote: string, path = '/', x = 80, y = 80) {
    const flow = activeFlow.value
    if (!flow || flowLocked(flow)) return null
    const g = toGraph(flow)
    const key = locationKey(remote, path)
    const existing = g.nodes.find((n) => n.key === key)
    if (existing) {
      selection.value = { kind: 'node', id: existing.id }
      return existing.id
    }
    const node: GraphNode = {
      id: newNodeId(`${key}-${Date.now()}`),
      remote,
      path,
      label: '',
      x,
      y,
      key,
    }
    g.nodes.push(node)
    dirty.value = true
    await persist(g, flow)
    selection.value = { kind: 'node', id: node.id }
    return node.id
  }

  async function connect(sourceId: string, targetId: string, action = 'push') {
    const flow = activeFlow.value
    if (!flow || flowLocked(flow)) return { ok: false as const, error: 'no-flow' }
    const g = toGraph(flow)
    const err = connectError(sourceId, targetId, action, g)
    if (err) return { ok: false as const, error: err }
    const op: Operation = withSyncedAction({
      ...emptyOperation(),
      source_remote: g.nodes.find((n) => n.id === sourceId)?.remote ?? '',
      source_path: g.nodes.find((n) => n.id === sourceId)?.path ?? '/',
      target_remote: g.nodes.find((n) => n.id === targetId)?.remote ?? '',
      target_path: g.nodes.find((n) => n.id === targetId)?.path ?? '/',
      action,
    })
    g.edges.push({
      id: op.id,
      source: sourceId,
      target: targetId,
      action,
      operation: op,
    })
    dirty.value = true
    await persist(g, flow)
    selection.value = { kind: 'edge', id: op.id }
    return { ok: true as const, id: op.id }
  }

  async function removeNode(id: string) {
    patchActive((g) => {
      g.edges = g.edges.filter((e) => e.source !== id && e.target !== id)
      g.nodes = g.nodes.filter((n) => n.id !== id)
    }, true)
    selection.value = { kind: 'flow' }
  }

  async function removeEdge(id: string) {
    patchActive((g) => {
      g.edges = g.edges.filter((e) => e.id !== id)
    }, true)
    selection.value = { kind: 'flow' }
  }

  function updateNode(id: string, patch: { remote?: string; path?: string; label?: string }) {
    patchActive((g) => {
      const n = g.nodes.find((x) => x.id === id)
      if (!n) return
      if (patch.remote !== undefined) n.remote = patch.remote
      if (patch.path !== undefined) n.path = patch.path
      if (patch.label !== undefined) n.label = patch.label
      n.key = locationKey(n.remote, n.path)
    })
    schedulePersist()
  }

  function updateEdge(id: string, patch: Partial<Operation>) {
    patchActive((g) => {
      const e = g.edges.find((x) => x.id === id)
      if (!e) return
      e.operation = withSyncedAction({ ...e.operation, ...patch })
      if (patch.action) e.action = patch.action
    })
    schedulePersist()
  }

  async function updatePositions(
    positions: Array<{ id: string; x: number; y: number }>,
    viewport?: FlowGraph['viewport'],
  ) {
    const flow = activeFlow.value
    if (!flow || flowLocked(flow)) return
    const g = toGraph(flow)
    const map = new Map(positions.map((p) => [p.id, p]))
    for (const n of g.nodes) {
      const p = map.get(n.id)
      if (p) {
        n.x = p.x
        n.y = p.y
      }
    }
    if (viewport) g.viewport = viewport
    await persist(g, flow)
  }

  async function updateFlowMeta(patch: Partial<Flow>) {
    const flow = activeFlow.value
    if (!flow || flowLocked(flow)) return
    dirty.value = true
    await flows.save({ ...flow, ...patch })
    dirty.value = false
  }

  return {
    activeFlowId,
    activeFlow,
    selection,
    dirty,
    graph,
    selectFlow,
    select,
    addFlow,
    addLocation,
    connect,
    removeNode,
    removeEdge,
    updateNode,
    updateEdge,
    updatePositions,
    updateFlowMeta,
    persist,
  }
})
