// InfoBase is an abstract class that cannot be instantiated directly.
// These tests use the plain-object factory to verify the data shape
// that concrete subclasses rely on.

import { describe, test, expect } from 'vitest'
import { makeInfoBase } from '../../helpers/factory'

describe('InfoBase (factory-based)', () => {
  test('makeInfoBase returns object with all required fields', () => {
    const info = makeInfoBase()
    expect(info.id).toBe('test-info-id')
    expect(info.is_deleted).toBe(false)
    expect(info.rep_name).toBe('test-rep')
    expect(info.data_type).toBe('kmemo')
    expect(info.related_time).toBeDefined()
    expect(info.create_time).toBeDefined()
    expect(info.create_app).toBe('gkill')
    expect(info.create_device).toBe('test-device')
    expect(info.create_user).toBe('admin')
    expect(info.update_time).toBeDefined()
    expect(info.update_app).toBe('gkill')
    expect(info.update_device).toBe('test-device')
    expect(info.update_user).toBe('admin')
  })

  test('makeInfoBase includes attached arrays initialized to empty', () => {
    const info = makeInfoBase()
    expect(info.attached_tags).toEqual([])
    expect(info.attached_texts).toEqual([])
    expect(info.attached_notifications).toEqual([])
    expect(info.attached_timeis_kyou).toEqual([])
  })

  test('makeInfoBase includes is_checked_kyou initialized to false', () => {
    const info = makeInfoBase()
    expect(info.is_checked_kyou).toBe(false)
  })

  test('makeInfoBase overrides work', () => {
    const info = makeInfoBase({ id: 'custom-id', data_type: 'timeis', is_deleted: true })
    expect(info.id).toBe('custom-id')
    expect(info.data_type).toBe('timeis')
    expect(info.is_deleted).toBe(true)
    // non-overridden fields keep defaults
    expect(info.rep_name).toBe('test-rep')
  })

  test('makeInfoBase creates independent objects', () => {
    const a = makeInfoBase()
    const b = makeInfoBase()
    a.id = 'modified'
    expect(b.id).toBe('test-info-id')
  })
})
