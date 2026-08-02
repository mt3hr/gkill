import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { expectCloneCopiesAllFields } from '../../helpers/clone-parity'

describe('ShareKyousInfo', () => {
  test('clone が全フィールドをコピーする', () => {
    expectCloneCopiesAllFields(new ShareKyousInfo())
  })
})
