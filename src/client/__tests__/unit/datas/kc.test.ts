import { KC } from '@/classes/datas/kc'

describe('KC', () => {
  test('can be instantiated', () => {
    const kc = new KC()
    expect(kc).toBeInstanceOf(KC)
  })

  describe('default field values', () => {
    let kc: KC

    beforeEach(() => {
      kc = new KC()
    })

    test('specific fields have correct defaults', () => {
      expect(kc.title).toBe('')
      expect(kc.num_value).toBe(0)
    })

    test('attached_histories defaults to empty array', () => {
      expect(kc.attached_histories).toEqual([])
    })

    test('inherited InfoBase fields have defaults', () => {
      expect(kc.is_deleted).toBe(false)
      expect(kc.id).toBe('')
      expect(kc.rep_name).toBe('')
      expect(kc.data_type).toBe('')
      expect(kc.create_app).toBe('')
      expect(kc.create_device).toBe('')
      expect(kc.create_user).toBe('')
      expect(kc.update_app).toBe('')
      expect(kc.update_user).toBe('')
      expect(kc.update_device).toBe('')
      expect(kc.related_time).toBeInstanceOf(Date)
      expect(kc.create_time).toBeInstanceOf(Date)
      expect(kc.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const kc = new KC()
      kc.id = 'kc-001'
      kc.title = 'Steps'
      kc.num_value = 10000
      kc.rep_name = 'my-rep'
      kc.is_deleted = true

      const cloned = kc.clone()

      expect(cloned).toBeInstanceOf(KC)
      expect(cloned.id).toBe('kc-001')
      expect(cloned.title).toBe('Steps')
      expect(cloned.num_value).toBe(10000)
      expect(cloned.rep_name).toBe('my-rep')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const kc = new KC()
      const cloned = kc.clone()
      expect(cloned).not.toBe(kc)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const kc = new KC()
      kc.id = 'kc-456'
      kc.create_time = new Date('2024-09-01T15:00:00Z')
      kc.update_time = new Date('2024-09-02T15:00:00Z')

      const identifier = kc.generate_info_identifer()

      expect(identifier.id).toBe('kc-456')
      expect(identifier.create_time).toEqual(new Date('2024-09-01T15:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-09-02T15:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const kc = new KC()
      const errors = await kc.clear_attached_histories()
      expect(errors).toEqual([])
      expect(kc.attached_histories).toEqual([])
    })
  })
})
