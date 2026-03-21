import { Kmemo } from '@/classes/datas/kmemo'

describe('Kmemo', () => {
  test('can be instantiated', () => {
    const kmemo = new Kmemo()
    expect(kmemo).toBeInstanceOf(Kmemo)
  })

  describe('default field values', () => {
    let kmemo: Kmemo

    beforeEach(() => {
      kmemo = new Kmemo()
    })

    test('content defaults to empty string', () => {
      expect(kmemo.content).toBe('')
    })

    test('attached_histories defaults to empty array', () => {
      expect(kmemo.attached_histories).toEqual([])
    })

    test('inherited InfoBase fields have defaults', () => {
      expect(kmemo.is_deleted).toBe(false)
      expect(kmemo.id).toBe('')
      expect(kmemo.rep_name).toBe('')
      expect(kmemo.data_type).toBe('')
      expect(kmemo.create_app).toBe('')
      expect(kmemo.create_device).toBe('')
      expect(kmemo.create_user).toBe('')
      expect(kmemo.update_app).toBe('')
      expect(kmemo.update_user).toBe('')
      expect(kmemo.update_device).toBe('')
      expect(kmemo.related_time).toBeInstanceOf(Date)
      expect(kmemo.create_time).toBeInstanceOf(Date)
      expect(kmemo.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const kmemo = new Kmemo()
      kmemo.id = 'memo-123'
      kmemo.content = 'Test memo content'
      kmemo.rep_name = 'my-rep'
      kmemo.is_deleted = true

      const cloned = kmemo.clone()

      expect(cloned).toBeInstanceOf(Kmemo)
      expect(cloned.id).toBe('memo-123')
      expect(cloned.content).toBe('Test memo content')
      expect(cloned.rep_name).toBe('my-rep')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const kmemo = new Kmemo()
      const cloned = kmemo.clone()
      expect(cloned).not.toBe(kmemo)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const kmemo = new Kmemo()
      kmemo.id = 'memo-456'
      kmemo.create_time = new Date('2024-03-01T12:00:00Z')
      kmemo.update_time = new Date('2024-03-02T12:00:00Z')

      const identifier = kmemo.generate_info_identifer()

      expect(identifier.id).toBe('memo-456')
      expect(identifier.create_time).toEqual(new Date('2024-03-01T12:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-03-02T12:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const kmemo = new Kmemo()
      const errors = await kmemo.clear_attached_histories()
      expect(errors).toEqual([])
      expect(kmemo.attached_histories).toEqual([])
    })
  })
})
