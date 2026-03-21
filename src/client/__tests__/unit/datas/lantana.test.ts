import { Lantana } from '@/classes/datas/lantana'

describe('Lantana', () => {
  test('can be instantiated', () => {
    const lantana = new Lantana()
    expect(lantana).toBeInstanceOf(Lantana)
  })

  describe('default field values', () => {
    let lantana: Lantana

    beforeEach(() => {
      lantana = new Lantana()
    })

    test('mood defaults to 0', () => {
      expect(lantana.mood).toBe(0)
    })

    test('attached_histories defaults to empty array', () => {
      expect(lantana.attached_histories).toEqual([])
    })

    test('inherited InfoBase fields have defaults', () => {
      expect(lantana.is_deleted).toBe(false)
      expect(lantana.id).toBe('')
      expect(lantana.rep_name).toBe('')
      expect(lantana.data_type).toBe('')
      expect(lantana.create_app).toBe('')
      expect(lantana.create_device).toBe('')
      expect(lantana.create_user).toBe('')
      expect(lantana.update_app).toBe('')
      expect(lantana.update_user).toBe('')
      expect(lantana.update_device).toBe('')
      expect(lantana.related_time).toBeInstanceOf(Date)
      expect(lantana.create_time).toBeInstanceOf(Date)
      expect(lantana.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const lantana = new Lantana()
      lantana.id = 'lantana-001'
      lantana.mood = 8
      lantana.rep_name = 'my-rep'
      lantana.is_deleted = true

      const cloned = lantana.clone()

      expect(cloned).toBeInstanceOf(Lantana)
      expect(cloned.id).toBe('lantana-001')
      expect(cloned.mood).toBe(8)
      expect(cloned.rep_name).toBe('my-rep')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const lantana = new Lantana()
      const cloned = lantana.clone()
      expect(cloned).not.toBe(lantana)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const lantana = new Lantana()
      lantana.id = 'lantana-456'
      lantana.create_time = new Date('2024-08-01T06:00:00Z')
      lantana.update_time = new Date('2024-08-02T06:00:00Z')

      const identifier = lantana.generate_info_identifer()

      expect(identifier.id).toBe('lantana-456')
      expect(identifier.create_time).toEqual(new Date('2024-08-01T06:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-08-02T06:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const lantana = new Lantana()
      const errors = await lantana.clear_attached_histories()
      expect(errors).toEqual([])
      expect(lantana.attached_histories).toEqual([])
    })
  })
})
