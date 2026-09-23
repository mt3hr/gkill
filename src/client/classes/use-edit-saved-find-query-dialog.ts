'use strict'

import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import EditSavedFindQueryListDialog from '@/pages/dialogs/edit-saved-find-query-list-dialog.vue'
import FindQueryEditorDialog from '@/pages/dialogs/find-query-editor-dialog.vue'
import MiFindQueryEditorDialog from '@/pages/dialogs/mi-find-query-editor-dialog.vue'
import FindTimeIsQueryEditorDialog from '@/pages/dialogs/find-time-is-query-editor-dialog.vue'
import { computed, ref, type Ref } from 'vue'
import type { EditSavedFindQueryDialogProps } from '@/pages/dialogs/edit-saved-find-query-dialog-props'
import type { EditSavedFindQueryDialogEmits } from '@/pages/dialogs/edit-saved-find-query-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { SavedFindQueryConfig, type SavedFindQueryItem } from '@/classes/datas/config/saved-find-query-config'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'
import { PlayingTimeIsConfig } from '@/classes/datas/config/playing-time-is-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

/** 「検索条件」ダイアログを開くときに渡す、3つのセクションの今の値（設定画面の clone から取る） */
export interface EditSavedFindQueryDialogInitialValues {
    saved_find_query_config: SavedFindQueryConfig
    // null = 未設定（実行中は標準の条件、ダッシュボードは画面側の既定の条件）
    playing_timeis_find_kyou_query: FindKyouQuery | null
    dashboard_dnote_find_kyou_query: FindKyouQuery | null
    dashboard_mi_find_kyou_query: FindKyouQuery | null
}

/**
 * 設定の「検索条件」ダイアログ。検索条件にかかわる設定を1か所に集めたもので、
 * 「検索ショートカット」（保存済みの検索条件）・「実行中」・「ダッシュボード」の3セクションを持つ。
 * 以前は「検索条件」「実行中」「ダッシュボード」が別々のダイアログで、設定画面のあちこちに散っていた。
 *
 * 適用はどのセクションも「設定画面の clone へ組み立てるだけ」で API を呼ばない（送信は設定画面の「適用」）。
 * **適用で渡すのは、このダイアログで触ったセクションだけ。** 旧ダッシュボードダイアログは開いて適用しただけで
 * 未設定（null）の条件を空の FindKyouQuery で書き潰し、ダッシュボードの既定の条件が効かなくなっていた。
 * 3つを1つにまとめたので、全部を渡すと「ショートカットだけ直した」でも同じことが起きる。
 */
export function useEditSavedFindQueryDialog(options: {
    props: EditSavedFindQueryDialogProps
    emits: EditSavedFindQueryDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-saved-find-query-dialog", {
        centerMode: "always",
    })

    // ── 検索ショートカット（保存済みの検索条件） ──
    // キャンセルで破棄できるよう、show() で受け取った設定のクローンを編集する
    const current_saved_find_query_config: Ref<SavedFindQueryConfig> = ref(new SavedFindQueryConfig())
    const is_saved_find_query_edited = ref(false)

    // ── 実行中 ──
    // null = 未設定（デフォルト動作）。チェックボックスのOFFを表現するため null を第一級の状態として持つ
    const current_playing_timeis_query = ref<FindKyouQuery | null>(null)
    // エディタダイアログの v-model 用。null だとエディタが描画されないので別持ちにする
    const playing_timeis_editor_model = ref<FindKyouQuery>(new FindKyouQuery())
    const is_playing_timeis_edited = ref(false)

    // Ryuu の「検索条件をカスタマイズする」と同じ意味論。
    // OFFにすると null に戻る＝デフォルト動作（旧「デフォルトに戻す」ボタン相当）
    const is_use_custom_find_kyou_query = computed<boolean>({
        get: () => current_playing_timeis_query.value !== null,
        set: (value: boolean) => {
            is_playing_timeis_edited.value = true
            if (!value) {
                current_playing_timeis_query.value = null
                return
            }
            if (current_playing_timeis_query.value === null) {
                current_playing_timeis_query.value = FindKyouQuery.generate_default_query_for_playing_timeis(props.application_config)
            }
        },
    })

    // ── ダッシュボード ──
    // エディタの v-model は非 null が要るので、未設定なら空の条件で開く。
    // ただし触らなかった側は元の値（null を含む）のまま返す
    const current_dnote_query = ref<FindKyouQuery>(new FindKyouQuery())
    const current_mi_query = ref<FindKyouQuery>(new FindKyouQuery())
    let initial_dnote_query: FindKyouQuery | null = null
    let initial_mi_query: FindKyouQuery | null = null
    const is_dashboard_dnote_query_edited = ref(false)
    const is_dashboard_mi_query_edited = ref(false)

    async function show(initial?: EditSavedFindQueryDialogInitialValues): Promise<void> {
        current_saved_find_query_config.value = initial?.saved_find_query_config.clone() ?? new SavedFindQueryConfig()
        current_playing_timeis_query.value = initial?.playing_timeis_find_kyou_query ?? null
        initial_dnote_query = initial?.dashboard_dnote_find_kyou_query ?? null
        initial_mi_query = initial?.dashboard_mi_find_kyou_query ?? null
        current_dnote_query.value = initial_dnote_query ?? new FindKyouQuery()
        current_mi_query.value = initial_mi_query ?? new FindKyouQuery()
        is_saved_find_query_edited.value = false
        is_playing_timeis_edited.value = false
        is_dashboard_dnote_query_edited.value = false
        is_dashboard_mi_query_edited.value = false
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const rykv_list_dialog = ref<InstanceType<typeof EditSavedFindQueryListDialog> | null>(null)
    const mi_list_dialog = ref<InstanceType<typeof EditSavedFindQueryListDialog> | null>(null)
    const find_time_is_query_editor_dialog = ref<InstanceType<typeof FindTimeIsQueryEditorDialog> | null>(null)
    const dnote_query_editor_dialog = ref<InstanceType<typeof FindQueryEditorDialog> | null>(null)
    const mi_query_editor_dialog = ref<InstanceType<typeof MiFindQueryEditorDialog> | null>(null)

    // ── 検索ショートカット ──
    function open_rykv_list_dialog(): void {
        rykv_list_dialog.value?.show(current_saved_find_query_config.value.saved_rykv_find_kyou_querys)
    }
    function open_mi_list_dialog(): void {
        mi_list_dialog.value?.show(current_saved_find_query_config.value.saved_mi_find_kyou_querys)
    }
    // 一覧ダイアログの適用はローカル反映のみ。永続化はこのダイアログの保存で親の clone へ渡し、
    // 設定画面全体の「適用」で確定する
    // (一覧側で適用した時点で親に伝えてしまうと、ここでキャンセルしても戻らなくなる)
    function onAppliedRykvItems(items: Array<SavedFindQueryItem>): void {
        current_saved_find_query_config.value.saved_rykv_find_kyou_querys = items
        is_saved_find_query_edited.value = true
    }
    function onAppliedMiItems(items: Array<SavedFindQueryItem>): void {
        current_saved_find_query_config.value.saved_mi_find_kyou_querys = items
        is_saved_find_query_edited.value = true
    }

    // ── 実行中 ──
    function open_playing_timeis_query_editor(): void {
        // このボタンはチェックONのときだけ出るので current_playing_timeis_query は必ず非null
        const initial_query = current_playing_timeis_query.value ?? FindKyouQuery.generate_default_query_for_playing_timeis(props.application_config)
        playing_timeis_editor_model.value = initial_query
        find_time_is_query_editor_dialog.value?.show(initial_query)
    }
    // エディタのSaveはローカル反映のみ。永続化はこのダイアログのSaveで確定する
    function onAppliedPlayingTimeIsQuery(query: FindKyouQuery): void {
        current_playing_timeis_query.value = query
        is_playing_timeis_edited.value = true
    }

    // ── ダッシュボード ──
    function open_dnote_query_editor(): void {
        dnote_query_editor_dialog.value?.show(current_dnote_query.value)
    }
    function open_mi_query_editor(): void {
        mi_query_editor_dialog.value?.show(current_mi_query.value)
    }
    // クエリエディタの適用はローカル反映のみ。永続化はこのダイアログの保存で確定する
    // （エディタで適用した時点で親に伝えてしまうと、ここでキャンセルしても戻らなくなる）
    function onAppliedDnoteQuery(query: FindKyouQuery): void {
        current_dnote_query.value = query
        is_dashboard_dnote_query_edited.value = true
    }
    function onAppliedMiQuery(query: FindKyouQuery): void {
        current_mi_query.value = query
        is_dashboard_mi_query_edited.value = true
    }

    // ── 適用 ──
    function emit_edited_sections(): void {
        if (is_saved_find_query_edited.value) {
            emits('requested_apply_saved_find_query_struct', current_saved_find_query_config.value.to_json())
        }
        if (is_playing_timeis_edited.value) {
            const config = new PlayingTimeIsConfig()
            config.playing_timeis_find_kyou_query = current_playing_timeis_query.value
            emits('requested_apply_playing_timeis', config.to_json())
        }
        if (is_dashboard_dnote_query_edited.value || is_dashboard_mi_query_edited.value) {
            const config = new DashboardConfig()
            config.dashboard_dnote_find_kyou_query = is_dashboard_dnote_query_edited.value ? current_dnote_query.value : initial_dnote_query
            config.dashboard_mi_find_kyou_query = is_dashboard_mi_query_edited.value ? current_mi_query.value : initial_mi_query
            emits('requested_apply_dashboard_struct', config.to_json())
        }
    }
    function onSave(): void {
        emit_edited_sections()
        hide()
    }
    function onCancel(): void {
        hide()
    }

    return {
        // Template refs
        help_dialog,
        rykv_list_dialog,
        mi_list_dialog,
        find_time_is_query_editor_dialog,
        dnote_query_editor_dialog,
        mi_query_editor_dialog,

        // State
        is_show_dialog,
        ui,
        current_saved_find_query_config,
        current_playing_timeis_query,
        playing_timeis_editor_model,
        is_use_custom_find_kyou_query,
        current_dnote_query,
        current_mi_query,

        // 検索ショートカット
        open_rykv_list_dialog,
        open_mi_list_dialog,
        onAppliedRykvItems,
        onAppliedMiItems,

        // 実行中
        open_playing_timeis_query_editor,
        onAppliedPlayingTimeIsQuery,

        // ダッシュボード
        open_dnote_query_editor,
        open_mi_query_editor,
        onAppliedDnoteQuery,
        onAppliedMiQuery,

        // Actions
        onSave,
        onCancel,
        show,
        hide,
    }
}
