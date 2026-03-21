// Kyou class has circular import chains that cause "Class extends value undefined"
// in jsdom. These tests use the plain-object factory to verify the data shape
// that the rest of the codebase relies on.

import { describe, test, expect } from 'vitest'
import { makeKyou } from '../../helpers/factory'

describe('Kyou (factory-based)', () => {
  test('makeKyou returns object with all required fields', () => {
    const kyou = makeKyou()
    expect(kyou.id).toBe('test-kyou-id')
    expect(kyou.is_deleted).toBe(false)
    expect(kyou.rep_name).toBe('test-rep')
    expect(kyou.data_type).toBe('kmemo')
    expect(kyou.related_time).toBeDefined()
    expect(kyou.create_time).toBeDefined()
    expect(kyou.create_app).toBe('gkill')
    expect(kyou.create_user).toBe('admin')
    expect(kyou.update_time).toBeDefined()
    expect(kyou.update_app).toBe('gkill')
    expect(kyou.update_user).toBe('admin')
  })

  test('makeKyou includes typed_* fields initialized to null', () => {
    const kyou = makeKyou()
    expect(kyou.typed_kmemo).toBeNull()
    expect(kyou.typed_urlog).toBeNull()
    expect(kyou.typed_nlog).toBeNull()
    expect(kyou.typed_timeis).toBeNull()
    expect(kyou.typed_mi).toBeNull()
    expect(kyou.typed_lantana).toBeNull()
    expect(kyou.typed_kc).toBeNull()
    expect(kyou.typed_idf_kyou).toBeNull()
    expect(kyou.typed_git_commit_log).toBeNull()
  })

  test('makeKyou includes image_source field', () => {
    const kyou = makeKyou()
    expect(kyou.image_source).toBe('')
  })

  test('makeKyou overrides work', () => {
    const kyou = makeKyou({ id: 'custom-id', data_type: 'timeis', is_deleted: true })
    expect(kyou.id).toBe('custom-id')
    expect(kyou.data_type).toBe('timeis')
    expect(kyou.is_deleted).toBe(true)
    // non-overridden fields keep defaults
    expect(kyou.rep_name).toBe('test-rep')
  })

  test('makeKyou creates independent objects', () => {
    const a = makeKyou()
    const b = makeKyou()
    a.id = 'modified'
    expect(b.id).toBe('test-kyou-id')
  })
})
