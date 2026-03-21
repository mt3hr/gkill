import { deepEquals } from '@/classes/deep-equals'

describe('deepEquals', () => {
  describe('primitives', () => {
    test('equal numbers return true', () => {
      expect(deepEquals(1, 1)).toBe(true)
      expect(deepEquals(0, 0)).toBe(true)
      expect(deepEquals(-5, -5)).toBe(true)
    })

    test('different numbers return false', () => {
      expect(deepEquals(1, 2)).toBe(false)
      expect(deepEquals(0, 1)).toBe(false)
    })

    test('equal strings return true', () => {
      expect(deepEquals('hello', 'hello')).toBe(true)
      expect(deepEquals('', '')).toBe(true)
    })

    test('different strings return false', () => {
      expect(deepEquals('hello', 'world')).toBe(false)
    })

    test('equal booleans return true', () => {
      expect(deepEquals(true, true)).toBe(true)
      expect(deepEquals(false, false)).toBe(true)
    })

    test('different booleans return false', () => {
      expect(deepEquals(true, false)).toBe(false)
    })
  })

  describe('null and undefined', () => {
    test('null equals null', () => {
      expect(deepEquals(null, null)).toBe(true)
    })

    test('undefined equals undefined', () => {
      expect(deepEquals(undefined, undefined)).toBe(true)
    })

    test('null does not equal undefined', () => {
      expect(deepEquals(null, undefined)).toBe(false)
    })

    test('NaN equals NaN', () => {
      expect(deepEquals(NaN, NaN)).toBe(true)
    })
  })

  describe('arrays', () => {
    test('equal arrays return true', () => {
      expect(deepEquals([1, 2, 3], [1, 2, 3])).toBe(true)
      expect(deepEquals([], [])).toBe(true)
    })

    test('different arrays return false', () => {
      expect(deepEquals([1, 2, 3], [1, 2, 4])).toBe(false)
    })

    test('arrays of different length return false', () => {
      expect(deepEquals([1, 2], [1, 2, 3])).toBe(false)
    })

    test('array does not equal non-array object', () => {
      expect(deepEquals([1, 2] as any, { 0: 1, 1: 2 } as any)).toBe(false)
    })
  })

  describe('nested objects', () => {
    test('equal nested objects return true', () => {
      const a = { x: 1, y: { z: 'hello' } }
      const b = { x: 1, y: { z: 'hello' } }
      expect(deepEquals(a, b)).toBe(true)
    })

    test('different nested objects return false', () => {
      const a = { x: 1, y: { z: 'hello' } }
      const b = { x: 1, y: { z: 'world' } }
      expect(deepEquals(a, b)).toBe(false)
    })

    test('objects with different keys return false', () => {
      const a = { x: 1 } as any
      const b = { x: 1, y: 2 } as any
      expect(deepEquals(a, b)).toBe(false)
    })

    test('objects with same keys but missing property return false', () => {
      const a = { x: 1, y: 2 } as any
      const b = { x: 1, z: 2 } as any
      expect(deepEquals(a, b)).toBe(false)
    })
  })

  describe('Date comparison', () => {
    test('equal dates return true', () => {
      const d1 = new Date('2024-01-01T00:00:00Z')
      const d2 = new Date('2024-01-01T00:00:00Z')
      expect(deepEquals(d1, d2)).toBe(true)
    })

    test('different dates return false', () => {
      const d1 = new Date('2024-01-01T00:00:00Z')
      const d2 = new Date('2024-06-15T00:00:00Z')
      expect(deepEquals(d1, d2)).toBe(false)
    })

    test('date does not equal non-date object', () => {
      const d = new Date('2024-01-01T00:00:00Z')
      const obj = { getTime: () => d.getTime() }
      expect(deepEquals(d as any, obj as any)).toBe(false)
    })
  })

  describe('RegExp comparison', () => {
    test('equal regexps return true', () => {
      expect(deepEquals(/abc/g, /abc/g)).toBe(true)
    })

    test('different regexps return false', () => {
      expect(deepEquals(/abc/g, /abc/i)).toBe(false)
      expect(deepEquals(/abc/, /def/)).toBe(false)
    })
  })
})
