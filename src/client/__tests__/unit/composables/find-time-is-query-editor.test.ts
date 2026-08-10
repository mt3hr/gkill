/**
 * plaing検索（実行中TimeIs）のカスタム検索条件エディタの検証。
 *
 * このエディタ（use-find-time-is-query-editor-view）が書き込むフィールドと、
 * 適用側（generate-plaing-timeis-query）がカスタム条件から拾うフィールドは
 * 1:1でなければならない。
 *   - エディタ側だけ増える → 「設定したのに検索に効かない」フィールドが生まれる
 *   - 適用側だけ増える     → 保存JSONに残った古い値が黙って復活する
 * 期待集合をこのファイルで宣言し、両側から突き合わせて固定する。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { nextTick, reactive, type Ref } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { generate_plaing_timeis_query } from '@/classes/api/find_query/generate-plaing-timeis-query'
import { deep_equals } from '@/classes/deep-equals'
import { useFindTimeIsQueryEditorView } from '@/classes/use-find-time-is-query-editor-view'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { FindTimeIsQueryEditorViewProps } from '@/pages/views/find-time-is-query-editor-view-props'
import type { FindTimeIsQueryEditorViewEmits } from '@/pages/views/find-time-is-query-editor-view-emits'

// ── 両側で共有する期待集合 ──

// generate-plaing-timeis-query.ts がカスタム条件から拾う（コピーする）フィールド。
// エディタが書けるのはこの6つだけ
const copied_fields = ['keywords', 'not_words', 'tags', 'tags_and', 'words', 'words_and']

// エディタが実行中検索として固定で立てるフィールド（ユーザには選ばせない）
const editor_fixed_fields = ['hide_tags', 'rep_types']

// 適用側が呼び出し元の意図で常に上書きするフィールド。
// reps は「エディタから記録保管場所が消えた」ため明示的に切られる（null）
const applier_owned_fields = ['hide_tags', 'plaing_time', 'rep_types', 'reps']

// ── 小道具 ──

// ApplicationConfig の実物は循環importを引き込むので、触るフィールドだけの構造フェイクを使う
function make_fake_application_config(): Record<string, unknown> {
    return {
        rep_struct: {
            rep_name: 'root',
            children: [{ rep_name: 'timeis_dev_202601', children: null }],
        },
        tag_struct: {
            tag_name: '',
            is_force_hide: false,
            check_when_inited: false,
            children: [
                { tag_name: '非表示タグ', is_force_hide: true, check_when_inited: false, children: null },
            ],
        },
        for_share_kyou: false,
        plaing_timeis_json_data: null,
    }
}

function make_props(application_config: Record<string, unknown>) {
    return reactive({
        application_config: application_config,
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        inited: true,
        find_kyou_query: new FindKyouQuery(),
    }) as unknown as FindTimeIsQueryEditorViewProps & { find_kyou_query: FindKyouQuery }
}

function collect_emits(emitted: Array<string>): FindTimeIsQueryEditorViewEmits {
    return ((event: string) => { emitted.push(event) }) as unknown as FindTimeIsQueryEditorViewEmits
}

function create_view(application_config?: Record<string, unknown>) {
    const config = application_config ?? make_fake_application_config()
    const props = make_props(config)
    const emitted: Array<string> = []
    const view = useFindTimeIsQueryEditorView({ props: props, emits: collect_emits(emitted) })
    return { config, props, view, emitted }
}

// 子コンポーネント（keyword-query / tag-query）の defineExpose 相当を差し込む。
// エディタが「子から読んで書く」経路を全部通すため、既定と違う値にしておく
function attach_child_stubs(view: ReturnType<typeof useFindTimeIsQueryEditorView>): void {
    const keyword_ref = view.keyword_query as unknown as Ref<unknown>
    keyword_ref.value = {
        get_use_words: () => true,
        get_use_word_and_search: () => true,
        get_keywords: () => '写真 -除外',
    }
    const tag_ref = view.tag_query as unknown as Ref<unknown>
    tag_ref.value = {
        get_tags: () => ['旅行'],
        get_is_and_search: () => true,
        update_check: () => Promise.resolve(),
    }
}

// new FindKyouQuery() から実際に動いたフィールド名を返す
function changed_field_names(query: FindKyouQuery): Array<string> {
    const base = new FindKyouQuery() as unknown as Record<string, unknown>
    const target = query as unknown as Record<string, unknown>
    return Object.keys(base).filter((name) => !deep_equals(target[name], base[name])).sort()
}

async function flush(times = 4): Promise<void> {
    for (let i = 0; i < times; i++) {
        await nextTick()
    }
}

describe('エディタが書き込むフィールド集合', () => {
    test('generate_query が動かすのはコピー対象6フィールドと実行中検索の固定分だけ', () => {
        const { view } = create_view()
        attach_child_stubs(view)
        view.use_tag_filter.value = true

        expect(
            changed_field_names(view.generate_query()),
            'エディタの書き込み先が増減している。generate-plaing-timeis-query のコピーリストと必ず両方そろえること',
        ).toEqual([...copied_fields, ...editor_fixed_fields].sort())
    })

    test('記録タイプは子UIの有無によらず timeis 固定', () => {
        const { view } = create_view()
        // 子がまだ生えていない（loading中の）状態でも記録タイプは立つ
        expect(view.generate_query().rep_types).toEqual(['timeis'])

        attach_child_stubs(view)
        expect(view.generate_query().rep_types).toEqual(['timeis'])
    })

    test('query_id を渡したときだけ採番し、渡さなければ空のまま', () => {
        const { view } = create_view()
        expect(view.generate_query().query_id).toBe('')
        expect(view.generate_query('q-1').query_id).toBe('q-1')
    })
})

describe('適用側が拾うフィールド集合', () => {
    test('generate_plaing_timeis_query はコピー対象6フィールドしか保存条件から採らない', () => {
        const config = make_fake_application_config()

        const saved = new FindKyouQuery()
        // コピーされるはずのもの
        saved.keywords = '写真 -除外'
        saved.words = []
        saved.not_words = []
        saved.words_and = true
        saved.tags = ['旅行']
        saved.tags_and = true
        // コピーされないはずのもの（記録保管場所を選べた頃の値も含む）
        saved.query_id = 'saved-query-id'
        saved.reps = ['古いrep']
        saved.rep_types = ['kmemo']
        saved.hide_tags = ['保存時点の非表示タグ']
        saved.timeis_words = ['作業']
        saved.timeis_tags = ['作業タグ']
        saved.calendar_start_date = new Date(2026, 6, 1)
        saved.calendar_end_date = new Date(2026, 6, 3)
        saved.map_latitude = 35.65
        saved.map_longitude = 139.74
        saved.map_radius = 500
        saved.period_of_time_week_of_days = [1, 2]
        saved.mi_board_name = '板A'
        saved.is_image_only = true
        config.plaing_timeis_json_data = { plaing_timeis_find_kyou_query: JSON.parse(JSON.stringify(saved)) }

        const applied = generate_plaing_timeis_query(config as unknown as ApplicationConfig, new Date(2026, 6, 2, 12, 0))

        expect(
            changed_field_names(applied),
            'コピー対象が増減している。use-find-time-is-query-editor-view の編集面と必ず両方そろえること',
        ).toEqual([...copied_fields, ...applier_owned_fields].sort())

        // 拾ったものは中身も保存値どおり（words/not_words は keywords からの導出）
        expect(applied.keywords).toBe('写真 -除外')
        expect(applied.words).toEqual(['写真'])
        expect(applied.not_words).toEqual(['除外'])
        expect(applied.words_and).toBe(true)
        expect(applied.tags).toEqual(['旅行'])
        expect(applied.tags_and).toBe(true)

        // 適用側が握るもの
        expect(applied.rep_types).toEqual(['timeis'])
        expect(applied.reps, 'rep名絞り込みを切らないとサーバ側で常に0件になる').toBeNull()
        expect(applied.hide_tags, '非表示タグは保存時のスナップショットではなく現在の設定から').toEqual(['非表示タグ'])
    })
})

describe('エディタ→保存→適用の往復', () => {
    test('エディタで設定した6フィールドが実行中検索まで生き残る', () => {
        const { config, view } = create_view()
        attach_child_stubs(view)
        view.use_tag_filter.value = true

        const edited = view.generate_query('editor-query-id')
        config.plaing_timeis_json_data = { plaing_timeis_find_kyou_query: JSON.parse(JSON.stringify(edited)) }

        const applied = generate_plaing_timeis_query(config as unknown as ApplicationConfig, new Date(2026, 6, 2, 12, 0))

        expect(applied.keywords, 'キーワードが効いていない').toBe('写真 -除外')
        expect(applied.words).toEqual(['写真'])
        expect(applied.not_words).toEqual(['除外'])
        expect(applied.words_and, 'AND検索の指定が効いていない').toBe(true)
        expect(applied.tags, 'タグ絞り込みが効いていない').toEqual(['旅行'])
        expect(applied.tags_and).toBe(true)
        expect(applied.rep_types).toEqual(['timeis'])
    })

    test('タグ絞り込みOFF（tags=null）も往復で保たれる', () => {
        const { config, view } = create_view()
        attach_child_stubs(view)
        view.use_tag_filter.value = false

        const edited = view.generate_query('editor-query-id')
        expect(edited.tags, 'OFFはnull（未使用）で表す').toBeNull()
        config.plaing_timeis_json_data = { plaing_timeis_find_kyou_query: JSON.parse(JSON.stringify(edited)) }

        const applied = generate_plaing_timeis_query(config as unknown as ApplicationConfig, new Date(2026, 6, 2, 12, 0))
        expect(applied.tags, '往復でタグ絞り込みが勝手にONになっている').toBeNull()
    })
})

describe('タグ絞り込みトグル', () => {
    test('既定はOFF（コンストラクタ既定の tags=[] を保存してしまわない）', () => {
        const { view } = create_view()
        expect(view.use_tag_filter.value).toBe(false)
    })

    test('着信クエリの tags の null 判定から導出される', async () => {
        const { props, view } = create_view()

        const q_on = new FindKyouQuery()
        q_on.query_id = 'q1'
        q_on.tags = ['旅行']
        props.find_kyou_query = q_on
        await flush()
        expect(view.use_tag_filter.value, 'tags非nullなのにトグルが立たない').toBe(true)

        const q_off = q_on.clone()
        q_off.tags = null
        props.find_kyou_query = q_off
        await flush()
        expect(view.use_tag_filter.value, 'tags=null（未使用）なのにトグルが倒れない').toBe(false)
    })
})

describe('inited の集約', () => {
    test('子の @inited が揃うまで loading、揃えば inited を emit する', async () => {
        const { view, emitted } = create_view()
        await nextTick()

        expect(view.loading.value).toBe(true)
        expect(view.inited.value, 'タグツリーがまだなのに inited が立っている').toBe(false)

        view.onInitedKeyword()
        await nextTick()
        expect(view.inited.value, 'キーワードだけで inited が立っている').toBe(false)

        view.onInitedTag()
        await nextTick()
        expect(view.inited.value, '子が揃っても inited が立たない').toBe(true)

        // loading 解除と '@inited' は inited の watcher → nextTick 2段で走る
        await flush()
        expect(view.loading.value, 'inited が立ったのに loading が晴れない（保存ボタンが出ない）').toBe(false)
        expect(emitted, "inited が立ったのに '@inited' を emit していない").toContain('inited')
    })

    test('未セット（query_idが空）なら既定条件を適用し、タグ絞り込みはOFFになる', async () => {
        const { view } = create_view()
        await nextTick()

        view.onInitedTag()
        await flush()

        expect(view.query.value.query_id, '既定条件にはIDを採番する').toBe('generated-uuid')
        expect(view.query.value.tags, '未設定時のplaing検索と同じくタグフィルタ未使用').toBeNull()
        expect(view.query.value.reps, '既定は全rep').toEqual(['timeis_dev_202601'])
        expect(view.use_tag_filter.value).toBe(false)
    })

    test('セット済み（query_idが空でない）なら着信クエリを優先する', async () => {
        const { props, view } = create_view()
        const stored = new FindKyouQuery()
        stored.query_id = 'stored-id'
        stored.keywords = '保存済みの条件'
        stored.tags = ['旅行']
        props.find_kyou_query = stored
        await nextTick()

        view.onInitedTag()
        await flush()

        expect(view.query.value.query_id).toBe('stored-id')
        expect(view.query.value.keywords, '保存済みの条件が既定で潰されている').toBe('保存済みの条件')
        expect(view.use_tag_filter.value).toBe(true)
    })
})
