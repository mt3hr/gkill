import { GitCommitLog } from '@/classes/datas/git-commit-log'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('GitCommitLog', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new GitCommitLog())
  })

  describe('clear_attached_histories', () => {
    test('clears attached_histories to empty array', async () => {
      const log = new GitCommitLog()
      const errors = await log.clear_attached_histories()
      expect(errors).toEqual([])
      expect(log.attached_histories).toEqual([])
    })
  })
})
