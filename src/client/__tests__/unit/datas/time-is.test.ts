import { TimeIs } from '@/classes/datas/time-is'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('TimeIs', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new TimeIs())
  })

  describe('generate_info_identifier', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const timeis = new TimeIs()
      timeis.id = 'timeis-789'
      timeis.create_time = new Date('2024-05-10T08:00:00Z')
      timeis.update_time = new Date('2024-05-11T08:00:00Z')

      const identifier = timeis.generate_info_identifier()

      expect(identifier.id).toBe('timeis-789')
      expect(identifier.create_time).toEqual(new Date('2024-05-10T08:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-05-11T08:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const timeis = new TimeIs()
      const errors = await timeis.clear_attached_histories()
      expect(errors).toEqual([])
      expect(timeis.attached_histories).toEqual([])
    })
  })
})
