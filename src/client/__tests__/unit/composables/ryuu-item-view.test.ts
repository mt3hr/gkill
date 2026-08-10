/**
 * useRyuuItemView（流の1項目）の検証。
 *
 * 1. 中継束の override。手書きで3件だけ並べていた頃は残り15件を落としていて、
 *    流に出ている Kyou は編集してもタグを足しても永久に古いままだった。
 * 2. タイトル一致条件の data_type 判定順。"mirekyou" は "mi" に前方一致するので
 *    MiReKyou を先に判定しないと、自分のタイトルを持たない MiReKyou のときに
 *    参照先の Mi のタイトルで検索してしまう。
 * 3. キーワードグループは words / not_words の非nullが有効の印。
 *    片側だけ非nullにすると parse_words_and_not_words の扱いが崩れるので、
 *    words を入れるときは not_words = [] も必ず入れる。
 */
import { describe, expect, test, vi, beforeEach } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

// 引き直しの手順そのものは kyou-reload.test.ts が見る。ここでは呼ばれ方と結果の反映だけ見る
vi.mock('@/classes/kyou-reload', () => ({
    refresh_kyou: vi.fn(),
    refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
    new_reload_batch: vi.fn(() => 0),
    build_mi_reload_query: vi.fn((query: unknown) => query),
}))

import { ref, toRaw } from 'vue'
import { useRyuuItemView } from '@/classes/use-ryuu-item-view'
import { refresh_kyou } from '@/classes/kyou-reload'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import AndPredicate from '@/classes/dnote/dnote-predicate/and-predicate'
import EqualTitleTargetKyouPredicate from '@/classes/dnote/dnote-predicate/target-kyou-predicate/equal-title-target-kyou-predicate'
import { RelatedTimeMatchType } from '@/classes/dnote/related-time-match-type'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { makeApplicationConfig } from '../../helpers/factory'
import type { Kyou } from '@/classes/datas/kyou'
import type RyuuItemViewProps from '@/pages/views/ryuu-item-view-props'
import type RyuuItemViewEmits from '@/pages/views/ryuu-item-view-emits'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

const refresh_kyou_mock = vi.mocked(refresh_kyou)

interface EmittedEvent {
    event: string
    args: Array<unknown>
}

/** Kyou の実クラスは通信を伴うので、この画面が触るフィールドだけの構造フェイクを使う */
function make_kyou(overrides: Record<string, unknown> = {}): Kyou {
    return {
        id: 'kyou-1',
        data_type: 'kmemo',
        related_time: new Date('2026-03-15T09:00:00+09:00'),
        attached_tags: [],
        typed_mi: null,
        typed_kmemo: null,
        async load_all(): Promise<Array<unknown>> { return [] },
        ...overrides,
    } as unknown as Kyou
}

function make_model_value(): RelatedKyouQuery {
    const query = new RelatedKyouQuery()
    query.id = 'related-query-1'
    // タイトル一致条件つき。これがあると検索条件へ words が積まれる
    query.predicate = new AndPredicate([new EqualTitleTargetKyouPredicate('')])
    query.related_time_match_type = RelatedTimeMatchType.near_related_time
    query.find_kyou_query = null
    query.find_duration_hour = 1
    return query
}

function create_view(options: { target_kyou?: Kyou | null } = {}) {
    const emitted = new Array<EmittedEvent>()
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, args: args })
    }) as unknown as RyuuItemViewEmits
    const get_kyous = vi.fn().mockResolvedValue({ kyous: [], messages: null, errors: null })
    const props = {
        gkill_api: {
            get_kyous: get_kyous,
            delete_updated_gkill_caches: vi.fn().mockResolvedValue(undefined),
        },
        application_config: makeApplicationConfig() as unknown as ApplicationConfig,
        find_kyou_query_default: new FindKyouQuery(),
        target_kyou: options.target_kyou ?? null,
        matched_kyous: null,
        enable_context_menu: true,
        enable_dialog: true,
        abort_controller: new AbortController(),
        editable: false,
    } as unknown as RyuuItemViewProps
    const model_value = ref<RelatedKyouQuery | undefined>(make_model_value())
    const view = useRyuuItemView({ props: props, emits: emits, model_value: model_value })
    return { view: view, props: props, emitted: emitted, get_kyous: get_kyous, model_value: model_value }
}

/** ref に入れた Kyou はプロキシ越しに返るので、同一性は生の参照で見る */
function shown_kyou(view: ReturnType<typeof useRyuuItemView>): Kyou | null {
    return view.match_kyou.value ? toRaw(view.match_kyou.value) : null
}

/** 中継ハンドラは await されずに走るのでマイクロタスクを回す */
async function flush(): Promise<void> {
    for (let i = 0; i < 4; i++) {
        await Promise.resolve()
    }
}

beforeEach(() => {
    refresh_kyou_mock.mockReset()
    refresh_kyou_mock.mockResolvedValue(null)
})

describe('中継束の override', () => {
    test('同じ id の updated_kyou で表示中の Kyou が差し替わる', async () => {
        const { view, emitted } = create_view()
        view.match_kyou.value = make_kyou({ id: 'kyou-1' })
        const refreshed = make_kyou({ id: 'kyou-1', data_type: 'kmemo' })
        refresh_kyou_mock.mockResolvedValue(refreshed)

        view.kyouViewRelayHandlers.updated_kyou(make_kyou({ id: 'kyou-1' }))
        await flush()

        expect(refresh_kyou_mock, '引き直しが呼ばれていない').toHaveBeenCalledTimes(1)
        expect(shown_kyou(view), '引き直した結果で差し替えていない（古いまま残る）').toBe(refreshed)
        expect(emitted.map((entry) => entry.event)).toContain('updated_kyou')
    })

    test('別の id の updated_kyou は無視する', async () => {
        const { view } = create_view()
        const shown = make_kyou({ id: 'kyou-1' })
        view.match_kyou.value = shown

        view.kyouViewRelayHandlers.updated_kyou(make_kyou({ id: 'kyou-other' }))
        await flush()

        expect(refresh_kyou_mock, '無関係な Kyou の更新で引き直している').not.toHaveBeenCalled()
        expect(shown_kyou(view)).toBe(shown)
    })

    test('requested_reload_kyou でも同じ id なら引き直す', async () => {
        const { view, emitted } = create_view()
        view.match_kyou.value = make_kyou({ id: 'kyou-1' })
        const refreshed = make_kyou({ id: 'kyou-1' })
        refresh_kyou_mock.mockResolvedValue(refreshed)

        view.kyouViewRelayHandlers.requested_reload_kyou(make_kyou({ id: 'kyou-1' }))
        await flush()

        expect(shown_kyou(view), 'タグ/テキスト/通知の変更を反映していない').toBe(refreshed)
        expect(emitted.map((entry) => entry.event)).toContain('requested_reload_kyou')
    })

    test('引き直しに失敗したら古い表示を残す', async () => {
        const { view } = create_view()
        const shown = make_kyou({ id: 'kyou-1' })
        view.match_kyou.value = shown
        refresh_kyou_mock.mockResolvedValue(null)

        view.kyouViewRelayHandlers.updated_kyou(make_kyou({ id: 'kyou-1' }))
        await flush()

        expect(shown_kyou(view), '半分だけ読めた状態や null で潰してはいけない').toBe(shown)
    })

    test('同じ id の deleted_kyou で表示が消えて「データなし」になる', async () => {
        const { view, emitted } = create_view()
        view.match_kyou.value = make_kyou({ id: 'kyou-1' })

        view.kyouViewRelayHandlers.deleted_kyou(make_kyou({ id: 'kyou-1' }))
        await flush()

        expect(view.match_kyou.value).toBeNull()
        expect(view.is_no_data.value, '削除後も「データなし」表示にならない').toBe(true)
        expect(emitted.map((entry) => entry.event)).toContain('deleted_kyou')
    })

    test('別の id の deleted_kyou では消さない', async () => {
        const { view } = create_view()
        const shown = make_kyou({ id: 'kyou-1' })
        view.match_kyou.value = shown

        view.kyouViewRelayHandlers.deleted_kyou(make_kyou({ id: 'kyou-other' }))
        await flush()

        expect(shown_kyou(view)).toBe(shown)
        expect(view.is_no_data.value).toBe(false)
    })
})

describe('タイトル一致条件の data_type 判定順', () => {
    test('Mi のときは Mi のタイトルで検索する', async () => {
        const target = make_kyou({ id: 'mi-1', data_type: 'mi_create', typed_mi: { title: 'タスクの題名' } })
        const { view, get_kyous } = create_view({ target_kyou: target })

        await view.load_related_kyou()

        expect(get_kyous).toHaveBeenCalledTimes(1)
        const query = get_kyous.mock.calls[0][0].query as FindKyouQuery
        expect(query.words, 'Mi のタイトルが検索条件へ載っていない').toEqual(['タスクの題名'])
    })

    // "mirekyou_*" は "mi" にも前方一致する。mi を先に判定していると、
    // 自分のタイトルを持たない MiReKyou で参照先 Mi のタイトルを拾ってしまう
    test.each([
        'mirekyou_create',
        'mirekyou_check',
        'mirekyou_limit',
        'mirekyou_start',
        'mirekyou_end',
    ])('%s は対象外なのでキーワード条件を積まない', async (data_type) => {
        const target = make_kyou({ id: 'mirekyou-1', data_type: data_type, typed_mi: { title: 'タスクの題名' } })
        const { view, get_kyous } = create_view({ target_kyou: target })

        await view.load_related_kyou()

        const query = get_kyous.mock.calls[0][0].query as FindKyouQuery
        expect(
            query.words,
            'mirekyou を mi より先に判定していない（MiReKyou のタイトルで検索している）',
        ).toBeNull()
        expect(query.not_words).toBeNull()
    })

    test.each([
        { data_type: 'lantana', typed: {} },
        { data_type: 'rekyou', typed: {} },
    ])('$data_type のようにタイトルを持たない種別でも積まない', async ({ data_type }) => {
        const target = make_kyou({ id: 'x-1', data_type: data_type })
        const { view, get_kyous } = create_view({ target_kyou: target })

        await view.load_related_kyou()

        const query = get_kyous.mock.calls[0][0].query as FindKyouQuery
        expect(query.words).toBeNull()
    })

    test('タイトルが空文字なら積まない', async () => {
        const target = make_kyou({ id: 'mi-1', data_type: 'mi_create', typed_mi: { title: '' } })
        const { view, get_kyous } = create_view({ target_kyou: target })

        await view.load_related_kyou()

        const query = get_kyous.mock.calls[0][0].query as FindKyouQuery
        expect(query.words).toBeNull()
    })
})

describe('キーワードグループの null 規約', () => {
    // words だけ非nullにすると片肺になり、parse_words_and_not_words の扱いが崩れる。
    // 「非nullの空配列 = 有効だが空指定」なので、not_words も明示的に [] を入れる
    test('words を積むときは not_words = [] も一緒に入れる', async () => {
        const target = make_kyou({ id: 'kmemo-1', data_type: 'kmemo', typed_kmemo: { content: 'メモ本文' } })
        const { view, get_kyous } = create_view({ target_kyou: target })

        await view.load_related_kyou()

        const query = get_kyous.mock.calls[0][0].query as FindKyouQuery
        expect(query.words).toEqual(['メモ本文'])
        expect(query.not_words, 'not_words が null のままだとキーワードグループが片肺になる').toEqual([])
    })

    test('条件を積まないときは両方 null のまま（フィルタ未使用）', async () => {
        const target = make_kyou({ id: 'lantana-1', data_type: 'lantana' })
        const { view, get_kyous } = create_view({ target_kyou: target })

        await view.load_related_kyou()

        const query = get_kyous.mock.calls[0][0].query as FindKyouQuery
        expect(query.words).toBeNull()
        expect(query.not_words).toBeNull()
    })
})

describe('該当なしの扱い', () => {
    test('1件も見つからなければ is_no_data が立つ', async () => {
        const target = make_kyou({ id: 'kmemo-1', data_type: 'kmemo', typed_kmemo: { content: 'メモ本文' } })
        const { view } = create_view({ target_kyou: target })

        await view.load_related_kyou()

        expect(view.match_kyou.value).toBeNull()
        expect(view.is_no_data.value).toBe(true)
    })
})
