import { describe, test, expect } from 'vitest'
import { KFTLRequestMap } from '@/classes/kftl/kftl-request-map'

describe('KFTLRequestMap', () => {
  test('can be instantiated', () => {
    const map = new KFTLRequestMap()
    expect(map).toBeInstanceOf(Map)
  })

  test('empty map has size 0', () => {
    const map = new KFTLRequestMap()
    expect(map.size).toBe(0)
  })

  test('extends Map', () => {
    const map = new KFTLRequestMap()
    expect(map).toBeInstanceOf(Map)
  })
})
