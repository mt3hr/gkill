// MiReKyou imports Kyou directly, which has circular import chains that cause
// "Class extends value undefined" in jsdom. These tests use the plain-object
// factory to verify the data shape.

import { describe, test, expect } from 'vitest'
import { makeMiReKyou } from '../../helpers/factory'

describe('MiReKyou (factory-based)', () => {
  test('makeMiReKyou returns object with all required fields', () => {
    const mirekyou = makeMiReKyou()
    expect(mirekyou.id).toBe('test-mirekyou-id')
    expect(mirekyou.is_deleted).toBe(false)
    expect(mirekyou.rep_name).toBe('test-rep')
    expect(mirekyou.data_type).toBe('mirekyou_create')
    expect(mirekyou.related_time).toBeDefined()
    expect(mirekyou.create_time).toBeDefined()
    expect(mirekyou.create_app).toBe('gkill')
    expect(mirekyou.create_user).toBe('admin')
    expect(mirekyou.update_time).toBeDefined()
    expect(mirekyou.update_app).toBe('gkill')
    expect(mirekyou.update_user).toBe('admin')
  })

  test('makeMiReKyou includes target_id field (ReKyou由来)', () => {
    const mirekyou = makeMiReKyou()
    expect(mirekyou.target_id).toBe('test-target-id')
  })

  test('makeMiReKyou includes Mi schedule fields (Mi由来)', () => {
    const mirekyou = makeMiReKyou()
    expect(mirekyou.is_checked).toBe(false)
    expect(mirekyou.board_name).toBe('default')
    expect(mirekyou.limit_time).toBeNull()
    expect(mirekyou.estimate_start_time).toBeNull()
    expect(mirekyou.estimate_end_time).toBeNull()
  })

  test('makeMiReKyou does not have a title (タイトルは持たない)', () => {
    const mirekyou = makeMiReKyou()
    expect('title' in mirekyou).toBe(false)
  })

  test('makeMiReKyou includes attached_kyou initialized to null', () => {
    const mirekyou = makeMiReKyou()
    expect(mirekyou.attached_kyou).toBeNull()
  })

  test('makeMiReKyou includes attached_histories initialized to empty array', () => {
    const mirekyou = makeMiReKyou()
    expect(mirekyou.attached_histories).toEqual([])
  })

  test('makeMiReKyou overrides work', () => {
    const mirekyou = makeMiReKyou({ id: 'custom-id', target_id: 'custom-target', board_name: 'work', is_checked: true })
    expect(mirekyou.id).toBe('custom-id')
    expect(mirekyou.target_id).toBe('custom-target')
    expect(mirekyou.board_name).toBe('work')
    expect(mirekyou.is_checked).toBe(true)
    // non-overridden fields keep defaults
    expect(mirekyou.rep_name).toBe('test-rep')
  })

  test('makeMiReKyou creates independent objects', () => {
    const a = makeMiReKyou()
    const b = makeMiReKyou()
    a.id = 'modified'
    expect(b.id).toBe('test-mirekyou-id')
  })

  test('data_type は mi で始まるが mirekyou 判定を先に行う必要がある', () => {
    const mirekyou = makeMiReKyou()
    // Kyou.load_typed_datas がこの前提で分岐している
    expect(mirekyou.data_type.startsWith('mi')).toBe(true)
    expect(mirekyou.data_type.startsWith('mirekyou')).toBe(true)
  })
})
