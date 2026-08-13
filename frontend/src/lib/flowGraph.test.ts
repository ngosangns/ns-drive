import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type { Flow, Operation } from '../api/types.ts'
import {
  connectError,
  DEFAULT_COL_GAP,
  fromGraph,
  locationKey,
  relocateNode,
  toGraph,
} from './flowGraph.ts'

function op(partial: Partial<Operation> & Pick<Operation, 'id'>): Operation {
  return {
    source_remote: '',
    source_path: '/',
    target_remote: '',
    target_path: '/',
    action: 'push',
    ...partial,
  }
}

describe('flowGraph', () => {
  it('maps one operation to two nodes and one edge', () => {
    const flow: Flow = {
      id: 'f1',
      name: 'one',
      operations: [op({ id: 'op1', source_remote: 'a', source_path: '/src', target_path: '/dst' })],
    }
    const g = toGraph(flow)
    assert.equal(g.nodes.length, 2)
    assert.equal(g.edges.length, 1)
    assert.equal(g.edges[0].id, 'op1')
    assert.equal(g.edges[0].source !== g.edges[0].target, true)
    const src = g.nodes.find((n) => n.id === g.edges[0].source)
    const dst = g.nodes.find((n) => n.id === g.edges[0].target)
    assert.equal(src?.remote, 'a')
    assert.equal(src?.path, '/src')
    assert.equal(dst?.path, '/dst')
    assert.equal(src?.x, 0)
    assert.equal(dst?.x, DEFAULT_COL_GAP)
  })

  it('merges two operations that share a source into three nodes', () => {
    const flow: Flow = {
      id: 'f1',
      name: 'fanout',
      operations: [
        op({ id: 'op1', source_remote: 'drv', source_path: '/in', target_path: '/out-a' }),
        op({ id: 'op2', source_remote: 'drv', source_path: '/in', target_path: '/out-b' }),
      ],
    }
    const g = toGraph(flow)
    assert.equal(g.nodes.length, 3)
    assert.equal(g.edges.length, 2)
    assert.equal(g.edges[0].source, g.edges[1].source)
    assert.notEqual(g.edges[0].target, g.edges[1].target)
  })

  it('round-trips graph back to operations and canvas_json', () => {
    const flow: Flow = {
      id: 'f1',
      name: 'rt',
      operations: [
        op({
          id: 'op1',
          source_remote: 'drv',
          source_path: '/in',
          target_remote: '',
          target_path: '/tmp/out',
          sync_config: { action: 'push', dryRun: true },
        }),
      ],
    }
    const g = toGraph(flow)
    const { operations, canvas_json } = fromGraph(g, flow.operations)
    assert.equal(operations.length, 1)
    assert.equal(operations[0].id, 'op1')
    assert.equal(operations[0].source_remote, 'drv')
    assert.equal(operations[0].source_path, '/in')
    assert.equal(operations[0].target_path, '/tmp/out')
    assert.equal((operations[0].sync_config as { dryRun?: boolean }).dryRun, true)
    assert.equal(canvas_json.nodes.length, 2)
    const again = toGraph({ ...flow, operations, canvas_json })
    assert.equal(again.nodes.length, 2)
    assert.equal(again.edges[0].source, g.edges[0].source)
  })

  it('relocating a shared source node updates every operation that used it', () => {
    const flow: Flow = {
      id: 'f1',
      name: 'share',
      operations: [
        op({ id: 'op1', source_remote: 'drv', source_path: '/old', target_path: '/a' }),
        op({ id: 'op2', source_remote: 'drv', source_path: '/old', target_path: '/b' }),
      ],
    }
    const g = toGraph(flow)
    const srcId = g.edges[0].source
    const next = relocateNode(flow, srcId, { remote: 'drv', path: '/new' })
    assert.equal(next.operations?.[0].source_path, '/new')
    assert.equal(next.operations?.[1].source_path, '/new')
    assert.equal(next.operations?.[0].target_path, '/a')
  })

  it('rejects self-loops and duplicate action on the same pair', () => {
    const flow: Flow = {
      id: 'f1',
      name: 'dup',
      operations: [op({ id: 'op1', source_path: '/a', target_path: '/b' })],
    }
    const g = toGraph(flow)
    assert.equal(connectError(g.edges[0].source, g.edges[0].source, 'push', g), 'self-loop')
    assert.equal(connectError(g.edges[0].source, g.edges[0].target, 'push', g), 'duplicate-edge')
    assert.equal(connectError(g.edges[0].source, g.edges[0].target, 'bi', g), null)
  })

  it('uses a stable location key', () => {
    assert.equal(locationKey('drv', '/x'), locationKey('drv', '/x'))
    assert.notEqual(locationKey('drv', '/x'), locationKey('drv', '/y'))
    assert.equal(locationKey('', ''), locationKey('', '/'))
  })
})
