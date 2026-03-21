import { Text } from '@/classes/datas/text'

describe('Text', () => {
  test('can be instantiated', () => {
    const text = new Text()
    expect(text).toBeInstanceOf(Text)
  })

  describe('default field values', () => {
    let text: Text

    beforeEach(() => {
      text = new Text()
    })

    test('text defaults to empty string', () => {
      expect(text.text).toBe('')
    })

    test('attached_histories defaults to empty array', () => {
      expect(text.attached_histories).toEqual([])
    })

    test('inherited MetaInfoBase fields have defaults', () => {
      expect(text.is_deleted).toBe(false)
      expect(text.id).toBe('')
      expect(text.target_id).toBe('')
      expect(text.create_app).toBe('')
      expect(text.create_device).toBe('')
      expect(text.create_user).toBe('')
      expect(text.update_app).toBe('')
      expect(text.update_device).toBe('')
      expect(text.update_user).toBe('')
      expect(text.related_time).toBeInstanceOf(Date)
      expect(text.create_time).toBeInstanceOf(Date)
      expect(text.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const text = new Text()
      text.id = 'text-001'
      text.target_id = 'kyou-abc'
      text.text = 'Some descriptive text'
      text.is_deleted = true

      const cloned = text.clone()

      expect(cloned).toBeInstanceOf(Text)
      expect(cloned.id).toBe('text-001')
      expect(cloned.target_id).toBe('kyou-abc')
      expect(cloned.text).toBe('Some descriptive text')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const text = new Text()
      const cloned = text.clone()
      expect(cloned).not.toBe(text)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const text = new Text()
      text.id = 'text-789'
      text.create_time = new Date('2024-10-01T14:00:00Z')
      text.update_time = new Date('2024-10-02T14:00:00Z')

      const identifier = text.generate_info_identifer()

      expect(identifier.id).toBe('text-789')
      expect(identifier.create_time).toEqual(new Date('2024-10-01T14:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-10-02T14:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const text = new Text()
      const errors = await text.clear_attached_histories()
      expect(errors).toEqual([])
      expect(text.attached_histories).toEqual([])
    })
  })
})
