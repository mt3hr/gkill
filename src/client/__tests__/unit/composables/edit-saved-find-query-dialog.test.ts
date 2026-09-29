/**
 * 設定の「検索条件」ダイアログ（use-edit-saved-find-query-dialog）の検証。
 *
 * 検索ショートカット（保存済みの検索条件）・実行中・ダッシュボードの3セクションを1つのダイアログに集めた。
 * 以前は3つの別々のダイアログで、そのうち実行中の検証（edit-playing-time-is-dialog.test.ts）をここへ移した。
 *
 * 1. 「検索条件をカスタマイズする」チェックの意味論は3状態で決まる:
 *   - OFF にしたら null（未設定＝従来どおりの既定動作）へ戻す
 *   - null から ON にしたら ApplicationConfig 由来の既定条件を生成する
 *   - 既に条件があるのに ON を押し直しても、その条件を潰さない
 *   3つ目を落とすと、チェックを触っただけで編集中の条件が消える。
 * 2. 適用で渡すのは触ったセクションだけ。全部を渡すと、ショートカットだけ直したときにも
 *    ダッシュボードの未設定（null）の条件が空の条件で書き潰され、既定の条件が効かなくなる
 *    （旧ダッシュボードダイアログは開いて適用しただけでそうなっていた）。
 * 3. 子ダイアログ（一覧・エディタ）の適用を受ける onApplied* は、値の取り込みと「触った」印の
 *    両方を行う。印を落とすと、エディタで「適用」を押したのに保存が黙って渡らない
 *    （利用者にはエラーも警告も出ず、設定画面の「適用」のあとで開き直すと元のまま）。
 *    取り込みを落とすと、印だけ立って開いたときの値が渡る。
 * 4. show() は受け取った SavedFindQueryConfig を書き換えない（clone してから編集する）。
 *    今の呼び出し元（use-application-config-view.ts の show_edit_saved_find_query_dialog）は
 *    開くたびに parse した新しい実体を渡すので、clone を外しても書き換わるのはその使い捨ての
 *    実体だけで、今は実害が出ない。呼び出し元が設定画面の clone の実体をそのまま渡す形に
 *    変わると、このダイアログのキャンセルが効かなくなるので、約束として固定しておく。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: vi.fn(),
    close_dialog_via_history: vi.fn(),
}))
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: vi.fn(() => ({})),
}))

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { toRaw } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { SavedFindQueryConfig, type SavedFindQueryItem } from '@/classes/datas/config/saved-find-query-config'
import { useEditSavedFindQueryDialog, type EditSavedFindQueryDialogInitialValues } from '@/classes/use-edit-saved-find-query-dialog'
import type { EditSavedFindQueryDialogProps } from '@/pages/dialogs/edit-saved-find-query-dialog-props'
import type { EditSavedFindQueryDialogEmits } from '@/pages/dialogs/edit-saved-find-query-dialog-emits'

// ApplicationConfig の実物は循環importを引き込むので、既定条件生成が触る枝だけの構造フェイクを使う
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
            children: null,
        },
    }
}

function create_dialog() {
    const emitted = new Array<{ event: string, args: Array<unknown> }>()
    const props = {
        application_config: make_fake_application_config(),
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        app_content_height: 800,
        app_content_width: 1200,
    } as unknown as EditSavedFindQueryDialogProps
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event, args })
    }) as unknown as EditSavedFindQueryDialogEmits
    return { view: useEditSavedFindQueryDialog({ props, emits }), emitted }
}

function initial_values(overrides: Partial<EditSavedFindQueryDialogInitialValues> = {}): EditSavedFindQueryDialogInitialValues {
    return {
        saved_find_query_config: new SavedFindQueryConfig(),
        playing_timeis_find_kyou_query: null,
        dashboard_dnote_find_kyou_query: null,
        dashboard_mi_find_kyou_query: null,
        ...overrides,
    }
}

function query_with_keywords(keywords: string): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = `id-${keywords}`
    query.keywords = keywords
    return query
}

function shortcut_item(title: string): SavedFindQueryItem {
    return { id: `item-${title}`, title, find_kyou_query: query_with_keywords(title) }
}

type EmittedShortcuts = Record<'saved_rykv_find_kyou_querys' | 'saved_mi_find_kyou_querys', Array<{ title: string }>>

describe('show()', () => {
    test('引数なしなら実行中は未設定（null）から始まり、チェックはOFF', async () => {
        const { view } = create_dialog()
        expect(view.is_show_dialog.value).toBe(false)

        await view.show()

        expect(view.is_show_dialog.value).toBe(true)
        expect(view.current_playing_timeis_query.value, '未設定は null で表す').toBeNull()
        expect(view.is_use_custom_find_kyou_query.value).toBe(false)
        expect(view.playing_timeis_editor_model.value, 'null だとエディタが描画されないので別持ちにしてある').not.toBeNull()
    })

    test('保存済みの実行中の条件を渡したらそれをそのまま編集対象にし、チェックはON', async () => {
        const { view } = create_dialog()
        const stored = query_with_keywords('保存済みの条件')

        await view.show(initial_values({ playing_timeis_find_kyou_query: stored }))

        // ref 越しに reactive proxy が被るので、同一性は toRaw で見る
        expect(toRaw(view.current_playing_timeis_query.value), '渡した条件が編集対象になっていない').toBe(stored)
        expect(view.is_use_custom_find_kyou_query.value).toBe(true)
    })

    test('開き直しは前回の編集内容を引きずらない', async () => {
        const { view } = create_dialog()
        await view.show(initial_values({ playing_timeis_find_kyou_query: query_with_keywords('前回') }))

        await view.show()

        expect(view.current_playing_timeis_query.value, '未設定で開き直したのに前回の条件が残っている').toBeNull()
        expect(view.is_use_custom_find_kyou_query.value).toBe(false)
    })

    test('ダッシュボードの条件は渡した値で開き、未設定なら空の条件でエディタを開ける', async () => {
        const { view } = create_dialog()
        const dnote_query = query_with_keywords('集計')

        await view.show(initial_values({ dashboard_dnote_find_kyou_query: dnote_query }))

        expect(toRaw(view.current_dnote_query.value)).toBe(dnote_query)
        expect(view.current_mi_query.value).not.toBeNull()
    })
})

describe('is_use_custom_find_kyou_query の3状態', () => {
    test('OFF にすると null（既定動作）へ戻る', async () => {
        const { view } = create_dialog()
        await view.show(initial_values({ playing_timeis_find_kyou_query: query_with_keywords('保存済みの条件') }))

        view.is_use_custom_find_kyou_query.value = false

        expect(view.current_playing_timeis_query.value, 'OFFは null でなければ「未設定」を表せない').toBeNull()
        expect(view.is_use_custom_find_kyou_query.value).toBe(false)
    })

    test('null から ON にすると既定条件を生成する', async () => {
        const { view } = create_dialog()
        await view.show()

        view.is_use_custom_find_kyou_query.value = true

        const generated = view.current_playing_timeis_query.value
        expect(generated, 'ONにしても条件が生成されない（エディタが空になる）').not.toBeNull()
        expect(generated?.tags, '未設定時のplaying検索と同じくタグフィルタ未使用').toBeNull()
        expect(generated?.reps, '既定は全rep').toEqual(['timeis_dev_202601'])
        expect(view.is_use_custom_find_kyou_query.value).toBe(true)
    })

    test('既に条件があるのに ON を押し直しても既存を潰さない', async () => {
        const { view } = create_dialog()
        const stored = query_with_keywords('保存済みの条件')
        await view.show(initial_values({ playing_timeis_find_kyou_query: stored }))

        view.is_use_custom_find_kyou_query.value = true

        expect(toRaw(view.current_playing_timeis_query.value), 'ONを押し直しただけで編集中の条件が既定へ差し替わっている').toBe(stored)
        expect(view.current_playing_timeis_query.value?.keywords).toBe('保存済みの条件')
    })
})

describe('適用で渡すのは触ったセクションだけ', () => {
    test('何も触らずに適用しても何も渡さない（未設定の条件を空の条件で書き潰さない）', async () => {
        const { view, emitted } = create_dialog()
        await view.show()

        view.onSave()

        expect(emitted.map(e => e.event)).toEqual([])
    })

    test('ショートカットだけ触ったら保存済みの検索条件だけを渡す', async () => {
        const { view, emitted } = create_dialog()
        await view.show()

        view.onAppliedRykvItems([])
        view.onSave()

        expect(emitted.map(e => e.event)).toEqual(['requested_apply_saved_find_query_struct'])
    })

    test('実行中のチェックを外したら null を渡す', async () => {
        const { view, emitted } = create_dialog()
        await view.show(initial_values({ playing_timeis_find_kyou_query: query_with_keywords('保存済みの条件') }))

        view.is_use_custom_find_kyou_query.value = false
        view.onSave()

        expect(emitted.map(e => e.event)).toEqual(['requested_apply_playing_timeis'])
        expect((emitted[0].args[0] as Record<string, unknown>).playing_timeis_find_kyou_query).toBeNull()
    })

    test('実行中のエディタで適用した条件は、チェックを触らなくても実行中の設定として渡る', async () => {
        const { view, emitted } = create_dialog()
        // チェックは開いた時点で ON（保存済みの条件がある）。ここでは触らず、エディタの適用だけで印が立つことを見る
        await view.show(initial_values({ playing_timeis_find_kyou_query: query_with_keywords('保存済みの条件') }))

        view.onAppliedPlayingTimeIsQuery(query_with_keywords('エディタで直した条件'))
        view.onSave()

        expect(emitted.map(e => e.event), 'エディタで適用したのに実行中の設定が渡らない').toEqual(['requested_apply_playing_timeis'])
        const playing = emitted[0].args[0] as Record<string, Record<string, unknown> | null>
        expect(playing.playing_timeis_find_kyou_query?.keywords, 'エディタで適用した条件ではなく開いたときの条件が渡っている').toBe('エディタで直した条件')
    })

    test('タスクのショートカットだけ触ったら、保存済みの検索条件のタスク側にその一覧が入る', async () => {
        const { view, emitted } = create_dialog()
        await view.show()

        view.onAppliedMiItems([shortcut_item('今日のタスク')])
        view.onSave()

        expect(emitted.map(e => e.event), 'タスクの一覧で適用したのに保存済みの検索条件が渡らない').toEqual(['requested_apply_saved_find_query_struct'])
        const shortcuts = emitted[0].args[0] as EmittedShortcuts
        expect(shortcuts.saved_mi_find_kyou_querys.map(item => item.title), '一覧で適用したタスクのショートカットが入っていない').toEqual(['今日のタスク'])
        expect(shortcuts.saved_rykv_find_kyou_querys, 'タスクの一覧を触ったのに記録側へ入っている').toEqual([])
    })

    test('ダッシュボードの片方だけ触ったら、もう片方は元の値（未設定なら null）のまま渡す', async () => {
        const { view, emitted } = create_dialog()
        await view.show()

        view.onAppliedDnoteQuery(query_with_keywords('集計'))
        view.onSave()

        expect(emitted.map(e => e.event)).toEqual(['requested_apply_dashboard_struct'])
        const dashboard = emitted[0].args[0] as Record<string, Record<string, unknown> | null>
        expect(dashboard.dashboard_dnote_find_kyou_query?.keywords).toBe('集計')
        expect(dashboard.dashboard_mi_find_kyou_query, '触っていないタスク検索条件を空の条件で書き潰している').toBeNull()
    })

    test('ダッシュボードのタスク側だけ触ったら、集計側は初期値（未設定なら null）のまま渡す', async () => {
        const { view, emitted } = create_dialog()
        await view.show()

        view.onAppliedMiQuery(query_with_keywords('タスク'))
        view.onSave()

        expect(emitted.map(e => e.event), 'タスク側のエディタで適用したのにダッシュボードの設定が渡らない').toEqual(['requested_apply_dashboard_struct'])
        const dashboard = emitted[0].args[0] as Record<string, Record<string, unknown> | null>
        expect(dashboard.dashboard_mi_find_kyou_query?.keywords, 'エディタで適用した条件ではなく開いたときの条件が渡っている').toBe('タスク')
        expect(dashboard.dashboard_dnote_find_kyou_query, '触っていない集計検索条件を空の条件で書き潰している').toBeNull()
    })

    test('ダッシュボードのタスク側だけ触ったら、集計側に保存済みの条件があればそれをそのまま渡す', async () => {
        const { view, emitted } = create_dialog()
        await view.show(initial_values({ dashboard_dnote_find_kyou_query: query_with_keywords('保存済みの集計条件') }))

        view.onAppliedMiQuery(query_with_keywords('タスク'))
        view.onSave()

        expect(emitted.map(e => e.event)).toEqual(['requested_apply_dashboard_struct'])
        const dashboard = emitted[0].args[0] as Record<string, Record<string, unknown> | null>
        expect(dashboard.dashboard_mi_find_kyou_query?.keywords).toBe('タスク')
        expect(dashboard.dashboard_dnote_find_kyou_query?.keywords, '触っていない集計側の保存済み条件が消えている').toBe('保存済みの集計条件')
    })

    test('キャンセルでは何も渡さない', async () => {
        const { view, emitted } = create_dialog()
        await view.show()

        view.onAppliedRykvItems([])
        view.onAppliedDnoteQuery(query_with_keywords('集計'))
        view.onCancel()

        expect(emitted).toEqual([])
    })

    test('開き直すと「触った」印も戻る', async () => {
        const { view, emitted } = create_dialog()
        await view.show()
        view.onAppliedRykvItems([])
        view.onCancel()

        await view.show()
        view.onSave()

        expect(emitted).toEqual([])
    })
})

describe('show() に渡した設定は編集しても書き換わらない', () => {
    test('一覧の適用も編集中の項目の書き換えも、渡した SavedFindQueryConfig には及ばない', async () => {
        const { view } = create_dialog()
        const given = new SavedFindQueryConfig()
        given.saved_rykv_find_kyou_querys = [shortcut_item('記録の条件')]
        given.saved_mi_find_kyou_querys = [shortcut_item('タスクの条件')]
        await view.show(initial_values({ saved_find_query_config: given }))

        // 一覧ダイアログの適用（配列の差し替え）と、編集中の項目そのものの書き換えの両方を試す
        view.onAppliedMiItems([])
        view.current_saved_find_query_config.value.saved_rykv_find_kyou_querys[0].title = '書き換え後'
        view.onCancel()

        expect(given.saved_mi_find_kyou_querys.map(item => item.title), '一覧の適用が渡した設定を直接書き換えている').toEqual(['タスクの条件'])
        expect(given.saved_rykv_find_kyou_querys[0].title, '編集中の項目が渡した設定の項目と同じ実体になっている').toBe('記録の条件')
    })
})
