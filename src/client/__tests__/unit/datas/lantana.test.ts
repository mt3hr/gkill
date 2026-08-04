import { Lantana } from '@/classes/datas/lantana'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('Lantana', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new Lantana())
  })

  describe('generate_info_identifier', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const lantana = new Lantana()
      lantana.id = 'lantana-456'
      lantana.create_time = new Date('2024-08-01T06:00:00Z')
      lantana.update_time = new Date('2024-08-02T06:00:00Z')

      const identifier = lantana.generate_info_identifier()

      expect(identifier.id).toBe('lantana-456')
      expect(identifier.create_time).toEqual(new Date('2024-08-01T06:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-08-02T06:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const lantana = new Lantana()
      const errors = await lantana.clear_attached_histories()
      expect(errors).toEqual([])
      expect(lantana.attached_histories).toEqual([])
    })
  })
})
