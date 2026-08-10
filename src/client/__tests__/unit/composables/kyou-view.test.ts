/**
 * useKyouView composable tests.
 * 中身が入る前のKyou（ReKyou/MiReKyouの参照先を取りに行っている間）で
 * ゼロ値の日付を出さないこと、その間を読み込み中として扱うことを検証する。
 */
import { describe, test, expect, vi } from 'vitest'
// use-kyou-view は req_res 経由で GkillAPIRequest に依存する。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させないと class extends が undefined になる。
import '@/classes/api/gkill-api'
import { useKyouView } from '@/classes/use-kyou-view'
import { refresh_kyou } from '@/classes/kyou-reload'
import type { Kyou } from '@/classes/datas/kyou'
import type { KyouViewProps } from '@/pages/views/kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

const noop_emits = (() => { }) as unknown as KyouViewEmits

/**
 * KyouViewに渡されるKyou。
 * 未取得のときは ReKyou/MiReKyou が置く `new Kyou()` と同じく、
 * idが空で日時だけ new Date(0) が入った状態にする。
 */
function createProps(options: {
    loaded: boolean,
    is_typed_data_loaded?: boolean,
    load_typed_datas?: () => Promise<Array<unknown>>,
}): KyouViewProps {
    const time = options.loaded ? new Date(2025, 2, 15, 9, 0, 0) : new Date(0)
    const kyou = {
        id: options.loaded ? 'test-kyou-id' : '',
        rep_name: options.loaded ? 'test-rep' : '',
        data_type: '',
        related_time: time,
        create_time: time,
        update_time: time,
        abort_controller: new AbortController(),
        attached_tags: [],
        attached_texts: [],
        attached_notifications: [],
        attached_timeis_kyou: [],
        is_typed_data_loaded: options.is_typed_data_loaded ?? false,
        load_typed_datas: options.load_typed_datas
            ? vi.fn().mockImplementation(options.load_typed_datas)
            : vi.fn().mockResolvedValue([]),
        load_attached_tags: vi.fn().mockResolvedValue([]),
        load_attached_texts: vi.fn().mockResolvedValue([]),
        load_attached_notifications: vi.fn().mockResolvedValue([]),
        load_attached_timeis: vi.fn().mockResolvedValue([]),
        reload: vi.fn().mockResolvedValue([]),
    }
    return {
        kyou: { ...kyou, clone: () => ({ ...kyou, abort_controller: new AbortController() }) },
        highlight_targets: [],
        height: 180,
        width: 400,
        show_related_time: true,
        show_update_time: true,
        show_rep_name: true,
        enable_context_menu: true,
        enable_dialog: true,
    } as unknown as KyouViewProps
}

describe('useKyouView 未取得のKyouの日時', () => {
    test('取得できるまでは日時を出さない (1970/01/01が見えてしまうため)', () => {
        const { related_time, update_time } = useKyouView({ props: createProps({ loaded: false }), emits: noop_emits })

        expect(related_time.value).toBe('')
        expect(update_time.value).toBe('')
    })

    test('取得できたら日時を出す', () => {
        const { related_time, update_time } = useKyouView({ props: createProps({ loaded: true }), emits: noop_emits })

        expect(related_time.value).toContain('2025/03/15')
        expect(update_time.value).toContain('2025/03/15')
    })
})

async function flush(): Promise<void> {
    for (let i = 0; i < 20; i++) {
        await Promise.resolve()
        await new Promise((resolve) => setTimeout(resolve, 0))
    }
}

describe('useKyouView 読み込み中の判定', () => {
    test('参照先を取りに行っている間(idが空)は読み込み中のまま', async () => {
        const { is_kyou_loading } = useKyouView({ props: createProps({ loaded: false }), emits: noop_emits })

        await flush()

        // 空のKyouはload_typed_datasが何もせず返るので、種別データ側だけでは判定できない
        expect(is_kyou_loading.value).toBe(true)
    })

    test('種別データを読んでいる間は読み込み中、読み終わると解除される', async () => {
        let resolve_load: (value: Array<unknown>) => void = () => { }
        const props = createProps({
            loaded: true,
            load_typed_datas: () => new Promise((resolve) => { resolve_load = resolve }),
        })
        const { is_kyou_loading } = useKyouView({ props, emits: noop_emits })

        expect(is_kyou_loading.value).toBe(true)

        resolve_load([])
        await flush()

        expect(is_kyou_loading.value).toBe(false)
    })

    test('読み込み済みのKyouを複製しただけなら最初から読み込み中にしない', async () => {
        // 一覧の再描画や仮想スクロールの行使い回しでスピナーが出ないようにする。
        // clone()はis_typed_data_loadedを引き継ぐ
        const props = createProps({ loaded: true, is_typed_data_loaded: true })
        const { is_kyou_loading } = useKyouView({ props, emits: noop_emits })

        expect(is_kyou_loading.value).toBe(false)
    })
})

/** ページ側の引き直し(refresh_kyou)を止めたまま走らせるための最小のKyou */
function make_reloadable_kyou(id: string, gate: Promise<void>): Kyou {
    const kyou = {
        id: id,
        is_typed_data_loaded: true,
        abort_controller: new AbortController(),
        async reload(): Promise<Array<never>> {
            await gate
            return []
        },
        async load_all(): Promise<Array<never>> {
            return []
        },
        clone(): unknown {
            return make_reloadable_kyou(id, gate)
        },
    }
    return kyou as unknown as Kyou
}

describe('useKyouView 読み込み中表示', () => {
    test('すぐ終わる読み込みではインジケータを出さない (一覧の明滅防止)', async () => {
        vi.useFakeTimers()
        try {
            let resolve_load: (value: Array<unknown>) => void = () => { }
            const props = createProps({
                loaded: true,
                load_typed_datas: () => new Promise((resolve) => { resolve_load = resolve }),
            })
            const { show_loading_indicator } = useKyouView({ props, emits: noop_emits })

            expect(show_loading_indicator.value).toBe(false)

            resolve_load([])
            await vi.advanceTimersByTimeAsync(0)
            await vi.advanceTimersByTimeAsync(1000)

            expect(show_loading_indicator.value).toBe(false)
        } finally {
            vi.useRealTimers()
        }
    })

    test('待たされたらインジケータを出す', async () => {
        vi.useFakeTimers()
        try {
            const props = createProps({
                loaded: true,
                load_typed_datas: () => new Promise(() => { }),
            })
            const { show_loading_indicator } = useKyouView({ props, emits: noop_emits })

            await vi.advanceTimersByTimeAsync(1000)

            expect(show_loading_indicator.value).toBe(true)
        } finally {
            vi.useRealTimers()
        }
    })
})

// 保存後の引き直しはページ側(reload_kyou)が回すので、このKyouViewからは見えない。
// idで購読して「引き直し中」を出す
describe('useKyouView 引き直し中表示', () => {
    test('引き直しの間だけインジケータを出し、中身は消さない', async () => {
        vi.useFakeTimers()
        try {
            const props = createProps({ loaded: true, is_typed_data_loaded: true })
            const { show_loading_indicator, show_reloading_indicator } = useKyouView({ props, emits: noop_emits })

            expect(show_reloading_indicator.value).toBe(false)

            let open_gate = (): void => { }
            const gate = new Promise<void>((resolve) => { open_gate = () => resolve() })
            const reloading = refresh_kyou(make_reloadable_kyou('test-kyou-id', gate))

            await vi.advanceTimersByTimeAsync(1000)
            expect(show_reloading_indicator.value).toBe(true)
            // 中身を差し替える側のフラグは立てない（消すと一覧の行がちらつく）
            expect(show_loading_indicator.value).toBe(false)

            open_gate()
            await reloading
            await vi.advanceTimersByTimeAsync(1000)

            expect(show_reloading_indicator.value).toBe(false)
        } finally {
            vi.useRealTimers()
        }
    })

    test('別のKyouの引き直しでは出さない', async () => {
        vi.useFakeTimers()
        try {
            const props = createProps({ loaded: true, is_typed_data_loaded: true })
            const { show_reloading_indicator } = useKyouView({ props, emits: noop_emits })

            let open_gate = (): void => { }
            const gate = new Promise<void>((resolve) => { open_gate = () => resolve() })
            const reloading = refresh_kyou(make_reloadable_kyou('other-kyou-id', gate))

            await vi.advanceTimersByTimeAsync(1000)
            expect(show_reloading_indicator.value).toBe(false)

            open_gate()
            await reloading
        } finally {
            vi.useRealTimers()
        }
    })
})
