import { Tag } from '@/classes/datas/tag'

describe('Tag', () => {
  test('can be instantiated', () => {
    const tag = new Tag()
    expect(tag).toBeInstanceOf(Tag)
  })

  describe('default field values', () => {
    let tag: Tag

    beforeEach(() => {
      tag = new Tag()
    })

    test('tag defaults to empty string', () => {
      expect(tag.tag).toBe('')
    })

    test('attached_histories defaults to empty array', () => {
      expect(tag.attached_histories).toEqual([])
    })

    test('inherited MetaInfoBase fields have defaults', () => {
      expect(tag.is_deleted).toBe(false)
      expect(tag.id).toBe('')
      expect(tag.target_id).toBe('')
      expect(tag.create_app).toBe('')
      expect(tag.create_device).toBe('')
      expect(tag.create_user).toBe('')
      expect(tag.update_app).toBe('')
      expect(tag.update_device).toBe('')
      expect(tag.update_user).toBe('')
      expect(tag.related_time).toBeInstanceOf(Date)
      expect(tag.create_time).toBeInstanceOf(Date)
      expect(tag.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const tag = new Tag()
      tag.id = 'tag-001'
      tag.target_id = 'kyou-abc'
      tag.tag = 'important'
      tag.is_deleted = true

      const cloned = tag.clone()

      expect(cloned).toBeInstanceOf(Tag)
      expect(cloned.id).toBe('tag-001')
      expect(cloned.target_id).toBe('kyou-abc')
      expect(cloned.tag).toBe('important')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const tag = new Tag()
      const cloned = tag.clone()
      expect(cloned).not.toBe(tag)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const tag = new Tag()
      tag.id = 'tag-789'
      tag.create_time = new Date('2024-05-10T08:00:00Z')
      tag.update_time = new Date('2024-05-11T08:00:00Z')

      const identifier = tag.generate_info_identifer()

      expect(identifier.id).toBe('tag-789')
      expect(identifier.create_time).toEqual(new Date('2024-05-10T08:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-05-11T08:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const tag = new Tag()
      const errors = await tag.clear_attached_histories()
      expect(errors).toEqual([])
      expect(tag.attached_histories).toEqual([])
    })
  })
})
