/**
 * useRyuuView（流の定義編集）の検証。
 *
 * 1. apply() は編集結果を親へ渡すだけにする。以前は v-model で受け取った
 *    ApplicationConfig のプロパティを直接書き換えていたが、それは設定画面の clone そのもので、
 *    設定画面でキャンセルしても流の編集だけ戻らなくなっていた。
 * 2. from_json はレガシーな「素のクエリ配列」を1定義に包む。ここを落とすと
 *    旧ビルドが保存した流の定義が全部消える。
 * 3. 並べ替えは remove 後の index 補正が要る。補正を忘れると from < target のときだけ
 *    1つずれた位置に入る（ドラッグの上下判定と食い違う）。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import { useRyuuView, type RyuuDefinition } from '@/classes/use-ryuu-view'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import AndPredicate from '@/classes/dnote/dnote-predicate/and-predicate'
import { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type RyuuViewProps from '@/pages/views/ryuu-view-props'
import type RyuuViewEmits from '@/pages/views/ryuu-view-emits'

interface EmittedEvent {
    event: string
    args: Array<unknown>
}

function make_query(id: string): RelatedKyouQuery {
    const query = new RelatedKyouQuery()
    query.id = id
    query.title = id
    query.predicate = new AndPredicate([])
    query.related_time_match_type = RelatedTimeMatchType.near_related_time
    query.find_duration_hour = 1
    return query
}

// ライフサイクル（onUnmounted）を通すためにコンポーネントとしてマウントする
function mount_view(options: { ryuu_json_data: unknown, editable?: boolean }) {
    const emitted = new Array<EmittedEvent>()
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, args: args })
    }) as unknown as RyuuViewEmits
    const application_config = reactive({ ryuu_json_data: options.ryuu_json_data })
    const props = reactive({
        gkill_api: {},
        application_config: application_config,
        find_kyou_query_default: new FindKyouQuery(),
        target_kyou: null,
        matched_kyous: null,
        editable: options.editable ?? true,
    }) as unknown as RyuuViewProps

    let view: ReturnType<typeof useRyuuView> | null = null
    const Host = defineComponent({
        setup() {
            view = useRyuuView({ props: props, emits: emits })
            return () => h('div')
        },
    })
    const app = createApp(Host)
    app.mount(document.createElement('div'))
    return { app: app, view: view!, props: props, emitted: emitted, application_config: application_config }
}

// 初期化は nextTick の入れ子（load_from_application_config → from_json）なので数tick回す
async function flush(): Promise<void> {
    for (let i = 0; i < 5; i++) {
        await nextTick()
    }
}

function ids_of(definitions: Array<RyuuDefinition>, index: number): Array<string> {
    return definitions[index].queries.map((query) => query.id)
}

describe('apply', () => {
    test('編集結果を emit するだけで、props.application_config.ryuu_json_data を書き換えない', async () => {
        const original = [{ name: 'def-1', queries: [] }]
        const { app, view, emitted, application_config } = mount_view({ ryuu_json_data: original })
        await flush()

        view.add_definition()
        await view.apply()
        await flush()

        const applied = emitted.filter((entry) => entry.event === 'requested_apply_ryuu_struct')
        expect(applied, '編集結果を親へ渡していない').toHaveLength(1)
        expect((applied[0].args[0] as Array<unknown>), '追加した定義が渡っていない').toHaveLength(2)
        // reactive() 越しなので同一性ではなく内容で見る。
        // 定義を1つ足したのに props 側が2件になっていたら直接書き換えている
        expect(
            application_config.ryuu_json_data,
            'props を直接書き換えている（設定画面のキャンセルが効かなくなる）',
        ).toStrictEqual([{ name: 'def-1', queries: [] }])
        expect(original, 'props 側の生の配列まで書き換えている').toHaveLength(1)
        app.unmount()
    })

    test('適用したらダイアログを閉じるよう頼む', async () => {
        const { app, view, emitted } = mount_view({ ryuu_json_data: [] })
        await flush()

        await view.apply()
        await flush()

        expect(emitted.map((entry) => entry.event)).toContain('requested_close_dialog')
        app.unmount()
    })

    test('emit する形は「定義名 + クエリ配列」の配列', async () => {
        const { app, view, emitted } = mount_view({ ryuu_json_data: [] })
        await flush()
        view.ryuu_definitions.value = [{ name: 'def-1', queries: [make_query('a')] }]

        await view.apply()
        await flush()

        const applied = emitted.filter((entry) => entry.event === 'requested_apply_ryuu_struct')[0]
        const json = applied.args[0] as Array<Record<string, unknown>>
        expect(json).toHaveLength(1)
        expect(json[0].name).toBe('def-1')
        expect((json[0].queries as Array<Record<string, unknown>>)[0].id).toBe('a')
        app.unmount()
    })
})

describe('from_json', () => {
    test('レガシーな素のクエリ配列は1定義に包まれる', async () => {
        const legacy = [
            { id: 'a', title: 'A', prefix: '', suffix: '', predicate: { logic: 'AND', type: 'AndPredicate', predicates: [] }, related_time_match_type: RelatedTimeMatchType.near_related_time, find_kyou_query: null, find_duration_hour: 1 },
            { id: 'b', title: 'B', prefix: '', suffix: '', predicate: { logic: 'AND', type: 'AndPredicate', predicates: [] }, related_time_match_type: RelatedTimeMatchType.near_related_time, find_kyou_query: null, find_duration_hour: 1 },
        ]
        const { app, view } = mount_view({ ryuu_json_data: legacy })
        await flush()

        expect(view.ryuu_definitions.value, 'レガシー形式が1定義に包まれていない').toHaveLength(1)
        expect(ids_of(view.ryuu_definitions.value, 0), 'レガシー形式のクエリが読めていない').toEqual(['a', 'b'])
        app.unmount()
    })

    test('新形式（name + queries）はそのまま読む', async () => {
        const struct = [
            { name: 'def-1', queries: [{ id: 'a', title: 'A', prefix: '', suffix: '', predicate: { logic: 'AND', type: 'AndPredicate', predicates: [] }, related_time_match_type: RelatedTimeMatchType.near_related_time, find_kyou_query: null, find_duration_hour: 1 }] },
            { name: 'def-2', queries: [] },
        ]
        const { app, view } = mount_view({ ryuu_json_data: struct })
        await flush()

        expect(view.ryuu_definitions.value).toHaveLength(2)
        expect(view.ryuu_definitions.value[0].name).toBe('def-1')
        expect(ids_of(view.ryuu_definitions.value, 0)).toEqual(['a'])
        expect(view.ryuu_definitions.value[1].name).toBe('def-2')
        app.unmount()
    })

    test.each([
        { name: 'null', value: null },
        { name: '文字列', value: 'not-a-struct' },
        { name: 'オブジェクト', value: { name: 'x' } },
        { name: '空配列', value: [] },
    ])('$name のような不正値では空定義が1件になる', async ({ value }) => {
        const { app, view } = mount_view({ ryuu_json_data: value })
        await flush()

        expect(view.ryuu_definitions.value, '不正値で定義が消えている').toHaveLength(1)
        expect(view.ryuu_definitions.value[0].queries).toHaveLength(0)
        expect(view.current_definition_index.value).toBe(0)
        app.unmount()
    })
})

describe('delete_current_definition', () => {
    test('最後の1件は消さない（定義が0件だと編集不能になる）', async () => {
        const { app, view } = mount_view({ ryuu_json_data: [] })
        await flush()
        expect(view.ryuu_definitions.value).toHaveLength(1)

        view.delete_current_definition()
        await flush()

        expect(view.ryuu_definitions.value, '最後の1件を消している').toHaveLength(1)
        app.unmount()
    })

    test('2件以上なら消して、はみ出した選択位置を末尾へ補正する', async () => {
        const { app, view } = mount_view({ ryuu_json_data: [] })
        await flush()
        view.ryuu_definitions.value = [
            { name: 'def-1', queries: [] },
            { name: 'def-2', queries: [] },
        ]
        view.current_definition_index.value = 1

        view.delete_current_definition()
        await flush()

        expect(view.ryuu_definitions.value).toHaveLength(1)
        expect(view.ryuu_definitions.value[0].name).toBe('def-1')
        expect(view.current_definition_index.value, '選択位置が配列の外を指している').toBe(0)
        app.unmount()
    })
})

describe('handle_move_related_kyou_query の index 補正', () => {
    // [a, b, c] から1件を抜いて挿し直す。from < target のときだけ remove で target が1つ前へずれる
    test.each([
        { name: '前から後ろへ down', src: 'a', target: 'c', drop_type: 'down' as const, expected: ['b', 'c', 'a'] },
        { name: '前から後ろへ up', src: 'a', target: 'c', drop_type: 'up' as const, expected: ['b', 'a', 'c'] },
        { name: '後ろから前へ up', src: 'c', target: 'a', drop_type: 'up' as const, expected: ['c', 'a', 'b'] },
        { name: '後ろから前へ down', src: 'c', target: 'a', drop_type: 'down' as const, expected: ['a', 'c', 'b'] },
        { name: '隣へ down', src: 'a', target: 'b', drop_type: 'down' as const, expected: ['b', 'a', 'c'] },
    ])('$name → $expected', async ({ src, target, drop_type, expected }) => {
        const { app, view } = mount_view({ ryuu_json_data: [] })
        await flush()
        view.ryuu_definitions.value = [{ name: 'def-1', queries: [make_query('a'), make_query('b'), make_query('c')] }]

        view.onRequestedMoveRelatedKyouQuery(src, target, drop_type)
        await flush()

        expect(ids_of(view.ryuu_definitions.value, 0)).toEqual(expected)
        app.unmount()
    })

    test('同じ要素へのドロップと未知のidでは並びが変わらない', async () => {
        const { app, view } = mount_view({ ryuu_json_data: [] })
        await flush()
        view.ryuu_definitions.value = [{ name: 'def-1', queries: [make_query('a'), make_query('b')] }]

        view.onRequestedMoveRelatedKyouQuery('a', 'a', 'down')
        view.onRequestedMoveRelatedKyouQuery('a', 'unknown', 'down')
        view.onRequestedMoveRelatedKyouQuery('unknown', 'b', 'down')
        await flush()

        expect(ids_of(view.ryuu_definitions.value, 0)).toEqual(['a', 'b'])
        app.unmount()
    })

    test('editable=false のときは並べ替えない', async () => {
        const { app, view } = mount_view({ ryuu_json_data: [], editable: false })
        await flush()
        view.ryuu_definitions.value = [{ name: 'def-1', queries: [make_query('a'), make_query('b')] }]

        view.onRequestedMoveRelatedKyouQuery('a', 'b', 'down')
        await flush()

        expect(ids_of(view.ryuu_definitions.value, 0), '閲覧中に並べ替えが効いている').toEqual(['a', 'b'])
        app.unmount()
    })
})
