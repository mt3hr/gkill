/**
 * useSharedRykvView（共有された rykv 画面）。
 *
 * 共有は通常の rykv（use-rykv-view）から**別実装として切り出した**ので、
 * 以前 use-rykv-view の早期 return が守っていたことをここで固定する。
 * どれも「例外もエラーも出ないまま静かに壊れる」たぐい。
 *
 * 1. 取得は1回だけで、**検索条件を渡さない**。何を共有するかはサーバ側（共有リンクに
 *    紐づく検索条件）が決めている。ここで条件を足すと共有した本人の意図より狭くなる。
 * 2. 取得が失敗してもスピナーを倒す。立てっぱなしは「読み込み中のまま固まる」画面になり、
 *    画面からは原因が何も分からない。
 * 3. 地図の時間窓は**幅0にしない**。GPSLog の取得は両端を含む厳密な範囲比較なので、
 *    開始と終了が同じ時刻だと軌跡が1本も出ない。
 * 4. 件数カレンダーの日付選択は一覧と地図の両方を動かし、地図の軌跡クリックは
 *    地図の窓を動かさない（地図だけ別の日を見ている状態を壊さない）。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
  default: { global: { t: (key: string) => key, locale: 'ja' } },
  i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/classes/kyou-reload', () => ({
  new_reload_batch: vi.fn(() => 0),
  refresh_kyou: vi.fn().mockResolvedValue(null),
  refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
}))

// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { useSharedRykvView } from '@/classes/use-shared-rykv-view'
import type { Kyou } from '@/classes/datas/kyou'

async function flush(): Promise<void> {
  for (let i = 0; i < 10; i++) {
    await Promise.resolve()
  }
}

function make_kyou(id: string, related_time: Date): Kyou {
  return {
    id: id,
    related_time: related_time,
    load_all: vi.fn().mockResolvedValue([]),
    clone: () => ({ id: id, related_time: related_time }),
  } as unknown as Kyou
}

function build(kyous: Array<Kyou> = []) {
  const api = {
    generate_uuid: vi.fn(() => 'dialog-1'),
    delete_updated_gkill_caches: vi.fn().mockResolvedValue(undefined),
    get_kyous: vi.fn().mockResolvedValue({ kyous: kyous, messages: [], errors: [] }),
    find_kyous: vi.fn().mockResolvedValue({ kyous: [], messages: [], errors: [] }),
  }
  const emits = vi.fn()
  const view = useSharedRykvView({
    props: {
      gkill_api: api,
      share_title: '共有',
      app_title_bar_height: 50,
      app_content_height: 600,
      app_content_width: 800,
    } as never,
    emits: emits as never,
  })
  return { api, emits, view }
}

describe('useSharedRykvView の初期化', () => {
  test('取得は1回だけで、検索条件で絞らない', async () => {
    const kyou = make_kyou('kyou-1', new Date('2024-01-15T10:30:00Z'))
    const { api, view } = build([kyou])

    await flush()

    expect(api.get_kyous).toHaveBeenCalledTimes(1)
    // 共有リンクに紐づく検索結果をそのまま見せるので、条件は1つも足さない
    // （GetKyousRequest は既定の条件を必ず持つので、「既定のまま」であることを見る）
    const request = api.get_kyous.mock.calls[0][0] as GetKyousRequest
    expect(
      JSON.stringify(request.query),
      '共有の取得に検索条件を足している。共有した本人の意図より狭くなる',
    ).toBe(JSON.stringify(new GetKyousRequest().query))
    expect(api.find_kyous, '共有で列の検索が走っている').not.toHaveBeenCalled()
    expect(view.match_kyous.value.map(matched => matched.id)).toEqual(['kyou-1'])
    expect(view.is_loading.value).toBe(false)
  })

  test('取得の間はスピナーを立て、失敗しても倒す', async () => {
    const { api, view } = build()
    // build() の直後はまだ初期化の nextTick が走っていない（マイクロタスク）ので、
    // ここで差し替えると初期化そのものがこの保留中の取得を使う
    let reject_get_kyous: (err: Error) => void = () => { }
    api.get_kyous.mockReturnValue(new Promise((_resolve, reject) => { reject_get_kyous = reject }))
    const logged = vi.spyOn(console, 'error').mockImplementation(() => { })

    await flush()
    expect(view.is_loading.value, '取得中なのにスピナーが立っていない').toBe(true)

    reject_get_kyous(new Error('共有の取得に失敗'))
    await flush()

    expect(view.is_loading.value, 'スピナーが立ったままになっている').toBe(false)
    expect(logged, '失敗が握りつぶされて console にも出ていない').toHaveBeenCalled()
    logged.mockRestore()
  })
})

describe('useSharedRykvView の地図の時間窓', () => {
  test('記録をクリックすると窓はその日の丸ごとになる（幅0にしない）', async () => {
    const related_time = new Date('2024-01-15T10:30:00Z')
    const { view } = build()
    await flush()

    view.onClickedKyou(make_kyou('kyou-1', related_time))

    const start = view.gps_log_map_start_time.value
    const end = view.gps_log_map_end_time.value
    expect(
      end.getTime() - start.getTime(),
      '窓の幅が0。GPSLog の範囲比較は両端を含むので軌跡が1本も出ない',
    ).toBeGreaterThan(0)
    expect(start.getTime()).toBeLessThanOrEqual(related_time.getTime())
    expect(end.getTime()).toBeGreaterThanOrEqual(related_time.getTime())
    expect(view.gps_log_map_marker_time.value.getTime()).toBe(related_time.getTime())
    expect(view.focused_kyou.value?.id).toBe('kyou-1')
  })

  test('件数カレンダーの日付選択は一覧と地図の両方を動かす', async () => {
    const { view } = build()
    await flush()
    const picked = new Date('2024-02-20T00:00:00Z')

    view.onRequestedFocusTime(picked)

    expect(view.focused_time.value.getTime()).toBe(picked.getTime())
    expect(view.gps_log_map_marker_time.value.getTime()).toBe(picked.getTime())
  })

  test('地図の軌跡クリックは地図の窓を動かさない', async () => {
    const { view } = build()
    await flush()
    const before_start = view.gps_log_map_start_time.value
    const before_end = view.gps_log_map_end_time.value
    const picked = new Date('2024-02-20T08:00:00Z')

    view.onGpsLogMapRequestedFocusTime(picked)

    expect(view.focused_time.value.getTime()).toBe(picked.getTime())
    expect(view.gps_log_map_start_time.value).toBe(before_start)
    expect(view.gps_log_map_end_time.value).toBe(before_end)
  })
})
