import { GPSLog } from '@/classes/datas/gps-log'

describe('GPSLog', () => {
  test('can be instantiated', () => {
    const gps = new GPSLog()
    expect(gps).toBeInstanceOf(GPSLog)
  })

  describe('default field values', () => {
    let gps: GPSLog

    beforeEach(() => {
      gps = new GPSLog()
    })

    test('related_time defaults to epoch', () => {
      expect(gps.related_time).toBeInstanceOf(Date)
      expect(gps.related_time.getTime()).toBe(0)
    })

    test('latitude defaults to 0', () => {
      expect(gps.latitude).toBe(0)
    })

    test('longitude defaults to 0', () => {
      expect(gps.longitude).toBe(0)
    })
  })

  describe('field assignment', () => {
    test('can set latitude and longitude', () => {
      const gps = new GPSLog()
      gps.latitude = 35.6812
      gps.longitude = 139.7671
      expect(gps.latitude).toBe(35.6812)
      expect(gps.longitude).toBe(139.7671)
    })

    test('can set related_time', () => {
      const gps = new GPSLog()
      const date = new Date('2025-01-15T10:30:00Z')
      gps.related_time = date
      expect(gps.related_time).toBe(date)
    })
  })
})
