import { Tag } from '@/classes/datas/tag'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('Tag', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new Tag())
  })

  describe('generate_info_identifer', () => {
    test('returns InfoIdentifier with matching id and times', () => {
      const tag = new Tag()
      tag.id = 'tag-789'
      tag.create_time = new Date('2024-05-10T08:00:00Z')
      tag.update_time = new Date('2024-05-11T08:00:00Z')

      const identifier = tag.generate_info_identifer()

      expect(identifier.id).toBe('tag-789')
      expect(identifier.create_time).toEqual(new Date('2024-05-10T08:00:00Z'))
      expect(identifier.update_time).toEqual(new Date('2024-05-11T08:00:00Z'))
    })
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const tag = new Tag()
      const errors = await tag.clear_attached_histories()
      expect(errors).toEqual([])
      expect(tag.attached_histories).toEqual([])
    })
  })
})
