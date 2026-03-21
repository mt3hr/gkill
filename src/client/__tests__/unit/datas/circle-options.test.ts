import { describe, test, expect } from 'vitest'
import type { CircleOptions } from '@/classes/datas/circle-options'
import type { LatLng } from '@/classes/datas/lat-lng'

describe('CircleOptions (interface)', () => {
  test('survives JSON round-trip', () => {
    const center: LatLng = { lat: 35.6812, lng: 139.7671 }
    const original: CircleOptions = {
      visible: true,
      center,
      radius: 500,
      strokeColor: '#FF0000',
      strokeOpacity: 0.8,
      strokeWeight: 2,
    }

    const restored: CircleOptions = JSON.parse(JSON.stringify(original))

    expect(restored.visible).toBe(original.visible)
    expect(restored.center.lat).toBe(original.center.lat)
    expect(restored.center.lng).toBe(original.center.lng)
    expect(restored.radius).toBe(original.radius)
    expect(restored.strokeColor).toBe(original.strokeColor)
    expect(restored.strokeOpacity).toBe(original.strokeOpacity)
    expect(restored.strokeWeight).toBe(original.strokeWeight)
  })

  test('handles zero/falsy values through JSON round-trip', () => {
    const original: CircleOptions = {
      visible: false,
      center: { lat: 0, lng: 0 },
      radius: 0,
      strokeColor: '',
      strokeOpacity: 0,
      strokeWeight: 0,
    }

    const restored: CircleOptions = JSON.parse(JSON.stringify(original))

    expect(restored.visible).toBe(false)
    expect(restored.center.lat).toBe(0)
    expect(restored.center.lng).toBe(0)
    expect(restored.radius).toBe(0)
    expect(restored.strokeColor).toBe('')
    expect(restored.strokeOpacity).toBe(0)
    expect(restored.strokeWeight).toBe(0)
  })
})
