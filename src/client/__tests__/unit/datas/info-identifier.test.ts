import { InfoIdentifier } from '@/classes/datas/info-identifier'

describe('InfoIdentifier', () => {
  test('can be instantiated', () => {
    const identifier = new InfoIdentifier()
    expect(identifier).toBeInstanceOf(InfoIdentifier)
  })

  describe('default field values', () => {
    let identifier: InfoIdentifier

    beforeEach(() => {
      identifier = new InfoIdentifier()
    })

    test('id defaults to empty string', () => {
      expect(identifier.id).toBe('')
    })

    test('create_time defaults to epoch', () => {
      expect(identifier.create_time).toBeInstanceOf(Date)
      expect(identifier.create_time.getTime()).toBe(0)
    })

    test('update_time defaults to epoch', () => {
      expect(identifier.update_time).toBeInstanceOf(Date)
      expect(identifier.update_time.getTime()).toBe(0)
    })
  })

  describe('field assignment', () => {
    test('can set id', () => {
      const identifier = new InfoIdentifier()
      identifier.id = 'abc-123'
      expect(identifier.id).toBe('abc-123')
    })

    test('can set create_time and update_time', () => {
      const identifier = new InfoIdentifier()
      const ct = new Date('2025-01-01T00:00:00Z')
      const ut = new Date('2025-06-15T12:00:00Z')
      identifier.create_time = ct
      identifier.update_time = ut
      expect(identifier.create_time).toBe(ct)
      expect(identifier.update_time).toBe(ut)
    })
  })
})
