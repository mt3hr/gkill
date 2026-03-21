import { URLog } from '@/classes/datas/ur-log'

describe('URLog', () => {
  test('can be instantiated', () => {
    const urlog = new URLog()
    expect(urlog).toBeInstanceOf(URLog)
  })

  describe('default field values', () => {
    let urlog: URLog

    beforeEach(() => {
      urlog = new URLog()
    })

    test('specific fields default to empty strings', () => {
      expect(urlog.url).toBe('')
      expect(urlog.title).toBe('')
      expect(urlog.description).toBe('')
      expect(urlog.favicon_image).toBe('')
      expect(urlog.thumbnail_image).toBe('')
    })

    test('attached_histories defaults to empty array', () => {
      expect(urlog.attached_histories).toEqual([])
    })

    test('inherited InfoBase fields have defaults', () => {
      expect(urlog.is_deleted).toBe(false)
      expect(urlog.id).toBe('')
      expect(urlog.rep_name).toBe('')
      expect(urlog.data_type).toBe('')
      expect(urlog.create_app).toBe('')
      expect(urlog.create_device).toBe('')
      expect(urlog.create_user).toBe('')
      expect(urlog.update_app).toBe('')
      expect(urlog.update_user).toBe('')
      expect(urlog.update_device).toBe('')
      expect(urlog.related_time).toBeInstanceOf(Date)
      expect(urlog.create_time).toBeInstanceOf(Date)
      expect(urlog.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const urlog = new URLog()
      urlog.id = 'urlog-001'
      urlog.url = 'https://example.com'
      urlog.title = 'Example Site'
      urlog.description = 'An example website'
      urlog.favicon_image = 'favicon.png'
      urlog.thumbnail_image = 'thumb.png'
      urlog.rep_name = 'my-rep'
      urlog.is_deleted = true

      const cloned = urlog.clone()

      expect(cloned).toBeInstanceOf(URLog)
      expect(cloned.id).toBe('urlog-001')
      expect(cloned.url).toBe('https://example.com')
      expect(cloned.title).toBe('Example Site')
      expect(cloned.description).toBe('An example website')
      expect(cloned.favicon_image).toBe('favicon.png')
      expect(cloned.thumbnail_image).toBe('thumb.png')
      expect(cloned.rep_name).toBe('my-rep')
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const urlog = new URLog()
      const cloned = urlog.clone()
      expect(cloned).not.toBe(urlog)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const urlog = new URLog()
      urlog.id = 'urlog-456'
      urlog.create_time = new Date('2024-04-01T10:00:00Z')
      urlog.update_time = new Date('2024-04-02T10:00:00Z')

      const identifier = urlog.generate_info_identifer()

      expect(identifier.id).toBe('urlog-456')
      expect(identifier.create_time).toEqual(new Date('2024-04-01T10:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-04-02T10:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const urlog = new URLog()
      const errors = await urlog.clear_attached_histories()
      expect(errors).toEqual([])
      expect(urlog.attached_histories).toEqual([])
    })
  })
})
