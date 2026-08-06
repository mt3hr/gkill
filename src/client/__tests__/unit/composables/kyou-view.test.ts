/**
 * useKyouView composable tests.
 * 中身が入る前のKyou（ReKyou/MiReKyouの参照先を取りに行っている間）で
 * ゼロ値の日付を出さないことを検証する。
 */
import { describe, test, expect, vi } from 'vitest'
// use-kyou-view は req_res 経由で GkillAPIRequest に依存する。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させないと class extends が undefined になる。
import '@/classes/api/gkill-api'
import { useKyouView } from '@/classes/use-kyou-view'
import type { KyouViewProps } from '@/pages/views/kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

const noop_emits = (() => { }) as unknown as KyouViewEmits

/**
 * KyouViewに渡されるKyou。
 * 未取得のときは ReKyou/MiReKyou が置く `new Kyou()` と同じく、
 * idが空で日時だけ new Date(0) が入った状態にする。
 */
function createProps(options: { loaded: boolean }): KyouViewProps {
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
        load_typed_datas: vi.fn().mockResolvedValue([]),
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
