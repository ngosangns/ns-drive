import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type { Flow } from '../api/types.ts'
import {
  applyRuntimeSnapshot,
  progressEventToOpStatus,
  shouldApplyLiveEvent,
} from './runtimeSync.ts'

describe('runtimeSync', () => {
  it('rejects live events until a snapshot has been seen', () => {
    assert.equal(shouldApplyLiveEvent(false, 0, 1), false)
    assert.equal(shouldApplyLiveEvent(true, 3, 0), true)
    assert.equal(shouldApplyLiveEvent(true, 3, 3), false)
    assert.equal(shouldApplyLiveEvent(true, 3, 4), true)
  })

  it('maps a progress payload onto FlowOpSyncStatus', () => {
    const snap = progressEventToOpStatus('sync:progress', {
      profile_id: 'flow1:op1',
      action: 'push',
      state: 'running',
      transferred: 20,
      total: 100,
      transfers: [{ name: 'a.txt', status: 'transferring', progress: 20, size: 10, bytes: 2 }],
    })
    assert.ok(snap)
    assert.equal(snap!.flow_id, 'flow1')
    assert.equal(snap!.op_id, 'op1')
    assert.equal(snap!.status, 'running')
    assert.equal(Math.round(snap!.progress), 20)
    assert.equal(snap!.transfers?.[0].name, 'a.txt')
  })

  it('hydrates Pinia-shaped runtime from a backend snapshot', () => {
    const items: Flow[] = [
      {
        id: 'flow1',
        name: 'one',
        operations: [
          {
            id: 'op1',
            source_remote: '',
            source_path: '/',
            target_remote: '',
            target_path: '/out',
            action: 'push',
            status: 'idle',
          },
        ],
      },
    ]
    const out = applyRuntimeSnapshot(
      {
        revision: 7,
        flows: [
          {
            id: 'flow1',
            status: 'running',
            last_error: '',
            ops: [{ id: 'op1', status: 'running' }],
            sync: {
              profile_id: 'flow1:op1',
              action: 'push',
              state: 'running',
              transferred: 1,
              total: 2,
              transfers: [{ name: 'b.bin', status: 'transferring', progress: 50 }],
            },
            log: [{ at: 1, status: 'running', label: 'Flow' }],
          },
        ],
      },
      items,
    )
    assert.equal(out.revision, 7)
    assert.equal(out.runStatus.flow1, 'running')
    assert.equal(out.items[0].operations?.[0].status, 'running')
    assert.equal(out.opSyncStatus.flow1.transfers?.[0].name, 'b.bin')
    assert.equal(out.runLog.flow1[0].label, 'Flow')
  })
})
