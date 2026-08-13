import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import en from './locales/en.ts'
import vi from './locales/vi.ts'

function leafPaths(value: unknown, prefix = ''): string[] {
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return Object.entries(value as Record<string, unknown>).flatMap(([k, v]) =>
      leafPaths(v, prefix ? `${prefix}.${k}` : k),
    )
  }
  return prefix ? [prefix] : []
}

describe('i18n locale shape', () => {
  it('keeps English and Vietnamese on the same key set', () => {
    const enKeys = leafPaths(en).sort()
    const viKeys = leafPaths(vi).sort()
    assert.deepEqual(viKeys, enKeys)
  })
})
