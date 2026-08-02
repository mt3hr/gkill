/**
 * useMiReKyouView composable tests.
 * 一覧の行に収まる表示 (compact) への切り替えと、参照先の1行サマリを検証する。
 */
import { describe, test, expect, vi } from 'vitest'
// use-mi-re-kyou-view は req_res 経由で GkillAPIRequest に依存する。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させないと class extends が undefined になる。
import '@/classes/api/gkill-api'
import { useMiReKyouView } from '@/classes/use-mi-re-kyou-view'
import type { MiReKyouViewProps } from '@/pages/views/mi-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { makeKyou, makeKyouWithKmemo, makeMiReKyou } from '../../helpers/factory'

const noop_emits = (() => { }) as unknown as KyouViewEmits

/** get_kyou が返す Kyou。本番は GkillAPI が Kyou のインスタンスに詰め直すのでメソッドが生えている */
function makeTargetKyou(content = '参照先のメモ') {
    return {
        ...makeKyouWithKmemo(content),
        load_typed_datas: vi.fn().mockResolvedValue([]),
    }
}

function createProps(options: {
    height?: number | string,
    target_kyou?: unknown,
    errors?: Array<unknown>,
    mirekyou?: Record<string, unknown>,
    draggable?: boolean,
} = {}): MiReKyouViewProps & { gkill_api: { get_kyou: ReturnType<typeof vi.fn> } } {
    const kyou_histories = options.target_kyou === null ? [] : [options.target_kyou ?? makeTargetKyou()]
    const gkill_api = {
        get_kyou: vi.fn().mockResolvedValue({ errors: options.errors ?? [], kyou_histories: kyou_histories }),
    }
    return {
        kyou: makeKyou({ data_type: 'mirekyou_create' }),
        mirekyou: makeMiReKyou(options.mirekyou ?? {}),
        gkill_api: gkill_api,
        application_config: { device: 'test-device', user_id: 'admin' },
        highlight_targets: [],
        height: options.height ?? 91,
        width: 400,
        draggable: options.draggable ?? true,
        is_readonly_mi_check: false,
        enable_context_menu: true,
        enable_dialog: true,
    } as unknown as MiReKyouViewProps & { gkill_api: { get_kyou: ReturnType<typeof vi.fn> } }
}

async function flush(): Promise<void> {
    // get_kyou → load_typed_datas → get_kyou_content_text の各awaitを消化する
    for (let i = 0; i < 20; i++) {
        await Promise.resolve()
        await new Promise((resolve) => setTimeout(resolve, 0))
    }
}

describe('useMiReKyouView is_compact', () => {
    // 実際に渡ってくる高さは 91 (タスクボードの行) / 180 (rykvの一覧) / 'unset' (詳細・ダイアログ)
    test.each([
        { height: 91 as number | string, expected: true, name: 'タスクボードの行高 91 はcompact' },
        { height: '91px', expected: true, name: '単位付きの文字列でもcompact' },
        { height: 180, expected: false, name: 'rykvの一覧 180 はcompactではない' },
        { height: 'unset', expected: false, name: "'unset' はcompactではない" },
    ])('$name', ({ height, expected }) => {
        const { is_compact } = useMiReKyouView({ props: createProps({ height }), emits: noop_emits })
        expect(is_compact.value).toBe(expected)
    })
})

describe('useMiReKyouView target_summary', () => {
    test('参照先の本文を1行サマリにする', async () => {
        const props = createProps({ target_kyou: makeTargetKyou('明日の会議資料を用意する') })
        const { target_summary } = useMiReKyouView({ props, emits: noop_emits })

        await flush()

        expect(props.gkill_api.get_kyou).toHaveBeenCalledWith(
            expect.objectContaining({ id: 'test-target-id' }),
        )
        expect(target_summary.value).toBe('明日の会議資料を用意する')
    })

    test('取得できるまでは「取得中」を入れて高さを確定させる', () => {
        const { target_summary } = useMiReKyouView({ props: createProps(), emits: noop_emits })
        // 空文字にするとレイアウトが後から動くので、必ず何か入っている
        expect(target_summary.value.length).toBeGreaterThan(0)
    })

    test('参照先が見つからないときはフォールバックのラベルにする', async () => {
        const props = createProps({ target_kyou: null })
        const { target_summary } = useMiReKyouView({ props, emits: noop_emits })

        await flush()

        expect(target_summary.value.length).toBeGreaterThan(0)
        expect(target_summary.value).not.toBe('参照先のメモ')
    })

    test('参照先がReKyouならもう1段たどって本文を出す', async () => {
        // リポストをタスク化した場合。attached_kyou は load_typed_datas では埋まらないので
        // get_kyou_content_text が target_id を引き直して解決する
        const nested = makeTargetKyou('リポストの中身')
        const rekyou_target = {
            ...makeKyou({ data_type: 'rekyou' }),
            typed_rekyou: { attached_kyou: null, target_id: 'nested-target' },
            load_typed_datas: vi.fn().mockResolvedValue([]),
        }
        const props = createProps({ target_kyou: rekyou_target })
        // 1回目はMiReKyouの参照先(ReKyou)、2回目はそのまた参照先(kmemo)
        props.gkill_api.get_kyou.mockImplementation((req: { id: string }) => Promise.resolve({
            errors: [],
            kyou_histories: [req.id === 'nested-target' ? nested : rekyou_target],
        }))

        const { target_summary } = useMiReKyouView({ props, emits: noop_emits })

        await flush()

        expect(props.gkill_api.get_kyou).toHaveBeenCalledWith(
            expect.objectContaining({ id: 'nested-target' }),
        )
        expect(target_summary.value).toBe('リポストの中身')
    })

    test('一覧の行では参照先の解決を1段までにする', async () => {
        // 行ごとに直列リクエストが伸びないようにする（max_lazy_depth: 1）
        const rekyou_target = {
            ...makeKyou({ data_type: 'rekyou' }),
            typed_rekyou: { attached_kyou: null, target_id: 'nested-target' },
            load_typed_datas: vi.fn().mockResolvedValue([]),
        }
        const deeper = {
            ...makeKyou({ data_type: 'rekyou' }),
            typed_rekyou: { attached_kyou: null, target_id: 'deeper-target' },
            load_typed_datas: vi.fn().mockResolvedValue([]),
        }
        const props = createProps({ height: 91, target_kyou: rekyou_target })
        props.gkill_api.get_kyou.mockImplementation((req: { id: string }) => Promise.resolve({
            errors: [],
            kyou_histories: [req.id === 'nested-target' ? deeper : rekyou_target],
        }))

        useMiReKyouView({ props, emits: noop_emits })

        await flush()

        // MiReKyouの参照先 + 1段だけ。deeper-target までは引かない
        expect(props.gkill_api.get_kyou).toHaveBeenCalledTimes(2)
        expect(props.gkill_api.get_kyou).not.toHaveBeenCalledWith(
            expect.objectContaining({ id: 'deeper-target' }),
        )
    })

    test('参照先が空の本文しか持たないときもフォールバックする', async () => {
        const props = createProps({ target_kyou: makeTargetKyou('') })
        const { target_summary } = useMiReKyouView({ props, emits: noop_emits })

        await flush()

        expect(target_summary.value.length).toBeGreaterThan(0)
    })
})

describe('useMiReKyouView 参照先が見つからないときのエラー', () => {
    test('compact では received_errors をemitしない (行数ぶんスナックバーが出るため)', async () => {
        const emits = vi.fn() as unknown as KyouViewEmits
        useMiReKyouView({ props: createProps({ height: 91, target_kyou: null }), emits })

        await flush()

        expect(emits).not.toHaveBeenCalledWith('received_errors', expect.anything())
    })

    test('compact でなければ従来どおり received_errors をemitする', async () => {
        const emits = vi.fn() as unknown as KyouViewEmits
        useMiReKyouView({ props: createProps({ height: 'unset', target_kyou: null }), emits })

        await flush()

        expect(emits).toHaveBeenCalledWith('received_errors', expect.anything())
    })
})

describe('useMiReKyouView primary_time', () => {
    const start = new Date('2026-08-05T09:00:00+09:00')
    const end = new Date('2026-08-05T12:00:00+09:00')
    const limit = new Date('2026-08-10T23:59:00+09:00')

    // 一覧の行には1行しか入らない。Miと同じ 開始 → 終了 → 制限 の順で先に来るものを選ぶ
    test.each([
        { name: '開始だけ', mirekyou: { estimate_start_time: start }, label: 'MI_START_DATE_TIME_TITLE', time: start },
        { name: '終了だけ', mirekyou: { estimate_end_time: end }, label: 'MI_END_DATE_TIME_TITLE', time: end },
        { name: '制限だけ', mirekyou: { limit_time: limit }, label: 'MI_LIMIT_DATE_TIME_TITLE', time: limit },
        {
            name: '全部あれば開始',
            mirekyou: { estimate_start_time: start, estimate_end_time: end, limit_time: limit },
            label: 'MI_START_DATE_TIME_TITLE',
            time: start,
        },
        {
            name: '開始が無ければ終了',
            mirekyou: { estimate_end_time: end, limit_time: limit },
            label: 'MI_END_DATE_TIME_TITLE',
            time: end,
        },
    ])('$name', ({ mirekyou, label, time }) => {
        const { primary_time } = useMiReKyouView({ props: createProps({ mirekyou }), emits: noop_emits })
        expect(primary_time.value).toEqual({ label_key: label, time: time })
    })

    test('日時が1つも無ければnull', () => {
        const { primary_time } = useMiReKyouView({ props: createProps(), emits: noop_emits })
        expect(primary_time.value).toBeNull()
    })
})

describe('useMiReKyouView その他', () => {
    test('参照先が同じなら引き直さない (仮想スクロールの行使い回し)', async () => {
        const props = createProps()
        const { get_target_kyou } = useMiReKyouView({ props, emits: noop_emits })

        await flush()
        expect(props.gkill_api.get_kyou).toHaveBeenCalledTimes(1)

        await get_target_kyou()
        expect(props.gkill_api.get_kyou).toHaveBeenCalledTimes(1)
    })

    test('ドラッグのペイロードは gkill_mi_re_kyou キーで渡す (板間ドロップが依存)', () => {
        const props = createProps()
        const { on_drag_start } = useMiReKyouView({ props, emits: noop_emits })

        const setData = vi.fn()
        on_drag_start({ dataTransfer: { setData } } as unknown as DragEvent)

        expect(setData).toHaveBeenCalledWith('gkill_mi_re_kyou', expect.any(String))
        expect(JSON.parse(setData.mock.calls[0][1]).target_id).toBe('test-target-id')
    })

    test('タッチデバイスでなければ draggable はpropsに従う', () => {
        const { effective_draggable } = useMiReKyouView({ props: createProps({ draggable: false }), emits: noop_emits })
        expect(effective_draggable.value).toBe(false)
    })
})
