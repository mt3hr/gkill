// ReKyou imports Kyou directly, which has circular import chains that cause
// "Class extends value undefined" in jsdom. These tests use the plain-object
// factory to verify the data shape.

import { describe, test, expect } from 'vitest'
import { makeReKyou } from '../../helpers/factory'

describe('ReKyou (factory-based)', () => {
  test('makeReKyou returns object with all required fields', () => {
    const rekyou = makeReKyou()
    expect(rekyou.id).toBe('test-rekyou-id')
    expect(rekyou.is_deleted).toBe(false)
    expect(rekyou.rep_name).toBe('test-rep')
    expect(rekyou.data_type).toBe('re_kyou')
    expect(rekyou.related_time).toBeDefined()
    expect(rekyou.create_time).toBeDefined()
    expect(rekyou.create_app).toBe('gkill')
    expect(rekyou.create_user).toBe('admin')
    expect(rekyou.update_time).toBeDefined()
    expect(rekyou.update_app).toBe('gkill')
    expect(rekyou.update_user).toBe('admin')
  })

  test('makeReKyou includes target_id field', () => {
    const rekyou = makeReKyou()
    expect(rekyou.target_id).toBe('test-target-id')
  })

  test('makeReKyou includes attached_kyou initialized to null', () => {
    const rekyou = makeReKyou()
    expect(rekyou.attached_kyou).toBeNull()
  })

  test('makeReKyou includes attached_histories initialized to empty array', () => {
    const rekyou = makeReKyou()
    expect(rekyou.attached_histories).toEqual([])
  })

  test('makeReKyou overrides work', () => {
    const rekyou = makeReKyou({ id: 'custom-id', target_id: 'custom-target', is_deleted: true })
    expect(rekyou.id).toBe('custom-id')
    expect(rekyou.target_id).toBe('custom-target')
    expect(rekyou.is_deleted).toBe(true)
    // non-overridden fields keep defaults
    expect(rekyou.rep_name).toBe('test-rep')
  })

  test('makeReKyou creates independent objects', () => {
    const a = makeReKyou()
    const b = makeReKyou()
    a.id = 'modified'
    expect(b.id).toBe('test-rekyou-id')
  })
})
