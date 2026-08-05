import { deep_equals } from '@/classes/deep-equals'

describe('deepEquals', () => {
  describe('primitives', () => {
    test('equal numbers return true', () => {
      expect(deep_equals(1, 1)).toBe(true)
      expect(deep_equals(0, 0)).toBe(true)
      expect(deep_equals(-5, -5)).toBe(true)
    })

    test('different numbers return false', () => {
      expect(deep_equals(1, 2)).toBe(false)
      expect(deep_equals(0, 1)).toBe(false)
    })

    test('equal strings return true', () => {
      expect(deep_equals('hello', 'hello')).toBe(true)
      expect(deep_equals('', '')).toBe(true)
    })

    test('different strings return false', () => {
      expect(deep_equals('hello', 'world')).toBe(false)
    })

    test('equal booleans return true', () => {
      expect(deep_equals(true, true)).toBe(true)
      expect(deep_equals(false, false)).toBe(true)
    })

    test('different booleans return false', () => {
      expect(deep_equals(true, false)).toBe(false)
    })
  })

  describe('null and undefined', () => {
    test('null equals null', () => {
      expect(deep_equals(null, null)).toBe(true)
    })

    test('undefined equals undefined', () => {
      expect(deep_equals(undefined, undefined)).toBe(true)
    })

    test('null does not equal undefined', () => {
      expect(deep_equals(null, undefined)).toBe(false)
    })

    test('NaN equals NaN', () => {
      expect(deep_equals(NaN, NaN)).toBe(true)
    })
  })

  describe('arrays', () => {
    test('equal arrays return true', () => {
      expect(deep_equals([1, 2, 3], [1, 2, 3])).toBe(true)
      expect(deep_equals([], [])).toBe(true)
    })

    test('different arrays return false', () => {
      expect(deep_equals([1, 2, 3], [1, 2, 4])).toBe(false)
    })

    test('arrays of different length return false', () => {
      expect(deep_equals([1, 2], [1, 2, 3])).toBe(false)
    })

    test('array does not equal non-array object', () => {
      expect(deep_equals([1, 2] as never, { 0: 1, 1: 2 } as never)).toBe(false)
    })
  })

  describe('nested objects', () => {
    test('equal nested objects return true', () => {
      const a = { x: 1, y: { z: 'hello' } }
      const b = { x: 1, y: { z: 'hello' } }
      expect(deep_equals(a, b)).toBe(true)
    })

    test('different nested objects return false', () => {
      const a = { x: 1, y: { z: 'hello' } }
      const b = { x: 1, y: { z: 'world' } }
      expect(deep_equals(a, b)).toBe(false)
    })

    test('objects with different keys return false', () => {
      const a = { x: 1 } as never
      const b = { x: 1, y: 2 } as never
      expect(deep_equals(a, b)).toBe(false)
    })

    test('objects with same keys but missing property return false', () => {
      const a = { x: 1, y: 2 } as never
      const b = { x: 1, z: 2 } as never
      expect(deep_equals(a, b)).toBe(false)
    })
  })

  describe('Date comparison', () => {
    test('equal dates return true', () => {
      const d1 = new Date('2024-01-01T00:00:00Z')
      const d2 = new Date('2024-01-01T00:00:00Z')
      expect(deep_equals(d1, d2)).toBe(true)
    })

    test('different dates return false', () => {
      const d1 = new Date('2024-01-01T00:00:00Z')
      const d2 = new Date('2024-06-15T00:00:00Z')
      expect(deep_equals(d1, d2)).toBe(false)
    })

    test('date does not equal non-date object', () => {
      const d = new Date('2024-01-01T00:00:00Z')
      const obj = { getTime: () => d.getTime() }
      expect(deep_equals(d as never, obj as never)).toBe(false)
    })
  })

  describe('RegExp comparison', () => {
    test('equal regexps return true', () => {
      expect(deep_equals(/abc/g, /abc/g)).toBe(true)
    })

    test('different regexps return false', () => {
      expect(deep_equals(/abc/g, /abc/i)).toBe(false)
      expect(deep_equals(/abc/, /def/)).toBe(false)
    })
  })
})
