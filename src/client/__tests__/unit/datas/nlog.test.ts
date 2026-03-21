import { Nlog } from '@/classes/datas/nlog'

describe('Nlog', () => {
  test('can be instantiated', () => {
    const nlog = new Nlog()
    expect(nlog).toBeInstanceOf(Nlog)
  })

  describe('default field values', () => {
    let nlog: Nlog

    beforeEach(() => {
      nlog = new Nlog()
    })

    test('specific fields have correct defaults', () => {
      expect(nlog.shop).toBe('')
      expect(nlog.title).toBe('')
      expect(nlog.amount).toBe(0)
    })

    test('attached_histories defaults to empty array', () => {
      expect(nlog.attached_histories).toEqual([])
    })

    test('inherited InfoBase fields have defaults', () => {
      expect(nlog.is_deleted).toBe(false)
      expect(nlog.id).toBe('')
      expect(nlog.rep_name).toBe('')
      expect(nlog.data_type).toBe('')
      expect(nlog.create_app).toBe('')
      expect(nlog.create_device).toBe('')
      expect(nlog.create_user).toBe('')
      expect(nlog.update_app).toBe('')
      expect(nlog.update_user).toBe('')
      expect(nlog.update_device).toBe('')
      expect(nlog.related_time).toBeInstanceOf(Date)
      expect(nlog.create_time).toBeInstanceOf(Date)
      expect(nlog.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const nlog = new Nlog()
      nlog.id = 'nlog-001'
      nlog.shop = 'Test Shop'
      nlog.title = 'Groceries'
      nlog.amount = 1500
      nlog.rep_name = 'my-rep'
      nlog.is_deleted = true

      const cloned = nlog.clone()

      expect(cloned).toBeInstanceOf(Nlog)
      expect(cloned.id).toBe('nlog-001')
      expect(cloned.shop).toBe('Test Shop')
      expect(cloned.title).toBe('Groceries')
      expect(cloned.amount).toBe(1500)
      expect(cloned.rep_name).toBe('my-rep')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const nlog = new Nlog()
      const cloned = nlog.clone()
      expect(cloned).not.toBe(nlog)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const nlog = new Nlog()
      nlog.id = 'nlog-456'
      nlog.create_time = new Date('2024-07-01T12:00:00Z')
      nlog.update_time = new Date('2024-07-02T12:00:00Z')

      const identifier = nlog.generate_info_identifer()

      expect(identifier.id).toBe('nlog-456')
      expect(identifier.create_time).toEqual(new Date('2024-07-01T12:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-07-02T12:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const nlog = new Nlog()
      const errors = await nlog.clear_attached_histories()
      expect(errors).toEqual([])
      expect(nlog.attached_histories).toEqual([])
    })
  })
})
