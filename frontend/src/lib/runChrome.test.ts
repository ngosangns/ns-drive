import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { fieldsLocked, isActiveRun, runBarVisible } from './runChrome.ts'

describe('runChrome', () => {
  it('treats running and cancelling as an active run', () => {
    assert.equal(isActiveRun('running'), true)
    assert.equal(isActiveRun('cancelling'), true)
    assert.equal(isActiveRun('completed'), false)
    assert.equal(isActiveRun('idle'), false)
    assert.equal(isActiveRun(undefined), false)
  })

  it('treats only active runs as canvas run state', () => {
    assert.equal(runBarVisible('running'), true)
    assert.equal(runBarVisible('cancelling'), true)
    assert.equal(runBarVisible('failed'), false)
    assert.equal(runBarVisible('completed'), false)
  })

  it('locks edit fields only during an active run', () => {
    assert.equal(fieldsLocked('running'), true)
    assert.equal(fieldsLocked('idle'), false)
  })
})
