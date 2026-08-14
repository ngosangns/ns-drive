import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import type { FileTransferInfo } from '../api/types.ts'
import {
  PARTICLE_CAP,
  ERROR_T,
  TARGET_T,
  desiredT,
  isSuccessStatus,
  statusColorToken,
  transfersToParticles,
} from './fileParticles.ts'

function tr(name: string, status: string, progress = 0): FileTransferInfo {
  return { name, size: 1, bytes: 0, progress, status }
}

describe('fileParticles', () => {
  it('keeps pending files in the card instead of static edge dots', () => {
    const transfers = [tr('a', 'pending'), tr('b', 'transferring'), tr('c', 'failed'), tr('d', 'pending')]
    const dots = transfersToParticles(transfers, 'push')
    assert.equal(dots.length, 2)
    assert.deepEqual(
      dots.map((d) => d.status),
      ['transferring', 'failed'],
    )
  })

  it('prioritizes active transfers when capping dots', () => {
    const transfers: FileTransferInfo[] = []
    for (let i = 0; i < 20; i++) transfers.push(tr(`p-${i}`, 'pending'))
    for (let i = 0; i < 30; i++) transfers.push(tr(`x-${i}`, 'transferring'))
    const dots = transfersToParticles(transfers, 'push')
    assert.equal(dots.length, PARTICLE_CAP)
    assert.ok(dots.every((d) => d.status === 'transferring'))
    assert.ok(dots.every((d) => d.dir === 1))
  })

  it('maps failed status to the danger token', () => {
    assert.equal(statusColorToken('failed'), '--color-danger')
    assert.equal(statusColorToken('completed'), '--color-success')
    assert.equal(statusColorToken('transferring'), '--color-accent-strong')
  })

  it('alternates direction on bidirectional actions', () => {
    const transfers = [tr('a', 'transferring'), tr('b', 'transferring'), tr('c', 'failed')]
    const dots = transfersToParticles(transfers, 'bi')
    assert.deepEqual(
      dots.map((d) => d.dir),
      [1, -1, 1],
    )
  })

  it('handles empty input', () => {
    assert.deepEqual(transfersToParticles(undefined), [])
    assert.deepEqual(transfersToParticles([]), [])
  })

  it('maps active file progress across the full edge', () => {
    assert.equal(desiredT({ status: 'checking', progress: 0, dir: 1, slot: 0, queued: 1 }), 0)
    assert.equal(desiredT({ status: 'checking', progress: 50, dir: 1, slot: 0, queued: 1 }), 0.5)
    assert.equal(desiredT({ status: 'transferring', progress: 100, dir: 1, slot: 0, queued: 1 }), 1)
    assert.equal(desiredT({ status: 'failed', progress: 20, dir: 1, slot: 0, queued: 1 }), ERROR_T)
    for (const status of ['pending']) {
      const t = desiredT({ status, progress: 0, dir: 1, slot: 0, queued: 1 })
      assert.ok(t > 0 && t < 1, `${status} t=${t}`)
    }
    const mid = desiredT({ status: 'transferring', progress: 50, dir: 1, slot: 0, queued: 1 })
    const done = desiredT({ status: 'transferring', progress: 100, dir: 1, slot: 0, queued: 1 })
    assert.ok(mid < done)
    assert.equal(mid, 0.5)
    assert.equal(done, 1)
  })

  it('only success is allowed to seek the target node', () => {
    assert.equal(desiredT({ status: 'completed', progress: 100, dir: 1, slot: 0, queued: 0 }), TARGET_T)
    assert.equal(desiredT({ status: 'checked', progress: 100, dir: 1, slot: 0, queued: 0 }), TARGET_T)
    assert.ok(isSuccessStatus('completed'))
    assert.equal(isSuccessStatus('failed'), false)
    assert.equal(isSuccessStatus('transferring'), false)
    const fail = transfersToParticles([tr('x', 'failed')], 'push')[0]
    assert.equal(fail.t, ERROR_T)
    const ok = transfersToParticles([tr('y', 'completed', 100)], 'push')[0]
    assert.equal(ok.t, TARGET_T)
  })

  it('packs queued files into distinct slots', () => {
    const dots = transfersToParticles(
      [tr('a', 'checking'), tr('b', 'failed'), tr('c', 'checking')],
      'push',
    )
    const ts = dots.map((d) => d.t)
    assert.equal(new Set(ts).size, 2)
    assert.ok(ts.every((t) => t >= 0 && t <= 1))
  })
})
