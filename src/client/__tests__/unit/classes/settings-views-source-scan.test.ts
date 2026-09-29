/**
 * 設定画面の .vue テンプレートの配線を、ソース走査で固定する。
 *
 * mount で検査しないのは、Vuetify の v-item-group / v-list を実描画するとテストが
 * コンポーネントツリー全体を引き込むため。ここで見るのは「emit 名と受け口の名前が一致している」
 * 「ラベルのキーが正しい」という、壊れても型検査も lint も通ってしまう配線だけ。
 *
 * - 板のコンテキストメニューの「編集」（8cafc11c）: emit と受け口の名前が食い違うと、
 *   メニューは出るのに何も起きない
 * - 時間帯の曜日ボタン（7a324e95）: 選択＝塗り潰し・未選択＝枠線・aria-pressed
 * - 記録保管場所の追加画面（8cafc11c で直した取り違え）: 「初期化時チェック」のラベルは
 *   CHECK_WHEN_INITED_TITLE（IS_FORCE_HIDE_TITLE だと i18n のキーは実在するので緑のまま）
 * - 設定の「検索条件」ダイアログ（756ad897 で3ダイアログを統合）: 適用 emit 3本が設定画面の
 *   対応する受け口へ繋がり、受け口が clone の対応する欄へ書くこと。子ダイアログ5つは一覧2つが
 *   同じコンポーネント、エディタ3つが同じ emit 名なので、取り違えても型検査は通る。
 *   取り違えると「適用」を押したのに黙って保存されない・別のセクションに保存される
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

const views_dir = resolve(__dirname, '../../../pages/views')
const dialogs_dir = resolve(__dirname, '../../../pages/dialogs')
const classes_dir = resolve(__dirname, '../../../classes')
const read_view = (name: string): string => readFileSync(resolve(views_dir, name), 'utf8')
const read_dialog = (name: string): string => readFileSync(resolve(dialogs_dir, name), 'utf8')
const read_class = (name: string): string => readFileSync(resolve(classes_dir, name), 'utf8')

/**
 * テンプレートから、その ref 名を持つ子コンポーネントのタグ（`<Tag … ref="name" />`）を1つ切り出す。
 * 属性値の `=>` は `/>` ではないので、タグの開きから最初の `/>` までの間に ref があるものを探す
 * （手前のタグから始めた候補は自分の `/>` で止まって捨てられる）。
 */
function find_component_tag_by_ref(source: string, ref_name: string): string | null {
    const pattern = new RegExp(`<[A-Z]\\w*\\b(?:(?!/>)[\\s\\S])*?\\sref="${ref_name}"\\s*/>`)
    return source.match(pattern)?.[0] ?? null
}

/** コンポーザブルから `function name(…) … {` の本文を切り出す（4スペース字下げの関数だけ） */
function find_function_body(source: string, name: string): string | null {
    const pattern = new RegExp(`\\n    function ${name}\\([^)]*\\)[^{]*\\{([\\s\\S]*?)\\n    \\}`)
    return source.match(pattern)?.[1] ?? null
}

describe('板のコンテキストメニューの「編集」', () => {
    test('メニューは requested_edit_mi_board を emit し、編集ビューがそれを編集ダイアログへ繋ぐ', () => {
        const menu = read_view('mi-board-struct-context-menu.vue')
        expect(menu).toContain("emits('requested_edit_mi_board', id)")
        expect(menu, '「編集」のラベルが共通キーでない').toContain('i18n.global.t("EDIT_TITLE")')

        const emits = read_view('mi-board-struct-context-menu-emits.ts')
        expect(emits).toMatch(/\(e: 'requested_edit_mi_board', id: string\): void/)

        const view = read_view('edit-mi-board-struct-view.vue')
        expect(view, 'メニューの emit を受ける口が無い').toMatch(/@requested_edit_mi_board="\(id: string\) => show_edit_mi_board_struct_dialog\(id\)"/)
        expect(view, 'ダブルクリックで編集を開く配線が無い').toContain('@dblclicked_item="onDblclickedItem"')
    })
})

describe('時間帯の曜日ボタン', () => {
    test('選択は塗り潰し、未選択は枠線で、aria-pressed を持つ', () => {
        const source = read_view('period-of-time-query.vue')
        expect(source).toContain('class="pa-0 ma-0 period_of_time_week_of_day_button"')
        expect(source, '選択/未選択の見た目が variant で分かれていない').toContain(`:variant="isSelected ? 'flat' : 'outlined'"`)
        expect(source, 'E2E と支援技術が押下状態を読む属性が無い').toContain(':aria-pressed="isSelected"')
        expect(source, '7曜日を v-item-group で複数選択する形').toContain('v-model="week_of_days" multiple')
    })
})

describe('記録保管場所の追加・編集画面', () => {
    test.each(['add-new-rep-struct-element-view.vue', 'edit-rep-struct-element-view.vue'])('%s の「初期化時チェック」のラベルは CHECK_WHEN_INITED_TITLE', (name) => {
        const source = read_view(name)
        const checkbox = source.match(/<v-checkbox v-model="check_when_inited"[^>]*>/)
        expect(checkbox, 'check_when_inited のチェックボックスが無い').not.toBeNull()
        expect(checkbox?.[0]).toContain(`i18n.global.t('CHECK_WHEN_INITED_TITLE')`)
        expect(checkbox?.[0], '「強制非表示」のラベルを取り違えている').not.toContain('IS_FORCE_HIDE_TITLE')
    })
})

describe('設定の「検索条件」ダイアログ（検索ショートカット・実行中・ダッシュボード）', () => {
    // ダイアログの適用 emit（触ったセクションだけ出る）と、設定画面の受け口・受け口が書く clone の欄の対。
    // 3本とも引数は Record<string, unknown> なので、受け口を取り違えても型検査は通る
    const apply_wirings = [
        { emit: 'requested_apply_saved_find_query_struct', handler: 'onRequestedApplySavedFindQueryStruct', field: 'saved_find_query_json_data' },
        { emit: 'requested_apply_playing_timeis', handler: 'onRequestedApplyPlayingTimeIs', field: 'playing_timeis_json_data' },
        { emit: 'requested_apply_dashboard_struct', handler: 'onRequestedApplyDashboardStruct', field: 'dashboard_json_data' },
    ]

    test('ダイアログが宣言する requested_apply_* は表の3本と一致する（走査の自己テスト）', () => {
        const emits = read_dialog('edit-saved-find-query-dialog-emits.ts')
        const declared = [...emits.matchAll(/\(e: '(requested_apply_\w+)', \w+: Record<string, unknown>\): void/g)].map(m => m[1])
        expect(declared.sort(), 'セクションを足したら設定画面の配線の表にも足すこと').toEqual(apply_wirings.map(w => w.emit).sort())
    })

    test('設定画面のタグが3本の適用 emit をそれぞれの受け口へ繋ぎ、受け口は clone の対応する欄へ書く', () => {
        const tag = find_component_tag_by_ref(read_view('application-config-view.vue'), 'edit_saved_find_query_dialog')
        expect(tag, '<EditSavedFindQueryDialog … ref="edit_saved_find_query_dialog" /> が見つからない').not.toBeNull()
        expect(tag).toMatch(/^<EditSavedFindQueryDialog\b/)

        const composable = read_class('use-application-config-view.ts')
        for (const { emit, handler, field } of apply_wirings) {
            expect(tag, `${emit} が ${handler} に繋がっていない`).toMatch(new RegExp(`@${emit}="\\(data: \\w+\\) => ${handler}\\(data\\)"`))
            const body = find_function_body(composable, handler)
            expect(body, `${handler} が use-application-config-view.ts に無い`).not.toBeNull()
            expect(body, `${handler} が ${field} 以外へ書いている（別のセクションに保存される）`).toMatch(new RegExp(`cloned_application_config\\.value\\.${field} = \\w+`))
            expect(body, `${handler} が props の設定を直接書いている（設定画面のキャンセルが効かなくなる）`).not.toContain('props.application_config')
            expect(body, `${handler} が未適用の編集の印を立てていない（props の差し替えで消える）`).toContain('has_pending_child_edits = true')
        }
    })

    // 子ダイアログの ref 名・コンポーネント・見分け（query_type / v-model）・開く関数と渡す値・適用の受け口の対。
    // 一覧2つは同じコンポーネント、エディタ3つは同じ emit 名 requested_apply なので、
    // 受け口や v-model を入れ替えても型検査は通り、「タスクの一覧で適用したら記録側に保存される」になる
    const child_dialogs = [
        {
            ref: 'rykv_list_dialog', tag: 'EditSavedFindQueryListDialog', discriminator: `:query_type="'rykv'"`,
            emit: 'requested_apply_saved_find_querys', handler: 'onAppliedRykvItems',
            opener: 'open_rykv_list_dialog', shown_with: 'current_saved_find_query_config.value.saved_rykv_find_kyou_querys',
        },
        {
            ref: 'mi_list_dialog', tag: 'EditSavedFindQueryListDialog', discriminator: `:query_type="'mi'"`,
            emit: 'requested_apply_saved_find_querys', handler: 'onAppliedMiItems',
            opener: 'open_mi_list_dialog', shown_with: 'current_saved_find_query_config.value.saved_mi_find_kyou_querys',
        },
        {
            ref: 'find_time_is_query_editor_dialog', tag: 'FindTimeIsQueryEditorDialog', discriminator: 'v-model="playing_timeis_editor_model"',
            emit: 'requested_apply', handler: 'onAppliedPlayingTimeIsQuery',
            opener: 'open_playing_timeis_query_editor', shown_with: 'initial_query',
        },
        {
            ref: 'dnote_query_editor_dialog', tag: 'FindQueryEditorDialog', discriminator: 'v-model="current_dnote_query"',
            emit: 'requested_apply', handler: 'onAppliedDnoteQuery',
            opener: 'open_dnote_query_editor', shown_with: 'current_dnote_query.value',
        },
        {
            ref: 'mi_query_editor_dialog', tag: 'MiFindQueryEditorDialog', discriminator: 'v-model="current_mi_query"',
            emit: 'requested_apply', handler: 'onAppliedMiQuery',
            opener: 'open_mi_query_editor', shown_with: 'current_mi_query.value',
        },
    ]

    test('ダイアログの子ダイアログは help_dialog を除いて表の5つと一致する（走査の自己テスト）', () => {
        const dialog = read_dialog('edit-saved-find-query-dialog.vue')
        const template = dialog.slice(0, dialog.indexOf('<script'))
        const refs = [...template.matchAll(/\sref="(\w+)"/g)].map(m => m[1]).filter(name => name !== 'help_dialog')
        expect(refs.sort(), '子ダイアログを足したら配線の表にも足すこと').toEqual(child_dialogs.map(d => d.ref).sort())
    })

    test.each(child_dialogs)('$ref は $tag で、開く関数が $shown_with を渡し、適用が $handler へ届く', ({ ref, tag, discriminator, emit, handler, opener, shown_with }) => {
        const element = find_component_tag_by_ref(read_dialog('edit-saved-find-query-dialog.vue'), ref)
        expect(element, `ref="${ref}" の子ダイアログが見つからない`).not.toBeNull()
        expect(element, `ref="${ref}" のコンポーネントが違う`).toMatch(new RegExp(`^<${tag}\\b`))
        expect(element, `ref="${ref}" の見分け（query_type / v-model）が違う`).toContain(discriminator)
        expect(element, `${ref} の適用が ${handler} に繋がっていない`).toMatch(new RegExp(`@${emit}="\\((\\w+)\\) => ${handler}\\(\\1\\)"`))

        const composable = read_class('use-edit-saved-find-query-dialog.ts')
        expect(find_function_body(composable, handler), `${handler} が use-edit-saved-find-query-dialog.ts に無い`).not.toBeNull()
        const opener_body = find_function_body(composable, opener)
        expect(opener_body, `${opener} が use-edit-saved-find-query-dialog.ts に無い`).not.toBeNull()
        expect(opener_body, `${opener} が ${ref} を ${shown_with} で開いていない`).toContain(`${ref}.value?.show(${shown_with})`)
    })
})
