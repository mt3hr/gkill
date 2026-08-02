import { Notification } from '@/classes/datas/notification'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('Notification', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new Notification())
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
