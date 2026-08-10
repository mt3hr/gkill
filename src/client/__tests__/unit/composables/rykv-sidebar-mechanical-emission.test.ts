/**
 * サイドバーの機械的な updated_query 発火の遮断の検証（rykv側・mi側）。
 *
 * 列クリック等で find_kyou_query を差し替えると、子ビューのprops同期の残響が
 * emits_current_query へ機械的に届く（1tickより遅れて来るのでタイミングでは抑止できない）。
 * これがそのまま updated_query になると、ホットリロード既定ONの環境では
 * 「検索中の列をクリック→飛行中の検索がabortされ最初からやり直し」となり、
 * 重いデータでは実質ハングになる（2026-08-10 実測で確定）。
 *
 * 固定する不変条件:
 * - 再生成結果が同期済みクエリと同値なら emits_current_query は emit しない（値比較ガード）
 * - generate_query は同期済みクエリに対して恒等（とくに include_*_mi をtrue固定で
 *   ドリフトさせない。ドリフトすると値比較ガードも search() 側の deep_equals 安全網も破れる）
 * - 旧形式JSON（use_*フラグ入り）を parse_find_kyou_query で読んだ列でも恒等が成り立ち、
 *   use_* キーがインスタンスへ復活しない（復活すると deep_equals のキー数比較が永久に破れる）
 * - ユーザーの実編集（ウィジェット値の変化）は従来どおり emit される
 *
 * 子ビューはテンプレートrefがnullのままなので、generate_query の子由来ブロックはスキップされる。
 * 実編集の再現はrefへfakeを差して行う。
 */
import { describe, expect, test } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { nextTick, reactive } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { deep_equals } from '@/classes/deep-equals'
import { useRykvQueryEditorSideBar } from '@/classes/use-rykv-query-editor-side-bar'
import { useMiQueryEditorSidebar } from '@/classes/use-mi-query-editor-sidebar'
import type { RykvQueryEditorSidebarProps } from '@/pages/views/rykv-query-editor-sidebar-props'
import type { RykvQueryEditorSidebarEmits } from '@/pages/views/rykv-query-editor-sidebar-emits'
import type { MiQueryEditorSidebarProps } from '@/pages/views/mi-query-editor-sidebar-props'
import type { MiQueryEditorSidebarEmits } from '@/pages/views/mi-query-editor-sidebar-emits'

function collect_emits(): { emitted: Array<{ event: string, payload: unknown }>, emits: (event: string, ...args: Array<unknown>) => void } {
    const emitted: Array<{ event: string, payload: unknown }> = []
    const emits = (event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, payload: args[0] })
    }
    return { emitted, emits }
}

function createRykvView() {
    const { emitted, emits } = collect_emits()
    const props = reactive({
        application_config: new ApplicationConfig(),
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        find_kyou_query: new FindKyouQuery(),
        inited: false,
        app_title_bar_height: 50,
        app_content_height: 800,
        app_content_width: 1200,
    }) as unknown as RykvQueryEditorSidebarProps & { find_kyou_query: FindKyouQuery }
    const view = useRykvQueryEditorSideBar({ props, emits: emits as unknown as RykvQueryEditorSidebarEmits })
    return { view, props, emitted }
}

function createMiView() {
    const { emitted, emits } = collect_emits()
    const props = reactive({
        application_config: new ApplicationConfig(),
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        find_kyou_query: new FindKyouQuery(),
        inited: false,
        app_title_bar_height: 50,
        app_content_height: 800,
        app_content_width: 1200,
    }) as unknown as MiQueryEditorSidebarProps & { find_kyou_query: FindKyouQuery }
    const view = useMiQueryEditorSidebar({ props, emits: emits as unknown as MiQueryEditorSidebarEmits })
    return { view, props, emitted }
}

function updated_query_events(emitted: Array<{ event: string, payload: unknown }>): Array<{ event: string, payload: unknown }> {
    return emitted.filter((emit) => emit.event === 'updated_query')
}

describe('rykvサイドバーの機械的updated_query遮断', () => {
    test('既定列クエリを同期した直後の emits_current_query は emit しない', async () => {
        const { view, props, emitted } = createRykvView()
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        props.find_kyou_query = column_query
        await nextTick()

        view.emits_current_query()

        expect(updated_query_events(emitted), '機械的な発火が検索(updated_query)に化けてはいけない').toHaveLength(0)
    })

    test('generate_query は自身の出力を同期しても恒等（固定点）', async () => {
        const { view, props, emitted } = createRykvView()
        const generated = view.generate_query('column-fixpoint')
        props.find_kyou_query = generated
        await nextTick()

        expect(deep_equals(view.generate_query('column-fixpoint'), generated), '再生成が同期済みクエリからドリフトしている').toBe(true)

        view.emits_current_query()
        expect(updated_query_events(emitted)).toHaveLength(0)
    })

    test('include_*_mi はtrue固定せず列クエリから引き継ぐ（ドリフト恒等性）', async () => {
        const { view, props } = createRykvView()
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        // FindKyouQueryコンストラクタ既定: create=true, 他4つ=false
        expect(column_query.include_create_mi).toBe(true)
        expect(column_query.include_check_mi).toBe(false)
        props.find_kyou_query = column_query
        await nextTick()

        const regenerated = view.generate_query('column-1')
        expect(regenerated.include_create_mi).toBe(true)
        expect(regenerated.include_check_mi).toBe(false)
        expect(regenerated.include_limit_mi).toBe(false)
        expect(regenerated.include_start_mi).toBe(false)
        expect(regenerated.include_end_mi).toBe(false)
        expect(deep_equals(regenerated, column_query), '無編集の再生成は列クエリと同値でなければならない').toBe(true)
    })

    test('旧形式JSON(use_*入り)を読み込んだ列でも恒等が成り立ち、use_*キーは復活しない', async () => {
        const { view, props, emitted } = createRykvView()

        // 列のchildless再生成と同値になる旧形式JSONを作る:
        // 新形式の生成結果をJSON化し、旧世代ビルドが書いていた use_* フラグ群と
        // update_time を混ぜる（値は新形式へ正規化すると元の null/[] に一致する組み合わせ）
        const base = JSON.parse(JSON.stringify(view.generate_query('legacy-col'))) as Record<string, unknown>
        const legacy_json = {
            ...base,
            use_words: false,
            use_timeis: false,
            use_timeis_tags: false,
            use_tags: true,
            use_reps: true,
            use_rep_types: false,
            use_map: false,
            use_calendar: false,
            use_plaing: false,
            use_period_of_time: false,
            use_mi_board_name: false,
            use_mi_sort_type: false,
            use_mi_check_state: false,
            use_include_id: false,
            use_update_time: false,
            update_time: '2020-01-01T00:00:00.000Z',
        }
        const parsed = FindKyouQuery.parse_find_kyou_query(legacy_json)

        // 旧キーがインスタンスへ混入しない（混入すると deep_equals のキー数比較が永久に破れる）
        const parsed_keys = Object.keys(parsed)
        expect(parsed_keys.filter((key) => key.startsWith('use_'))).toEqual([])
        expect(parsed_keys).not.toContain('update_time')

        props.find_kyou_query = parsed
        await nextTick()

        // 同期済みクエリに対して generate_query は恒等（use_* が復活すればここで破れる）
        const regenerated = view.generate_query('legacy-col')
        expect(Object.keys(regenerated).filter((key) => key.startsWith('use_'))).toEqual([])
        expect(deep_equals(regenerated, parsed), '旧形式由来の列で再生成がドリフトしている').toBe(true)

        view.emits_current_query()
        expect(updated_query_events(emitted), '旧形式由来の列で deep_equals ガードが破れている').toHaveLength(0)
    })

    test('ユーザーの実編集は従来どおり emit される', async () => {
        const { view, props, emitted } = createRykvView()
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        props.find_kyou_query = column_query
        await nextTick()

        // キーワード欄をONにして入力した状態のfake
        view.keyword_query.value = {
            get_use_words: () => true,
            get_use_word_and_search: () => false,
            get_keywords: () => '写真',
        } as unknown as typeof view.keyword_query.value

        view.emits_current_query()

        const updated = updated_query_events(emitted)
        expect(updated, '実編集の emit まで捨ててはいけない').toHaveLength(1)
        const payload = updated[0].payload as FindKyouQuery
        expect(payload.words, '有効時は未パースプレースホルダの[]').toEqual([])
        expect(payload.not_words).toEqual([])
        expect(payload.keywords).toBe('写真')
        expect(payload.query_id, 'query_id は列のものを維持する').toBe('column-1')
    })
})

describe('miサイドバーの機械的updated_query遮断', () => {
    test('generate_query の出力を同期した直後の emits_current_query は emit しない（固定点）', async () => {
        const { view, props, emitted } = createMiView()
        const generated = view.generate_query('mi-column-1')
        props.find_kyou_query = generated
        await nextTick()

        expect(deep_equals(view.generate_query('mi-column-1'), generated), '再生成が同期済みクエリからドリフトしている').toBe(true)

        view.emits_current_query()
        expect(updated_query_events(emitted), '機械的な発火が検索(updated_query)に化けてはいけない').toHaveLength(0)
    })

    test('ユーザーの実編集は従来どおり emit される', async () => {
        const { view, props, emitted } = createMiView()
        const generated = view.generate_query('mi-column-1')
        props.find_kyou_query = generated
        await nextTick()

        view.keyword_query.value = {
            get_use_words: () => true,
            get_use_word_and_search: () => true,
            get_keywords: () => '牛乳',
        } as unknown as typeof view.keyword_query.value

        view.emits_current_query()

        const updated = updated_query_events(emitted)
        expect(updated, '実編集の emit まで捨ててはいけない').toHaveLength(1)
        const payload = updated[0].payload as FindKyouQuery
        expect(payload.words, '有効時は未パースプレースホルダの[]').toEqual([])
        expect(payload.words_and).toBe(true)
        expect(payload.keywords).toBe('牛乳')
        expect(payload.query_id).toBe('mi-column-1')
    })
})
