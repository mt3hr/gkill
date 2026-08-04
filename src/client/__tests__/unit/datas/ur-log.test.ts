import { URLog } from '@/classes/datas/ur-log'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('URLog', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new URLog())
  })

  describe('generate_info_identifier', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const urlog = new URLog()
      urlog.id = 'urlog-456'
      urlog.create_time = new Date('2024-04-01T10:00:00Z')
      urlog.update_time = new Date('2024-04-02T10:00:00Z')

      const identifier = urlog.generate_info_identifier()

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
