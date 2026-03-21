import { describe, test, expect } from 'vitest'
import type { LatLng } from '@/classes/datas/lat-lng'

describe('LatLng (interface)', () => {
  test('survives JSON round-trip', () => {
    const original: LatLng = { lat: 35.6812, lng: 139.7671 }
    const restored: LatLng = JSON.parse(JSON.stringify(original))
    expect(restored.lat).toBe(original.lat)
    expect(restored.lng).toBe(original.lng)
  })

  test('handles zero coordinates through JSON round-trip', () => {
    const original: LatLng = { lat: 0, lng: 0 }
    const restored: LatLng = JSON.parse(JSON.stringify(original))
    expect(restored.lat).toBe(0)
    expect(restored.lng).toBe(0)
  })

  test('handles negative coordinates through JSON round-trip', () => {
    const original: LatLng = { lat: -33.8688, lng: -151.2093 }
    const restored: LatLng = JSON.parse(JSON.stringify(original))
    expect(restored.lat).toBe(original.lat)
    expect(restored.lng).toBe(original.lng)
  })

  test('different LatLng objects are independent after round-trip', () => {
    const a: LatLng = { lat: 10, lng: 20 }
    const b: LatLng = JSON.parse(JSON.stringify(a))
    b.lat = 99
    expect(a.lat).toBe(10)
    expect(b.lat).toBe(99)
  })
})
