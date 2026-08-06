/**
 * useReKyouView composable tests.
 * MiReKyou側と揃えた挙動（空配列ガード・再取得抑止・行でのエラー抑制）を検証する。
 */
import { describe, test, expect, vi } from 'vitest'
// use-re-kyou-view は req_res 経由で GkillAPIRequest に依存する。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させないと class extends が undefined になる。
import '@/classes/api/gkill-api'
import { useReKyouView } from '@/classes/use-re-kyou-view'
import type { ReKyouViewProps } from '@/pages/views/re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { makeKyou, makeKyouWithKmemo, makeReKyou } from '../../helpers/factory'

const noop_emits = (() => { }) as unknown as KyouViewEmits

function createProps(options: {
    height?: number | string,
    target_kyou?: unknown,
    errors?: Array<unknown>,
    rekyou?: Record<string, unknown>,
} = {}): ReKyouViewProps & { gkill_api: { get_kyou: ReturnType<typeof vi.fn> } } {
    const kyou_histories = options.target_kyou === null ? [] : [options.target_kyou ?? makeKyouWithKmemo('参照先のメモ')]
    const gkill_api = {
        get_kyou: vi.fn().mockResolvedValue({ errors: options.errors ?? [], kyou_histories: kyou_histories }),
    }
    return {
        kyou: makeKyou({ data_type: 'rekyou' }),
        rekyou: makeReKyou(options.rekyou ?? {}),
        gkill_api: gkill_api,
        application_config: { device: 'test-device', user_id: 'admin' },
        highlight_targets: [],
        height: options.height ?? 180,
        width: 400,
        enable_context_menu: true,
        enable_dialog: true,
    } as unknown as ReKyouViewProps & { gkill_api: { get_kyou: ReturnType<typeof vi.fn> } }
}

async function flush(): Promise<void> {
    for (let i = 0; i < 20; i++) {
        await Promise.resolve()
        await new Promise((resolve) => setTimeout(resolve, 0))
    }
}

describe('useReKyouView is_row', () => {
    // 実際に渡ってくる高さは 91 (タスクボードの行) / 180 (rykvの一覧) / 'unset' (詳細・ダイアログ)
    test.each([
        { height: 91 as number | string, expected: true, name: 'タスクボードの行高 91 は行' },
        { height: '91px', expected: true, name: '単位付きの文字列でも行' },
        { height: 180, expected: false, name: 'rykvの一覧 180 は行として扱わない' },
        { height: 'unset', expected: false, name: "'unset' は行ではない" },
    ])('$name', ({ height, expected }) => {
        const { is_row } = useReKyouView({ props: createProps({ height }), emits: noop_emits })
        expect(is_row.value).toBe(expected)
    })
})

describe('useReKyouView 参照先の取得', () => {
    test('参照先を取得して target_kyou に入れる', async () => {
        const target = makeKyouWithKmemo('リポスト元のメモ')
        const props = createProps({ target_kyou: target })
        const { target_kyou } = useReKyouView({ props, emits: noop_emits })

        await flush()

        expect(props.gkill_api.get_kyou).toHaveBeenCalledWith(
            expect.objectContaining({ id: 'test-target-id' }),
        )
        // refに入るとreactive proxyになるので同一性ではなく中身で見る
        expect(target_kyou.value.typed_kmemo?.content).toBe('リポスト元のメモ')
    })

    test('参照先が見つからなくても undefined を入れない', async () => {
        // 以前は res.kyou_histories[0] を無条件代入していたので、
        // 参照先を消すと target_kyou が undefined になっていた
        const props = createProps({ target_kyou: null })
        const { target_kyou } = useReKyouView({ props, emits: noop_emits })

        await flush()

        expect(target_kyou.value).toBeDefined()
        expect(target_kyou.value).not.toBeUndefined()
    })

    test('参照先が同じなら引き直さない (仮想スクロールの行使い回し)', async () => {
        const props = createProps()
        const { get_target_kyou } = useReKyouView({ props, emits: noop_emits })

        await flush()
        expect(props.gkill_api.get_kyou).toHaveBeenCalledTimes(1)

        await get_target_kyou()
        expect(props.gkill_api.get_kyou).toHaveBeenCalledTimes(1)
    })
})

describe('useReKyouView 参照先が見つからないときのエラー', () => {
    test('行では received_errors をemitしない (行数ぶんスナックバーが出るため)', async () => {
        const emits = vi.fn() as unknown as KyouViewEmits
        useReKyouView({ props: createProps({ height: 91, target_kyou: null }), emits })

        await flush()

        expect(emits).not.toHaveBeenCalledWith('received_errors', expect.anything())
    })

    test('詳細では received_errors をemitする (消失に気づけなくなるため)', async () => {
        const emits = vi.fn() as unknown as KyouViewEmits
        useReKyouView({ props: createProps({ height: 'unset', target_kyou: null }), emits })

        await flush()

        expect(emits).toHaveBeenCalledWith('received_errors', expect.anything())
    })

    test('APIがエラーを返したときも行では黙る', async () => {
        const emits = vi.fn() as unknown as KyouViewEmits
        useReKyouView({ props: createProps({ height: 91, errors: [{ error_message: 'ng' }] }), emits })

        await flush()

        expect(emits).not.toHaveBeenCalledWith('received_errors', expect.anything())
    })
})

describe('useReKyouView 参照先なしの終端状態', () => {
    // 終端状態が無いと、参照先が消えているReKyouでKyouViewが読み込み中表示のまま止まる
    test('参照先が空なら is_target_not_found が立つ', async () => {
        const props = createProps({ target_kyou: null })
        const { is_target_not_found } = useReKyouView({ props, emits: noop_emits })

        await flush()

        expect(is_target_not_found.value).toBe(true)
    })

    test('APIがエラーを返したときも is_target_not_found が立つ', async () => {
        const props = createProps({ errors: [{ error_message: 'ng' }] })
        const { is_target_not_found } = useReKyouView({ props, emits: noop_emits })

        await flush()

        expect(is_target_not_found.value).toBe(true)
    })

    test('取得できたら is_target_not_found は立たない', async () => {
        const props = createProps()
        const { is_target_not_found } = useReKyouView({ props, emits: noop_emits })

        await flush()

        expect(is_target_not_found.value).toBe(false)
    })

    test('target_idが空ならリクエストせず即 is_target_not_found が立つ', async () => {
        // loaded_target_idの初期値が'' なので、使い回しガードに引っかかって
        // リクエストが飛ばず、以前はプレースホルダのまま止まっていた
        const props = createProps({ rekyou: { target_id: '' } })
        const { is_target_not_found } = useReKyouView({ props, emits: noop_emits })

        await flush()

        expect(props.gkill_api.get_kyou).not.toHaveBeenCalled()
        expect(is_target_not_found.value).toBe(true)
    })
})
