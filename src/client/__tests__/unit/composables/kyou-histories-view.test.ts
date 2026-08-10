/**
 * useKyouHistoriesView（履歴一覧）の検証。
 *
 * 1. 履歴の related_time 付け替えループは、新しく読み直した cloned_kyou_value を見ること。
 *    古い ref (cloned_kyou.value) を見ていたときは初期値が new Kyou()（履歴0件）なので
 *    ループが1度も回らず、一覧の並び順の元になる related_time が更新時刻に揃わなかった。
 * 2. requested_reload_kyou では引き直さない。タグ/テキスト/通知を足しても履歴は増えないので、
 *    引くと無駄な get_kyou が1往復増えるだけになる。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { nextTick, reactive } from 'vue'
import { useKyouHistoriesView } from '@/classes/use-kyou-histories-view'
import type { KyouHistoriesViewProps } from '@/pages/views/kyou-histories-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'

interface FakeHistory {
    id: string
    update_time: Date
    related_time: Date
}

/**
 * Kyou の実クラスは通信を伴うので、useKyouHistoriesView が実際に触る
 * clone() / load_attached_histories() / attached_histories だけを持つ構造フェイクを使う。
 */
function make_kyou(id: string, histories: Array<FakeHistory>) {
    const load_counter = { count: 0 }
    const kyou = {
        id: id,
        // 実物と同じく「履歴は clone 側で読む」。clone は履歴を引き継がない
        attached_histories: new Array<FakeHistory>(),
        clone(): unknown {
            const cloned = {
                id: id,
                attached_histories: new Array<FakeHistory>(),
                async load_attached_histories(): Promise<Array<unknown>> {
                    load_counter.count++
                    cloned.attached_histories = histories.map((history) => ({
                        id: history.id,
                        update_time: history.update_time,
                        related_time: history.related_time,
                    }))
                    return []
                },
                clone(): unknown { return cloned },
            }
            return cloned
        },
        async load_attached_histories(): Promise<Array<unknown>> { return [] },
    }
    return { kyou: kyou as unknown as Kyou, load_counter: load_counter }
}

function make_histories(): Array<FakeHistory> {
    return [
        { id: 'h1', update_time: new Date('2026-03-01T10:00:00+09:00'), related_time: new Date('2020-01-01T00:00:00+09:00') },
        { id: 'h2', update_time: new Date('2026-03-02T10:00:00+09:00'), related_time: new Date('2020-01-01T00:00:00+09:00') },
    ]
}

function create_view(kyou: Kyou) {
    const emitted = new Array<string>()
    const emits = ((event: string) => { emitted.push(event) }) as unknown as KyouViewEmits
    const props = reactive({
        kyou: kyou,
        highlight_targets: [],
        enable_context_menu: false,
        enable_dialog: false,
        application_config: {},
        gkill_api: {},
    }) as unknown as KyouHistoriesViewProps
    const view = useKyouHistoriesView({ props: props, emits: emits })
    return { view: view, props: props, emitted: emitted }
}

async function flush(): Promise<void> {
    for (let i = 0; i < 4; i++) {
        await nextTick()
    }
}

describe('履歴の related_time 付け替え', () => {
    test('読み直した clone の履歴を対象にするので、全件が update_time に揃う', async () => {
        const histories = make_histories()
        const { kyou } = make_kyou('kyou-1', histories)
        const { view } = create_view(kyou)
        await flush()

        const loaded = view.cloned_kyou.value.attached_histories as unknown as Array<FakeHistory>
        expect(loaded, '履歴が読み込まれていない').toHaveLength(2)
        for (let i = 0; i < loaded.length; i++) {
            expect(
                loaded[i].related_time.getTime(),
                '付け替えループが古い ref を見ているので1度も回っていない',
            ).toBe(loaded[i].update_time.getTime())
        }
    })

    test('履歴0件でも例外にならず、空のまま入る', async () => {
        const { kyou } = make_kyou('kyou-1', [])
        const { view } = create_view(kyou)
        await flush()

        expect(view.cloned_kyou.value.attached_histories).toHaveLength(0)
    })
})

describe('props.kyou の差し替え', () => {
    test('別の Kyou が来たら履歴を読み直す', async () => {
        const { kyou } = make_kyou('kyou-1', make_histories())
        const { view, props } = create_view(kyou)
        await flush()
        expect(view.cloned_kyou.value.id).toBe('kyou-1')

        const next = make_kyou('kyou-2', make_histories())
        ;(props as unknown as { kyou: Kyou }).kyou = next.kyou
        await flush()

        expect(next.load_counter.count, 'props.kyou を差し替えても読み直していない').toBe(1)
        expect(view.cloned_kyou.value.id).toBe('kyou-2')
    })
})

describe('中継束の再読込条件', () => {
    test('同じ Kyou の updated_kyou では読み直す', async () => {
        const { kyou, load_counter } = make_kyou('kyou-1', make_histories())
        const { view, emitted } = create_view(kyou)
        await flush()
        const before = load_counter.count

        view.crudRelayHandlers.updated_kyou(kyou)
        await flush()

        expect(load_counter.count - before, '編集しても履歴一覧が増えない').toBe(1)
        expect(emitted, '中継先の親へ updated_kyou を上げていない').toContain('updated_kyou')
    })

    test('同じ Kyou の deleted_kyou でも読み直す', async () => {
        const { kyou, load_counter } = make_kyou('kyou-1', make_histories())
        const { view, emitted } = create_view(kyou)
        await flush()
        const before = load_counter.count

        view.crudRelayHandlers.deleted_kyou(kyou)
        await flush()

        expect(load_counter.count - before).toBe(1)
        expect(emitted).toContain('deleted_kyou')
    })

    // タグ/テキスト/通知の追加では履歴が増えないので、引くと get_kyou が1往復増えるだけ
    test('requested_reload_kyou では読み直さない', async () => {
        const { kyou, load_counter } = make_kyou('kyou-1', make_histories())
        const { view, emitted } = create_view(kyou)
        await flush()
        const before = load_counter.count

        view.crudRelayHandlers.requested_reload_kyou(kyou)
        await flush()

        expect(load_counter.count - before, '履歴が増えないイベントで無駄に引き直している').toBe(0)
        expect(emitted, '中継自体は行うこと').toContain('requested_reload_kyou')
    })

    test('別の Kyou の updated_kyou では読み直さない', async () => {
        const { kyou, load_counter } = make_kyou('kyou-1', make_histories())
        const other = make_kyou('kyou-other', make_histories())
        const { view } = create_view(kyou)
        await flush()
        const before = load_counter.count

        view.crudRelayHandlers.updated_kyou(other.kyou)
        await flush()

        expect(load_counter.count - before, '無関係な Kyou の更新で引き直している').toBe(0)
    })
})
