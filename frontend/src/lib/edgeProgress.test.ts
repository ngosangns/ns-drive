import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { formatFileProgressBadge, formatTransferProgress } from './edgeProgress.ts'

describe('formatFileProgressBadge', () => {
  it('uses backend completion counters instead of the stable transfer row count', () => {
    assert.equal(formatFileProgressBadge(2, 5, 5), '2/5')
  })

  it('falls back to known transfer rows before a total is available', () => {
    assert.equal(formatFileProgressBadge(0, 0, 3), '3')
  })

  it('renders a bounded percentage for an active file', () => {
    assert.equal(formatTransferProgress(42.4), '42%')
    assert.equal(formatTransferProgress(101), '100%')
  })
})
