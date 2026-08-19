'use strict'

import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import FindTimeIsQueryEditorDialog from '@/pages/dialogs/find-time-is-query-editor-dialog.vue'
import { PlaingTimeIsConfig } from '@/classes/datas/config/plaing-time-is-config'
import { computed, ref, type Ref } from 'vue'
import type { EditPlaingTimeIsDialogProps } from '@/pages/dialogs/edit-plaing-time-is-dialog-props'
import type { EditPlaingTimeIsDialogEmits } from '@/pages/dialogs/edit-plaing-time-is-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

export function useEditPlaingTimeIsDialog(options: {
    props: EditPlaingTimeIsDialogProps
    emits: EditPlaingTimeIsDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("edit-plaing-time-is-dialog", {
        centerMode: "always",
    })

    // null = 未設定（デフォルト動作）。チェックボックスのOFFを表現するため
    // dashboard版と違い null を第一級の状態として持つ
    const current_query = ref<FindKyouQuery | null>(null)
    // エディタダイアログの v-model 用。null だとエディタが描画されないので別持ちにする
    const editor_model = ref<FindKyouQuery>(new FindKyouQuery())

    // Ryuu の「検索条件をカスタマイズする」と同じ意味論。
    // OFFにすると null に戻る＝デフォルト動作（旧「デフォルトに戻す」ボタン相当）
    const is_use_custom_find_kyou_query = computed<boolean>({
        get: () => current_query.value !== null,
        set: (value: boolean) => {
            if (!value) {
                current_query.value = null
                return
            }
            if (current_query.value === null) {
                current_query.value = FindKyouQuery.generate_default_query_for_plaing_timeis(props.application_config)
            }
        },
    })

    async function show(initial_query?: FindKyouQuery): Promise<void> {
        current_query.value = initial_query ?? null
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const find_time_is_query_editor_dialog = ref<InstanceType<typeof FindTimeIsQueryEditorDialog> | null>(null)
    function open_query_editor(): void {
            // このボタンはチェックONのときだけ出るので current_query は必ず非null
            const initial = current_query.value ?? FindKyouQuery.generate_default_query_for_plaing_timeis(props.application_config)
            editor_model.value = initial
            find_time_is_query_editor_dialog.value?.show(initial)
    }
    // エディタのSaveはローカル反映のみ。永続化はこのダイアログのSaveで確定する
    // （dashboard版と違い、キャンセルすれば破棄される）
    function onAppliedQuery(query: FindKyouQuery): void {
            current_query.value = query
    }
    function emit_current_config(): void {
            const config = new PlaingTimeIsConfig()
            config.plaing_timeis_find_kyou_query = current_query.value
            emits('requested_apply_plaing_timeis', config.to_json())
    }
    function onSave(): void {
            emit_current_config()
            hide()
    }
    function onCancel(): void {
            hide()
    }

    return {
        help_dialog,
        find_time_is_query_editor_dialog,
        open_query_editor,
        onAppliedQuery,
        onSave,
        onCancel,
        is_show_dialog,
        ui,
        current_query,
        editor_model,
        is_use_custom_find_kyou_query,
        show,
        hide,
    }
}
