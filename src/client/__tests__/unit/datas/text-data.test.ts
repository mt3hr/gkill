import { Text } from '@/classes/datas/text'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('Text', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new Text())
  })

  describe('generate_info_identifier', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const text = new Text()
      text.id = 'text-789'
      text.create_time = new Date('2024-10-01T14:00:00Z')
      text.update_time = new Date('2024-10-02T14:00:00Z')

      const identifier = text.generate_info_identifier()

      expect(identifier.id).toBe('text-789')
      expect(identifier.create_time).toEqual(new Date('2024-10-01T14:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-10-02T14:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const text = new Text()
      const errors = await text.clear_attached_histories()
      expect(errors).toEqual([])
      expect(text.attached_histories).toEqual([])
    })
  })
})
