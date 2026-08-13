/**
 * Pure mapping between a Flow (operations) and a location-node / sync-edge graph.
 * Canvas node ids live in flow.canvas_json; operations keep source/target paths
 * so flowengine can run without reading the canvas.
 */
import type { Flow, Operation } from '@/api/types'
import { normalizeFlowAction } from '@/constants/forms'

export const DEFAULT_COL_GAP = 280
export const DEFAULT_ROW_GAP = 140

export interface CanvasLocation {
  id: string
  remote: string
  path: string
  label: string
  x: number
  y: number
}

export interface FlowCanvas {
  viewport: { x: number; y: number; zoom: number }
  nodes: CanvasLocation[]
}

export interface GraphNode extends CanvasLocation {
  key: string
}

export interface GraphEdge {
  id: string
  source: string
  target: string
  action: string
  operation: Operation
}

export interface FlowGraph {
  nodes: GraphNode[]
  edges: GraphEdge[]
  viewport: FlowCanvas['viewport']
}

export function locationKey(remote: string, path: string): string {
  const r = (remote ?? '').trim()
  const p = normalizePath(path)
  return `${r}\0${p}`
}

export function normalizePath(path: string | undefined | null): string {
  const p = (path ?? '').trim()
  return p || '/'
}

export function parseCanvas(raw: unknown): FlowCanvas {
  const empty: FlowCanvas = { viewport: { x: 0, y: 0, zoom: 1 }, nodes: [] }
  if (!raw || typeof raw !== 'object') return empty
  const o = raw as Record<string, unknown>
  const vp = o.viewport && typeof o.viewport === 'object' ? (o.viewport as Record<string, unknown>) : {}
  const nodesIn = Array.isArray(o.nodes) ? o.nodes : []
  return {
    viewport: {
      x: num(vp.x, 0),
      y: num(vp.y, 0),
      zoom: num(vp.zoom, 1) || 1,
    },
    nodes: nodesIn
      .filter((n): n is Record<string, unknown> => !!n && typeof n === 'object')
      .map((n, i) => ({
        id: String(n.id || `n_${i}`),
        remote: String(n.remote ?? ''),
        path: normalizePath(String(n.path ?? '/')),
        label: String(n.label ?? ''),
        x: num(n.x, 0),
        y: num(n.y, i * DEFAULT_ROW_GAP),
      })),
  }
}

export function defaultLayout(ops: Operation[]): FlowCanvas {
  const nodes: CanvasLocation[] = []
  const seen = new Map<string, CanvasLocation>()
  ;(ops ?? []).forEach((op, i) => {
    place(op.source_remote, op.source_path, 0, i * DEFAULT_ROW_GAP)
    place(op.target_remote, op.target_path, DEFAULT_COL_GAP, i * DEFAULT_ROW_GAP)
  })
  return { viewport: { x: 0, y: 0, zoom: 1 }, nodes }

  function place(remote: string, path: string, x: number, y: number) {
    const key = locationKey(remote, path)
    if (seen.has(key)) return
    const node: CanvasLocation = {
      id: stableNodeId(key),
      remote: (remote ?? '').trim(),
      path: normalizePath(path),
      label: '',
      x,
      y,
    }
    seen.set(key, node)
    nodes.push(node)
  }
}

/** Build a graph: unique (remote, path) → node; each operation → edge. */
export function toGraph(flow: Flow): FlowGraph {
  const ops = flow.operations ?? []
  const stored = parseCanvas(flow.canvas_json)
  const byKey = new Map<string, GraphNode>()
  const byId = new Map<string, GraphNode>()

  for (const n of stored.nodes) {
    const key = locationKey(n.remote, n.path)
    const node: GraphNode = { ...n, path: normalizePath(n.path), key }
    byKey.set(key, node)
    byId.set(node.id, node)
  }

  const needed = new Set<string>()
  for (const op of ops) {
    needed.add(locationKey(op.source_remote, op.source_path))
    needed.add(locationKey(op.target_remote, op.target_path))
  }

  if (stored.nodes.length === 0 && ops.length > 0) {
    const laid = defaultLayout(ops)
    for (const n of laid.nodes) {
      const key = locationKey(n.remote, n.path)
      const node: GraphNode = { ...n, key }
      byKey.set(key, node)
      byId.set(node.id, node)
    }
  } else {
    let extra = 0
    for (const key of needed) {
      if (byKey.has(key)) continue
      const [remote, path] = key.split('\0')
      const node: GraphNode = {
        id: stableNodeId(key),
        remote,
        path: normalizePath(path),
        label: '',
        x: DEFAULT_COL_GAP,
        y: extra * DEFAULT_ROW_GAP,
        key,
      }
      extra++
      byKey.set(key, node)
      byId.set(node.id, node)
    }
  }

  const edges: GraphEdge[] = []
  for (const op of ops) {
    const src = byKey.get(locationKey(op.source_remote, op.source_path))
    const dst = byKey.get(locationKey(op.target_remote, op.target_path))
    if (!src || !dst) continue
    edges.push({
      id: op.id,
      source: src.id,
      target: dst.id,
      action: normalizeFlowAction(op.action),
      operation: op,
    })
  }

  return {
    nodes: [...byKey.values()],
    edges,
    viewport: stored.viewport,
  }
}

export function connectError(
  sourceId: string,
  targetId: string,
  action: string,
  graph: Pick<FlowGraph, 'nodes' | 'edges'>,
): string | null {
  if (!sourceId || !targetId) return 'missing-endpoint'
  if (sourceId === targetId) return 'self-loop'
  const src = graph.nodes.find((n) => n.id === sourceId)
  const dst = graph.nodes.find((n) => n.id === targetId)
  if (!src || !dst) return 'unknown-node'
  const act = normalizeFlowAction(action)
  const dup = graph.edges.some(
    (e) => e.source === sourceId && e.target === targetId && normalizeFlowAction(e.action) === act,
  )
  if (dup) return 'duplicate-edge'
  return null
}

/**
 * Project graph back onto operations + canvas_json.
 * Edge ids are operation ids; node remote/path overwrite the op slots.
 */
export function fromGraph(
  graph: FlowGraph,
  previousOps: Operation[] = [],
): { operations: Operation[]; canvas_json: FlowCanvas } {
  const nodesById = new Map(graph.nodes.map((n) => [n.id, n]))
  const prevById = new Map((previousOps ?? []).map((op) => [op.id, op]))
  const operations: Operation[] = []

  graph.edges.forEach((e, i) => {
    const src = nodesById.get(e.source)
    const dst = nodesById.get(e.target)
    if (!src || !dst) return
    const prev = prevById.get(e.id) ?? e.operation
    const action = normalizeFlowAction(e.action || prev?.action)
    const sc =
      prev?.sync_config && typeof prev.sync_config === 'object' ? { ...prev.sync_config } : {}
    operations.push({
      ...(prev ?? {}),
      id: e.id,
      source_remote: src.remote,
      source_path: normalizePath(src.path),
      target_remote: dst.remote,
      target_path: normalizePath(dst.path),
      action,
      sync_config: { ...sc, action },
      sort_order: i,
    })
  })

  return {
    operations,
    canvas_json: {
      viewport: graph.viewport ?? { x: 0, y: 0, zoom: 1 },
      nodes: graph.nodes.map(({ id, remote, path, label, x, y }) => ({
        id,
        remote,
        path: normalizePath(path),
        label: label ?? '',
        x,
        y,
      })),
    },
  }
}

/** Move a location node; every operation that used the old slot follows. */
export function relocateNode(
  flow: Flow,
  nodeId: string,
  next: { remote: string; path: string; label?: string },
): Flow {
  const graph = toGraph(flow)
  const node = graph.nodes.find((n) => n.id === nodeId)
  if (!node) return flow
  node.remote = (next.remote ?? '').trim()
  node.path = normalizePath(next.path)
  node.label = next.label ?? node.label
  node.key = locationKey(node.remote, node.path)
  const { operations, canvas_json } = fromGraph(graph, flow.operations)
  return { ...flow, operations, canvas_json }
}

export function newNodeId(key: string): string {
  return stableNodeId(key)
}

function num(v: unknown, fallback: number): number {
  const n = typeof v === 'number' ? v : Number(v)
  return Number.isFinite(n) ? n : fallback
}

function stableNodeId(key: string): string {
  let h = 2166136261
  for (let i = 0; i < key.length; i++) {
    h ^= key.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return `n_${(h >>> 0).toString(16)}`
}
