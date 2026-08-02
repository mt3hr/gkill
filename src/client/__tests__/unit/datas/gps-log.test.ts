import { GPSLog } from '@/classes/datas/gps-log'

describe('GPSLog', () => {
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
