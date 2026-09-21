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
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'

const views_dir = resolve(__dirname, '../../../pages/views')
const read_view = (name: string): string => readFileSync(resolve(views_dir, name), 'utf8')

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
