// MetaInfoBase is an abstract class that cannot be instantiated directly.
// These tests use the plain-object factory to verify the data shape
// that concrete subclasses (Tag, Text, Notification) rely on.

import { describe, test, expect } from 'vitest'
import { makeMetaInfoBase } from '../../helpers/factory'

describe('MetaInfoBase (factory-based)', () => {
  test('makeMetaInfoBase returns object with all required fields', () => {
    const meta = makeMetaInfoBase()
    expect(meta.id).toBe('test-meta-id')
    expect(meta.is_deleted).toBe(false)
    expect(meta.target_id).toBe('test-target-id')
    expect(meta.related_time).toBeDefined()
    expect(meta.create_time).toBeDefined()
    expect(meta.create_app).toBe('gkill')
    expect(meta.create_device).toBe('test-device')
    expect(meta.create_user).toBe('admin')
    expect(meta.update_time).toBeDefined()
    expect(meta.update_app).toBe('gkill')
    expect(meta.update_device).toBe('test-device')
    expect(meta.update_user).toBe('admin')
  })

  test('makeMetaInfoBase overrides work', () => {
    const meta = makeMetaInfoBase({ id: 'custom-meta-id', target_id: 'custom-target', is_deleted: true })
    expect(meta.id).toBe('custom-meta-id')
    expect(meta.target_id).toBe('custom-target')
    expect(meta.is_deleted).toBe(true)
    // non-overridden fields keep defaults
    expect(meta.create_app).toBe('gkill')
  })

  test('makeMetaInfoBase creates independent objects', () => {
    const a = makeMetaInfoBase()
    const b = makeMetaInfoBase()
    a.id = 'modified'
    expect(b.id).toBe('test-meta-id')
  })
})
