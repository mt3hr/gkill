import { Kmemo } from '@/classes/datas/kmemo'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('Kmemo', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new Kmemo())
  })

  describe('generate_info_identifier', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const kmemo = new Kmemo()
      kmemo.id = 'memo-456'
      kmemo.create_time = new Date('2024-03-01T12:00:00Z')
      kmemo.update_time = new Date('2024-03-02T12:00:00Z')

      const identifier = kmemo.generate_info_identifier()

      expect(identifier.id).toBe('memo-456')
      expect(identifier.create_time).toEqual(new Date('2024-03-01T12:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-03-02T12:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const kmemo = new Kmemo()
      const errors = await kmemo.clear_attached_histories()
      expect(errors).toEqual([])
      expect(kmemo.attached_histories).toEqual([])
    })
  })
})
