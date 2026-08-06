// Kyou class has circular import chains that cause "Class extends value undefined"
// in jsdom. These tests use the plain-object factory to verify the data shape
// that the rest of the codebase relies on.

import { describe, test, expect } from 'vitest'
// 実クラスを触るテストのために、本番同様 gkill-api を先に評価させる。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// これを先に済ませないと class extends が undefined になる。
import '@/classes/api/gkill-api'
import { Kyou } from '@/classes/datas/kyou'
import { makeKyou } from '../../helpers/factory'

describe('Kyou (factory-based)', () => {
  test('makeKyou returns object with all required fields', () => {
    const kyou = makeKyou()
    expect(kyou.id).toBe('test-kyou-id')
    expect(kyou.is_deleted).toBe(false)
    expect(kyou.rep_name).toBe('test-rep')
    expect(kyou.data_type).toBe('kmemo')
    expect(kyou.related_time).toBeDefined()
    expect(kyou.create_time).toBeDefined()
    expect(kyou.create_app).toBe('gkill')
    expect(kyou.create_user).toBe('admin')
    expect(kyou.update_time).toBeDefined()
    expect(kyou.update_app).toBe('gkill')
    expect(kyou.update_user).toBe('admin')
  })

  test('makeKyou includes typed_* fields initialized to null', () => {
    const kyou = makeKyou()
    expect(kyou.typed_kmemo).toBeNull()
    expect(kyou.typed_urlog).toBeNull()
    expect(kyou.typed_nlog).toBeNull()
    expect(kyou.typed_timeis).toBeNull()
    expect(kyou.typed_mi).toBeNull()
    expect(kyou.typed_lantana).toBeNull()
    expect(kyou.typed_kc).toBeNull()
    expect(kyou.typed_idf_kyou).toBeNull()
    expect(kyou.typed_git_commit_log).toBeNull()
  })

  test('makeKyou includes image_source field', () => {
    const kyou = makeKyou()
    expect(kyou.image_source).toBe('')
  })

  test('makeKyou overrides work', () => {
    const kyou = makeKyou({ id: 'custom-id', data_type: 'timeis', is_deleted: true })
    expect(kyou.id).toBe('custom-id')
    expect(kyou.data_type).toBe('timeis')
    expect(kyou.is_deleted).toBe(true)
    // non-overridden fields keep defaults
    expect(kyou.rep_name).toBe('test-rep')
  })

  test('makeKyou creates independent objects', () => {
    const a = makeKyou()
    const b = makeKyou()
    a.id = 'modified'
    expect(b.id).toBe('test-kyou-id')
  })
})

describe('Kyou.load_typed_datas 未取得のプレースホルダ', () => {
  // ReKyou/MiReKyouは参照先を取りに行っている間、空のKyouを入れ子のKyouViewに渡す。
  // これを既知プレフィックスに当たらないKyouとしてプラグイン扱いすると、
  // rep_nameが空のままContent HTMLを取りに行き
  // 「プラグインが見つかりません」がサーバから返って表示されてしまう
  test('idが空ならプラグインKyouとして扱わない', async () => {
    const kyou = new Kyou()

    await kyou.load_typed_datas()

    expect(kyou.typed_plugin).toBeNull()
  })

  test('idが空なら読み込み済みにしない (中身が入ったら読み直させる)', async () => {
    const kyou = new Kyou()

    await kyou.load_typed_datas()

    expect(kyou.is_typed_data_loaded).toBe(false)
  })

  test('idがあって未知のdata_typeなら従来どおりプラグイン扱いする', async () => {
    // プラグインのdata_typeはプラグイン側の申告をそのまま使うので空にもなりうる。
    // だからこそ判定はdata_typeではなくidで行う
    const kyou = new Kyou()
    kyou.id = 'plugin-kyou-id'
    kyou.rep_name = 'my_plugin_rep'
    kyou.data_type = ''

    await kyou.load_typed_datas()

    expect(kyou.typed_plugin).toEqual({ rep_name: 'my_plugin_rep' })
    expect(kyou.is_typed_data_loaded).toBe(true)
  })
})

describe('Kyou.clear_typed_datas', () => {
  test('typed_pluginも消して読み込み済みフラグも戻す', async () => {
    // フラグを戻さないと次のload_typed_datas()が冒頭で早期returnし、
    // 種別データが二度と入らないKyouになる
    const kyou = new Kyou()
    kyou.id = 'plugin-kyou-id'
    kyou.rep_name = 'my_plugin_rep'
    await kyou.load_typed_datas()

    await kyou.clear_typed_datas()

    expect(kyou.typed_plugin).toBeNull()
    expect(kyou.is_typed_data_loaded).toBe(false)
  })
})
