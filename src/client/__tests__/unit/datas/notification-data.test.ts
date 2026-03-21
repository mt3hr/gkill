import { Notification } from '@/classes/datas/notification'

describe('Notification', () => {
  test('can be instantiated', () => {
    const notification = new Notification()
    expect(notification).toBeInstanceOf(Notification)
  })

  describe('default field values', () => {
    let notification: Notification

    beforeEach(() => {
      notification = new Notification()
    })

    test('content defaults to empty string', () => {
      expect(notification.content).toBe('')
    })

    test('is_notificated defaults to false', () => {
      expect(notification.is_notificated).toBe(false)
    })

    test('notification_time defaults to epoch', () => {
      expect(notification.notification_time).toBeInstanceOf(Date)
      expect(notification.notification_time.getTime()).toBe(0)
    })

    test('attached_histories defaults to empty array', () => {
      expect(notification.attached_histories).toEqual([])
    })

    test('inherited MetaInfoBase fields have defaults', () => {
      expect(notification.is_deleted).toBe(false)
      expect(notification.id).toBe('')
      expect(notification.target_id).toBe('')
      expect(notification.create_app).toBe('')
      expect(notification.create_device).toBe('')
      expect(notification.create_user).toBe('')
      expect(notification.update_app).toBe('')
      expect(notification.update_device).toBe('')
      expect(notification.update_user).toBe('')
      expect(notification.related_time).toBeInstanceOf(Date)
      expect(notification.create_time).toBeInstanceOf(Date)
      expect(notification.update_time).toBeInstanceOf(Date)
    })
  })

  describe('clone', () => {
    test('clone creates a copy with same field values', () => {
      const notification = new Notification()
      notification.id = 'notif-001'
      notification.target_id = 'kyou-abc'
      notification.content = 'Reminder: check task'
      notification.is_notificated = true
      notification.notification_time = new Date('2025-06-01T12:00:00Z')
      notification.is_deleted = true

      const cloned = notification.clone()

      expect(cloned).toBeInstanceOf(Notification)
      expect(cloned.id).toBe('notif-001')
      expect(cloned.target_id).toBe('kyou-abc')
      expect(cloned.content).toBe('Reminder: check task')
      expect(cloned.is_notificated).toBe(true)
      expect(cloned.notification_time).toEqual(new Date('2025-06-01T12:00:00Z'))
      expect(cloned.is_deleted).toBe(true)
    })

    test('clone returns a different instance', () => {
      const notification = new Notification()
      const cloned = notification.clone()
      expect(cloned).not.toBe(notification)
    })
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const notification = new Notification()
      notification.id = 'notif-789'
      notification.create_time = new Date('2024-07-01T08:00:00Z')
      notification.update_time = new Date('2024-07-02T08:00:00Z')

      const identifier = notification.generate_info_identifer()

      expect(identifier.id).toBe('notif-789')
      expect(identifier.create_time).toEqual(new Date('2024-07-01T08:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-07-02T08:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const notification = new Notification()
      const errors = await notification.clear_attached_histories()
      expect(errors).toEqual([])
      expect(notification.attached_histories).toEqual([])
    })
  })
})
