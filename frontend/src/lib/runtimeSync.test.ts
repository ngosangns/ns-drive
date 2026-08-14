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
      files_transferred: 2,
      total_files: 5,
      transfers: [{ name: 'a.txt', status: 'transferring', progress: 20, size: 10, bytes: 2 }],
    })
    assert.ok(snap)
    assert.equal(snap!.flow_id, 'flow1')
    assert.equal(snap!.op_id, 'op1')
    assert.equal(snap!.status, 'running')
    assert.equal(Math.round(snap!.progress), 20)
    assert.equal(snap!.files_transferred, 2)
    assert.equal(snap!.total_files, 5)
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

  it('keeps live transfer details when a lagging snapshot has no sync payload', () => {
    const out = applyRuntimeSnapshot(
      { revision: 8, flows: [{ id: 'flow1', status: 'running', ops: [{ id: 'op1', status: 'running' }] }] },
      [],
      {
        flow1: {
          flow_id: 'flow1', op_id: 'op1', action: 'push', status: 'running', progress: 40,
          speed_bps: 0, eta_secs: 0, files_transferred: 0, total_files: 1,
          bytes_transferred: 40, total_bytes: 100, current_file: 'uploading.bin', errors: 0,
          checks: 0, total_checks: 0, deletes: 0, renames: 0, updated_at: 1,
          transfers: [{ name: 'uploading.bin', status: 'transferring', progress: 40, size: 100, bytes: 40 }],
        },
      },
    )
    assert.equal(out.opSyncStatus.flow1.transfers?.[0].name, 'uploading.bin')
  })
})
