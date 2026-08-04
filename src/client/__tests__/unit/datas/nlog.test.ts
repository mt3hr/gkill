import { Nlog } from '@/classes/datas/nlog'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('Nlog', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new Nlog())
  })

  describe('generate_info_identifier', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const nlog = new Nlog()
      nlog.id = 'nlog-456'
      nlog.create_time = new Date('2024-07-01T12:00:00Z')
      nlog.update_time = new Date('2024-07-02T12:00:00Z')

      const identifier = nlog.generate_info_identifier()

      expect(identifier.id).toBe('nlog-456')
      expect(identifier.create_time).toEqual(new Date('2024-07-01T12:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-07-02T12:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const nlog = new Nlog()
      const errors = await nlog.clear_attached_histories()
      expect(errors).toEqual([])
      expect(nlog.attached_histories).toEqual([])
    })
  })
})
