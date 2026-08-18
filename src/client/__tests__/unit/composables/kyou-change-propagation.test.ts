/**
 * ポート(rudbeckia)で並べた画面のあいだの変更伝播。
 *
 * ここで固定するのは、設計上いちばん壊れやすい2点:
 *
 * 1. **ループしない。** ビューの中継ハンドラ（`crudRelayHandlers`）は適用したあと
 *    `emits(...)` する。購読側がそれを呼ぶとホストが再 publish して往復が止まらない。
 *    購読側へ渡してよいのは **emit を含まない適用関数だけ**。
 * 2. **発生元は自分の通知を受けない。** 受けると二重適用する。
 *    追加は `insert_kyou_sorted` の id 重複判定で救われるが、削除と引き直しは救われない。
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
  new_reload_batch: vi.fn(() => 42),
  refresh_kyou: vi.fn().mockResolvedValue(null),
  refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
}))

import '@/classes/api/gkill-api'
import { nextTick } from 'vue'
import { useRykvView } from '@/classes/use-rykv-view'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { create_kyou_change_bus } from '@/classes/kyou-change-bus'
import type { RykvViewProps } from '@/pages/views/rykv-view-props'
import type { RykvViewEmits } from '@/pages/views/rykv-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import {
  createColumnViewMockApi,
  makeColumnQuery,
  makeColumnViewProps,
  finish_application_config_load,
  setupColumns,
  flushAsync,
} from '../../helpers/rykv-mi-harness'

const noop_emits = (() => { }) as unknown as RykvViewEmits

function kyous(ids: string[]): Kyou[] {
  return ids.map((id) => ({ id: id })) as unknown as Kyou[]
}

/** ポートに載せた rykv ウィンドウ1枚 */
function createHostedView(origin_id: string, bus: ReturnType<typeof create_kyou_change_bus>) {
  const { api, pending_get_kyous } = createColumnViewMockApi()
  const raw_props = makeColumnViewProps(api, {}, {
    is_shared_rykv_view: false,
    share_title: '',
    is_hosted_in_dialog: true,
    column_state_instance_key: origin_id === 'window-a' ? '' : '2',
    kyou_change_channel: { bus: bus, origin_id: origin_id },
  })
  const props = raw_props as unknown as RykvViewProps
  const view = useRykvView({ props, emits: noop_emits })
  view.query_editor_sidebar.value = { get_default_query: () => new FindKyouQuery() }
  return {
    api,
    pending_get_kyous,
    view,
    start_init: () => finish_application_config_load(raw_props),
  }
}

/** 2枚の列へ同じ Kyou を並べた状態を作る */
async function setup_two_windows() {
  const bus = create_kyou_change_bus()
  const a = createHostedView('window-a', bus)
  const b = createHostedView('window-b', bus)
  // init() は通さない。購読は useRykvView() の中で張られるので、列だけ用意すれば足りる
  setupColumns(a.view, [makeColumnQuery('col-a')], [kyous(['k1', 'k2'])])
  setupColumns(b.view, [makeColumnQuery('col-b')], [kyous(['k1', 'k2'])])
  await nextTick()
  return { bus, a, b }
}

describe('画面間の変更伝播', () => {
  test('片方の削除がもう片方の一覧へ届く', async () => {
    const { a, b } = await setup_two_windows()

    a.view.crudRelayHandlers.deleted_kyou(kyous(['k1'])[0])
    await flushAsync()

    expect(
      b.view.match_kyous_list.value[0].map((kyou) => kyou.id),
      '他のウィンドウの一覧から消えていない',
    ).toEqual(['k2'])
  })

  /**
   * 購読側が中継束（emit する側）を呼ぶとホストが再 publish して往復が止まらない。
   * 通知が1件のまま増えないことで「適用関数だけを呼んでいる」ことを見る
   */
  test('伝播はループしない（通知は1件のまま）', async () => {
    const { bus, a } = await setup_two_windows()
    const before = bus.log.length

    a.view.crudRelayHandlers.deleted_kyou(kyous(['k1'])[0])
    await flushAsync()

    expect(bus.log.length - before, '通知が再発行されている（ループしている）').toBe(1)
  })

  test('発生元は自分の通知で二重適用しない', async () => {
    const { a } = await setup_two_windows()

    a.view.crudRelayHandlers.deleted_kyou(kyous(['k1'])[0])
    await flushAsync()

    // 自分のぶんは中継ハンドラの中で1回適用済み。通知でもう一度適用されると
    // 別の Kyou まで巻き添えで消える
    expect(a.view.match_kyous_list.value[0].map((kyou) => kyou.id)).toEqual(['k2'])
  })

  // requested_at を運ばないと kyou-reload.ts の合流が成立せず、
  // 同じ Kyou を画面の枚数ぶん取りに行く
  test('引き直しの合流キーは発生元が採番して通知に載る', async () => {
    const { bus, a } = await setup_two_windows()

    a.view.crudRelayHandlers.updated_kyou(kyous(['k1'])[0])
    await flushAsync()

    const notice = bus.log[bus.log.length - 1]
    expect(notice.change.kind).toBe('reload')
    expect(notice.requested_at, '発生元が採番した合流キーが載っていない').toBe(42)
  })

  // タグ/テキスト/通知の変更は updated_kyou を出さない。唯一の信号がこれ
  test('requested_reload_kyou も配る', async () => {
    const { bus, a } = await setup_two_windows()
    const before = bus.log.length

    a.view.allColumnsRequestHandlers.requested_reload_kyou(kyous(['k1'])[0])
    await flushAsync()

    expect(bus.log.length - before, 'タグの変更が他の画面へ届かない').toBe(1)
    expect(bus.log[bus.log.length - 1].change.kind).toBe('reload')
  })

  test('単独ページ（チャネルなし）では publish も購読もしない', async () => {
    const bus = create_kyou_change_bus()
    const { api } = createColumnViewMockApi()
    const raw_props = makeColumnViewProps(api, {}, {
      is_shared_rykv_view: false,
      share_title: '',
      kyou_change_channel: null,
    })
    const props = raw_props as unknown as RykvViewProps
    const view = useRykvView({ props, emits: noop_emits })
    view.query_editor_sidebar.value = { get_default_query: () => new FindKyouQuery() }
    setupColumns(view, [makeColumnQuery('col-solo')], [kyous(['k1'])])
    await nextTick()

    view.crudRelayHandlers.deleted_kyou(kyous(['k1'])[0])
    await flushAsync()

    expect(bus.log.length, '単独ページなのにバスへ流している').toBe(0)
  })
})
