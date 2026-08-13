import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import {
  acceptLive,
  parseFlowPayload,
  shouldReloadFlows,
  shouldReloadRemotes,
} from './sseRuntime.ts'

describe('sseRuntime', () => {
  it('parses flow and legacy board payloads', () => {
    const a = parseFlowPayload({ flow_id: 'f1', status: 'running', op_id: 'op1' })
    assert.deepEqual(a, { id: 'f1', status: 'running', opId: 'op1', error: undefined })
    const b = parseFlowPayload({ board_id: 'f1', status: 'failed', action: 'boom' })
    assert.equal(b?.error, 'boom')
    assert.equal(parseFlowPayload({ status: 'running' }), null)
  })

  it('skips flow reload while the canvas is dirty', () => {
    assert.equal(shouldReloadFlows('flows', true), false)
    assert.equal(shouldReloadFlows('flows', false), true)
    assert.equal(shouldReloadFlows('remotes', false), false)
    assert.equal(shouldReloadRemotes('remotes'), true)
  })

  it('drops live events until a snapshot is seen', () => {
    assert.equal(acceptLive(false, 0, { revision: 1 }), false)
    assert.equal(acceptLive(true, 3, { revision: 2 }), false)
    assert.equal(acceptLive(true, 3, {}), true)
  })
})
