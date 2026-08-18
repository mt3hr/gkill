/**
 * useRykvView の初期化（画面を見せるまで）の検証。
 *
 * 以前は「保存済み列の初期検索が全部終わるまで画面全体を隠す」実装で、
 * サイドバーもテーブルも v-show で消え、全画面オーバーレイが被さっていた。
 * 検索が1本でも解決しないだけで画面全体が固まるうえ、初期化中は
 * サイドバーの編集を丸ごと捨てていた。
 *
 * いまは「列の骨組みを確定 → 可視化 → 検索」の順で、初期検索の完了は待たない。
 * 進行は列ごとのスピナーで見せる。この順序と、初期検索の飛行中に
 * ユーザの編集が勝つことをここで固定する。
 */
import { describe, test, expect, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/router', () => ({ default: { replace: vi.fn(), push: vi.fn() } }))
vi.mock('@/classes/delete-gkill-cache', () => ({
  default: vi.fn().mockResolvedValue(undefined),
  delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
  delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
  reset_dialog_history: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-scoped-enter-for-kftl', () => ({ useScopedEnterForKFTL: vi.fn() }))
vi.mock('@/classes/use-scoped-ctrl-v-for-clipboard', () => ({ useScopedCtrlVForClipboard: vi.fn() }))
vi.mock('@/classes/kyou-reload', () => ({
  build_mi_reload_query: vi.fn((query: unknown) => query),
  new_reload_batch: vi.fn(() => 0),
  refresh_kyou: vi.fn().mockResolvedValue(null),
  refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { useRykvView } from '@/classes/use-rykv-view'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { RykvViewProps } from '@/pages/views/rykv-view-props'
import type { RykvViewEmits } from '@/pages/views/rykv-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import {
  createColumnViewMockApi,
  makeColumnQuery,
  makeColumnViewProps,
  finish_application_config_load,
  attachFakeKyouListViews,
  flushAsync,
} from '../../helpers/rykv-mi-harness'

const noop_emits = (() => { }) as unknown as RykvViewEmits

function kyous(ids: string[]): Kyou[] {
  return ids.map((id) => ({ id })) as unknown as Kyou[]
}

function kyouIds(list: unknown): string[] {
  return (list as Array<{ id: string }>).map((kyou) => kyou.id)
}

function createView(saved_query_ids: string[], config_overrides: Record<string, unknown> = {}) {
  const { api, pending_get_kyous } = createColumnViewMockApi()
  const saved_querys = saved_query_ids.map((query_id) => makeColumnQuery(query_id))
  api.get_saved_rykv_find_kyou_querys.mockReturnValue(saved_querys)
  const raw_props = makeColumnViewProps(api, config_overrides, {
    is_shared_rykv_view: false,
    share_title: '',
  })
  const props = raw_props as unknown as RykvViewProps
  const view = useRykvView({ props, emits: noop_emits })
  view.query_editor_sidebar.value = { get_default_query: () => new FindKyouQuery() }
  // init() は application_config.is_loaded の watch で起動する
  const start_init = () => finish_application_config_load(raw_props)
  return { api, pending_get_kyous, view, saved_querys, start_init }
}

describe('useRykvView 初期化', () => {
  test('初期検索の完了を待たずに画面が見える', async () => {
    // 本件の本体。旧実装は Promise.all(初期検索) の then で inited を立てていた
    const { pending_get_kyous, view, start_init } = createView(['col-a', 'col-b'])

    start_init()
    await flushAsync()

    expect(view.inited.value, '初期検索の飛行中なのに画面が見えていない').toBe(true)
    expect(view.is_loading.value, 'オーバーレイが残っている').toBe(false)
    expect(pending_get_kyous.length, '初期検索が飛んでいない').toBe(2)
  })

  test('検索が1本も解決しなくても画面は固まらない', async () => {
    // #1 の裏返し。旧実装ならここで inited が false のまま
    const { pending_get_kyous, view, start_init } = createView(['col-a'])

    start_init()
    await flushAsync()
    await flushAsync()

    expect(pending_get_kyous.length, '検索が解決してしまっている').toBe(1)
    expect(view.inited.value).toBe(true)
    expect(view.drawer_mode_is_mobile.value, 'drawerの判定が未確定のまま見えている').toBe(false)
  })

  test('列の骨組みは検索を投げる前に確定している', async () => {
    const { view, start_init } = createView(['col-a', 'col-b'])

    start_init()
    await flushAsync()

    expect(view.querys.value.map((query) => query.query_id)).toEqual(['col-a', 'col-b'])
    expect(view.match_kyous_list.value.length, '列と結果リストの本数が合わない').toBe(2)
    expect(view.focused_column_index.value).toBe(0)
    expect(view.focused_query.value.query_id).toBe('col-a')
  })

  test('初期検索の飛行中でもサイドバーの編集が勝つ', async () => {
    const { pending_get_kyous, view, start_init } = createView(['col-a'])
    start_init()
    await flushAsync()
    expect(pending_get_kyous.length).toBe(1)
    attachFakeKyouListViews(view, ['col-a'])

    // 初期検索を解決させないままユーザが条件を変える
    const edited = makeColumnQuery('col-a')
    edited.keywords = 'user-edit'
    view.onSidebarUpdatedQuery(edited)
    await flushAsync()

    expect(pending_get_kyous.length, 'ユーザの編集が検索にならなかった').toBe(2)
    expect(pending_get_kyous[1].req.query.keywords).toBe('user-edit')

    // 遅れて着地した初期検索の結果は捨てられる（世代照合）
    pending_get_kyous[0].resolve({ kyous: kyous(['stale']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[0]), '古い初期検索の結果で上書きされた').toEqual([])

    pending_get_kyous[1].resolve({ kyous: kyous(['fresh']), messages: [], errors: [] })
    await flushAsync()
    expect(kyouIds(view.match_kyous_list.value[0])).toEqual(['fresh'])
  })

  test('1列目の完了で抑止が解けて2列目が誤検索されない', async () => {
    // 初期化の間じゅう skip_search_this_tick を立てっぱなしにしていたころは、
    // 機械的なemitが1つ届くだけで抑止が消費され、以降の残響が検索になっていた
    const { pending_get_kyous, view, saved_querys, start_init } = createView(['col-a', 'col-b'])
    start_init()
    await flushAsync()
    attachFakeKyouListViews(view, ['col-a', 'col-b'])
    expect(pending_get_kyous.length).toBe(2)

    pending_get_kyous[0].resolve({ kyous: kyous(['a1']), messages: [], errors: [] })
    await flushAsync()

    // 列1と同値の機械的な updated_query（サイドバーのprops同期の残響）
    view.onSidebarUpdatedQuery(saved_querys[1].clone())
    await flushAsync()

    expect(pending_get_kyous.length, '無変更の残響が検索になった').toBe(2)
  })

  test('初期検索中の列スクロール通知は保存スクロール位置を壊さない', async () => {
    // 検索開始でリストを空にする副作用でスクロール通知が届く。
    // 取り込むと preserve_scroll の復元先が0で潰れ、保存位置にも焼き付く
    const { api, pending_get_kyous, view, start_init } = createView(['col-a'])
    api.get_saved_rykv_scroll_indexs.mockReturnValue([120])
    start_init()
    await flushAsync()
    const fakes = attachFakeKyouListViews(view, ['col-a'])
    fakes.get('col-a')!.set_loading(true)
    api.set_saved_rykv_scroll_indexs.mockClear()

    view.onColumnScrollList(0, 0)

    expect(api.set_saved_rykv_scroll_indexs, '検索中の機械的なスクロール通知を保存した').not.toHaveBeenCalled()

    pending_get_kyous[0].resolve({ kyous: kyous(['a1']), messages: [], errors: [] })
    await flushAsync()
    expect(fakes.get('col-a')!.scroll_to, '保存スクロール位置へ戻していない').toHaveBeenCalledWith(120)
    expect(fakes.get('col-a')!.scroll_to).not.toHaveBeenCalledWith(0)
  })

  test('復元完了後のユーザのスクロールは保存される', async () => {
    // #6 のガードが行き過ぎていないこと
    const { api, pending_get_kyous, view, start_init } = createView(['col-a'])
    start_init()
    await flushAsync()
    attachFakeKyouListViews(view, ['col-a'])
    pending_get_kyous[0].resolve({ kyous: kyous(['a1']), messages: [], errors: [] })
    await flushAsync()
    api.set_saved_rykv_scroll_indexs.mockClear()

    view.onColumnScrollList(0, 300)

    expect(api.set_saved_rykv_scroll_indexs).toHaveBeenCalledWith([300], '')
  })

  test('is_view_ready は復元中false・完了でtrue・再検索でまたfalse', async () => {
    const { pending_get_kyous, view, start_init } = createView(['col-a'])
    start_init()
    await flushAsync()
    attachFakeKyouListViews(view, ['col-a'])

    expect(view.is_view_ready.value, '復元中なのに準備完了になっている').toBe(false)

    pending_get_kyous[0].resolve({ kyous: kyous(['a1']), messages: [], errors: [] })
    await flushAsync()
    expect(view.is_view_ready.value, '復元が終わっても準備完了にならない').toBe(true)

    view.reload_list(0)
    await flushAsync()
    expect(view.is_view_ready.value, '再検索中なのに準備完了のまま').toBe(false)

    pending_get_kyous[1].resolve({ kyous: kyous(['a2']), messages: [], errors: [] })
    await flushAsync()
    expect(view.is_view_ready.value).toBe(true)
  })

  test('init は is_loaded が真になったときに走る', async () => {
    // 起動条件をサイドバーの @inited から application_config.is_loaded へ移した。
    // @inited は子ビューの「その節が描けた」の集約でしかなく、設定の到着を
    // 表していたのは偶然だった
    const { pending_get_kyous, view, start_init } = createView(['col-a'])

    await flushAsync()
    expect(view.inited.value, 'is_loaded が false なのに初期化が走った').toBe(false)
    expect(pending_get_kyous.length, 'is_loaded が false なのに検索が飛んだ').toBe(0)
    expect(view.querys.value.map((query) => query.query_id)).not.toContain('col-a')

    start_init()
    await flushAsync()
    expect(view.inited.value).toBe(true)
    expect(view.querys.value.map((query) => query.query_id)).toEqual(['col-a'])
  })

  test('is_loaded が複数回立っても init は1回だけ', async () => {
    // 設定ダイアログからの再読込などで ref が再度差し替わる
    const { view, start_init } = createView(['col-a'])
    start_init()
    await flushAsync()
    expect(view.querys.value.length).toBe(1)

    start_init()
    await flushAsync()

    expect(view.querys.value.length, '列が二重に作られた').toBe(1)
  })

  test('hot reload OFF でも即可視で、検索は飛ばさない', async () => {
    const { pending_get_kyous, view, start_init } = createView(['col-a'], { rykv_hot_reload: false })

    start_init()
    await flushAsync()

    expect(view.inited.value).toBe(true)
    expect(view.is_loading.value).toBe(false)
    expect(pending_get_kyous.length, 'hot reload OFF なのに検索が飛んだ').toBe(0)
    expect(view.is_view_ready.value, '検索が無いのに準備完了にならない').toBe(true)
  })
  /**
   * ポート(rudbeckia)は同じ画面を複数枚開ける。
   * 列の検索条件は localStorage の単一キーに入っていたので、2枚目が1枚目を上書きしていた。
   * 枝番(column_state_instance_key)で分ける約束をここで固定する。
   */
  describe('ライフログビューを2枚開く', () => {
    function createViewWithInstanceKey(instance_key: string, api: ReturnType<typeof createColumnViewMockApi>['api']) {
      const raw_props = makeColumnViewProps(api, {}, Object.assign({}, { is_shared_rykv_view: false, share_title: '' }, { column_state_instance_key: instance_key }))
      const props = raw_props as unknown as RykvViewProps
      const view = useRykvView({ props, emits: noop_emits })
  view.query_editor_sidebar.value = { get_default_query: () => new FindKyouQuery() }
      return { view, start_init: () => finish_application_config_load(raw_props) }
    }

    test('保存も読み出しも自分の枝番のキーで行う', async () => {
      const { api } = createColumnViewMockApi()

      const first = createViewWithInstanceKey('', api)
      const second = createViewWithInstanceKey('2', api)
      first.start_init()
      second.start_init()
      await flushAsync()

      expect(api.get_saved_rykv_find_kyou_querys, '1枚目が自分の枝番で読んでいない').toHaveBeenCalledWith('')
      expect(api.get_saved_rykv_find_kyou_querys, '2枚目が自分の枝番で読んでいない').toHaveBeenCalledWith('2')
      for (const call of api.set_saved_rykv_find_kyou_querys.mock.calls) {
        expect(['', '2'], '知らない枝番へ保存している').toContain(call[1])
      }
    })

    // 2枚目は最初まっさら。1枚目の保存内容を種付けすると query_id が重複し、
    // query_id→列の逆引きが別インスタンスへ誤配送する
    test('2枚目は1枚目の列を引き継がない', async () => {
      const { api } = createColumnViewMockApi()
      api.set_saved_rykv_find_kyou_querys([makeColumnQuery('col-a')], '')

      const second = createViewWithInstanceKey('2', api)
      second.start_init()
      await flushAsync()

      const query_ids = second.view.querys.value.map((query) => query.query_id)
      expect(query_ids, '2枚目が1枚目の列条件を引き継いでいる').not.toContain('col-a')
    })

    test('2枚目の保存は1枚目を壊さない', async () => {
      const { api } = createColumnViewMockApi()
      api.set_saved_rykv_find_kyou_querys([makeColumnQuery('col-a')], '')

      const second = createViewWithInstanceKey('2', api)
      second.start_init()
      await flushAsync()

      const first_saved = api.get_saved_rykv_find_kyou_querys('') as Array<{ query_id: string }>
      expect(first_saved.map((query) => query.query_id), '2枚目の初期化で1枚目の列条件が消えた').toEqual(['col-a'])
    })
  })
})