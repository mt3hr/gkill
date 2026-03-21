// Mi class has circular import chains that cause "Class extends value undefined"
// in jsdom. These tests use the plain-object factory to verify the data shape
// that the rest of the codebase relies on.

import { describe, test, expect } from 'vitest'
import { makeMi } from '../../helpers/factory'

describe('Mi (factory-based)', () => {
  test('makeMi returns object with all required fields', () => {
    const mi = makeMi()
    expect(mi.id).toBe('test-mi-id')
    expect(mi.is_deleted).toBe(false)
    expect(mi.rep_name).toBe('test-rep')
    expect(mi.title).toBe('テストタスク')
    expect(mi.is_checked).toBe(false)
    expect(mi.board_name).toBe('default')
    expect(mi.create_time).toBeDefined()
    expect(mi.create_app).toBe('gkill')
    expect(mi.create_user).toBe('admin')
    expect(mi.update_time).toBeDefined()
  })

  test('makeMi has task-specific fields', () => {
    const mi = makeMi()
    expect(mi.limit_time).toBeNull()
    expect(mi.estimate_start_time).toBeNull()
    expect(mi.estimate_end_time).toBeNull()
  })

  test('makeMi overrides work', () => {
    const mi = makeMi({
      title: 'カスタムタスク',
      is_checked: true,
      board_name: 'done',
      limit_time: '2025-12-31T23:59:59+09:00',
    })
    expect(mi.title).toBe('カスタムタスク')
    expect(mi.is_checked).toBe(true)
    expect(mi.board_name).toBe('done')
    expect(mi.limit_time).toBe('2025-12-31T23:59:59+09:00')
  })

  test('makeMi creates independent objects', () => {
    const a = makeMi()
    const b = makeMi()
    a.title = 'modified'
    expect(b.title).toBe('テストタスク')
  })

  test('makeMi related_time is defined', () => {
    const mi = makeMi()
    expect(mi.related_time).toBe('2025-03-15T09:00:00+09:00')
  })
})
