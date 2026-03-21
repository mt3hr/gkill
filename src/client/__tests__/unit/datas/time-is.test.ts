import { TimeIs } from '@/classes/datas/time-is'

describe('TimeIs', () => {
  test('can be instantiated', () => {
    const timeis = new TimeIs()
    expect(timeis).toBeInstanceOf(TimeIs)
  })

  describe('default field values', () => {
    let timeis: TimeIs

    beforeEach(() => {
      timeis = new TimeIs()
    })

    test('title defaults to empty string and times have defaults', () => {
      expect(timeis.title).toBe('')
      expect(timeis.start_time).toEqual(new Date(0))
      expect(timeis.end_time).toBeNull()
    })

    test('attached_histories defaults to empty array', () => {
      expect(timeis.attached_histories).toEqual([])
    })

    test('inherited InfoBase fields have defaults', () => {
      expect(timeis.is_deleted).toBe(false)
      expect(timeis.id).toBe('')
      expect(timeis.rep_name).toBe('')
      expect(timeis.data_type).toBe('')
      expect(timeis.create_app).toBe('')
      expect(timeis.create_device).toBe('')
      expect(timeis.create_user).toBe('')
      expect(timeis.update_app).toBe('')
      expect(timeis.update_user).toBe('')
      expect(timeis.update_device).toBe('')
      expect(timeis.related_time).toBeInstanceOf(Date)
      expect(timeis.create_time).toBeInstanceOf(Date)
      expect(timeis.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const timeis = new TimeIs()
      timeis.id = 'timeis-001'
      timeis.title = 'Working'
      timeis.rep_name = 'my-rep'
      timeis.start_time = new Date('2024-06-01T09:00:00Z')
      timeis.end_time = new Date('2024-06-01T17:00:00Z')
      timeis.is_deleted = true

      const cloned = timeis.clone()

      expect(cloned).toBeInstanceOf(TimeIs)
      expect(cloned.id).toBe('timeis-001')
      expect(cloned.title).toBe('Working')
      expect(cloned.rep_name).toBe('my-rep')
      expect(cloned.start_time).toEqual(new Date('2024-06-01T09:00:00Z'))
      expect(cloned.end_time).toEqual(new Date('2024-06-01T17:00:00Z'))
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const timeis = new TimeIs()
      const cloned = timeis.clone()
      expect(cloned).not.toBe(timeis)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const timeis = new TimeIs()
      timeis.id = 'timeis-789'
      timeis.create_time = new Date('2024-05-10T08:00:00Z')
      timeis.update_time = new Date('2024-05-11T08:00:00Z')

      const identifier = timeis.generate_info_identifer()

      expect(identifier.id).toBe('timeis-789')
      expect(identifier.create_time).toEqual(new Date('2024-05-10T08:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-05-11T08:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const timeis = new TimeIs()
      const errors = await timeis.clear_attached_histories()
      expect(errors).toEqual([])
      expect(timeis.attached_histories).toEqual([])
    })
  })
})
