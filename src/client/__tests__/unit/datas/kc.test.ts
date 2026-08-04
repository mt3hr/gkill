import { KC } from '@/classes/datas/kc'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('KC', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new KC())
  })

  describe('generate_info_identifier', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const kc = new KC()
      kc.id = 'kc-456'
      kc.create_time = new Date('2024-09-01T15:00:00Z')
      kc.update_time = new Date('2024-09-02T15:00:00Z')

      const identifier = kc.generate_info_identifier()

      expect(identifier.id).toBe('kc-456')
      expect(identifier.create_time).toEqual(new Date('2024-09-01T15:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-09-02T15:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const kc = new KC()
      const errors = await kc.clear_attached_histories()
      expect(errors).toEqual([])
      expect(kc.attached_histories).toEqual([])
    })
  })
})
