import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'

describe('ShareKyousInfo', () => {
  test('can be instantiated', () => {
    const info = new ShareKyousInfo()
    expect(info).toBeInstanceOf(ShareKyousInfo)
  })

  describe('default field values', () => {
    let info: ShareKyousInfo

    beforeEach(() => {
      info = new ShareKyousInfo()
    })

    test('share_id defaults to empty string', () => {
      expect(info.share_id).toBe('')
    })

    test('user_id defaults to empty string', () => {
      expect(info.user_id).toBe('')
    })

    test('device defaults to empty string', () => {
      expect(info.device).toBe('')
    })

    test('share_title defaults to empty string', () => {
      expect(info.share_title).toBe('')
    })

    test('view_type defaults to rykv', () => {
      expect(info.view_type).toBe('rykv')
    })

    test('is_share_time_only defaults to false', () => {
      expect(info.is_share_time_only).toBe(false)
    })

    test('is_share_with_tags defaults to false', () => {
      expect(info.is_share_with_tags).toBe(false)
    })

    test('is_share_with_texts defaults to false', () => {
      expect(info.is_share_with_texts).toBe(false)
    })

    test('is_share_with_timeiss defaults to false', () => {
      expect(info.is_share_with_timeiss).toBe(false)
    })

    test('is_share_with_locations defaults to false', () => {
      expect(info.is_share_with_locations).toBe(false)
    })

    test('find_query_json defaults to a FindKyouQuery instance', () => {
      expect(info.find_query_json).toBeDefined()
      expect(info.find_query_json).not.toBeNull()
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const info = new ShareKyousInfo()
      info.share_id = 'share-001'
      info.user_id = 'admin'
      info.device = 'my-phone'
      info.share_title = 'My shared records'
      info.view_type = 'mi'
      info.is_share_time_only = true
      info.is_share_with_tags = true
      info.is_share_with_texts = true
      info.is_share_with_timeiss = true
      info.is_share_with_locations = true

      const cloned = info.clone()

      expect(cloned).toBeInstanceOf(ShareKyousInfo)
      expect(cloned.share_id).toBe('share-001')
      expect(cloned.user_id).toBe('admin')
      expect(cloned.device).toBe('my-phone')
      expect(cloned.share_title).toBe('My shared records')
      expect(cloned.view_type).toBe('mi')
      expect(cloned.is_share_time_only).toBe(true)
      expect(cloned.is_share_with_tags).toBe(true)
      expect(cloned.is_share_with_texts).toBe(true)
      expect(cloned.is_share_with_timeiss).toBe(true)
      expect(cloned.is_share_with_locations).toBe(true)
    })

    test('clone returns a different instance', () => {
      const info = new ShareKyousInfo()
      const cloned = info.clone()
      expect(cloned).not.toBe(info)
    })
  })
})
