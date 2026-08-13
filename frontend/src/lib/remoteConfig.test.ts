import assert from 'node:assert/strict'
import { describe, it } from 'node:test'
import { configFieldsFor, missingRequiredFields, toConfigKVs } from './remoteConfig.ts'

describe('remoteConfig', () => {
  it('requires user and pass for mega', () => {
    const keys = configFieldsFor('mega').map((f) => f.key)
    assert.deepEqual(keys, ['user', 'pass'])
    assert.deepEqual(missingRequiredFields('mega', {}), ['user', 'pass'])
    assert.deepEqual(missingRequiredFields('mega', { user: 'a@b.c', pass: 'x' }), [])
  })

  it('needs no extra fields for local', () => {
    assert.deepEqual(configFieldsFor('local'), [])
    assert.deepEqual(missingRequiredFields('local', {}), [])
  })

  it('builds rclone key=value pairs and skips blanks', () => {
    assert.deepEqual(toConfigKVs({ user: 'a@b.c', pass: 's3cret', extra: '  ' }), [
      'user=a@b.c',
      'pass=s3cret',
    ])
  })
})
