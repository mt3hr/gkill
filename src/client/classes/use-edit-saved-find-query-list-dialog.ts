'use strict'

import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import FindQueryEditorDialog from '@/pages/dialogs/find-query-editor-dialog.vue'
import MiFindQueryEditorDialog from '@/pages/dialogs/mi-find-query-editor-dialog.vue'
import { computed, ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import type { EditSavedFindQueryListDialogProps } from '@/pages/dialogs/edit-saved-find-query-list-dialog-props'
import type { EditSavedFindQueryListDialogEmits } from '@/pages/dialogs/edit-saved-find-query-list-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { SavedFindQueryConfig, type SavedFindQueryItem } from '@/classes/datas/config/saved-find-query-config'

export function useEditSavedFindQueryListDialog(options: {
    props: EditSavedFindQueryListDialogProps
    emits: EditSavedFindQueryListDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    // rykv用とmi用の2インスタンスが同居するため、位置・サイズの保存キーを分ける
    const ui = useFloatingDialog(`edit-saved-find-query-list-dialog-${props.query_type}`, {
        centerMode: "always",
    })

    // キャンセルで破棄できるよう、show() で受け取ったリストのクローンを編集する
    const editing_items: Ref<Array<SavedFindQueryItem>> = ref([])
    const current_editing_index: Ref<number> = ref(-1)
    const current_editing_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

    const title = computed(() => props.query_type === 'rykv'
        ? i18n.global.t('SAVED_RYKV_FIND_KYOU_QUERY_TITLE')
        : i18n.global.t('SAVED_MI_FIND_KYOU_QUERY_TITLE'))

    async function show(items: Array<SavedFindQueryItem>): Promise<void> {
        editing_items.value = SavedFindQueryConfig.clone_items(items)
        current_editing_index.value = -1
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    function add_item(): void {
        const find_kyou_query = props.query_type === 'rykv'
            ? FindKyouQuery.generate_default_query_for_rykv(props.application_config)
            : FindKyouQuery.generate_default_query_for_mi(props.application_config)
        editing_items.value.push({
            id: props.gkill_api.generate_uuid(),
            title: i18n.global.t('SAVED_FIND_QUERY_DEFAULT_NAME'),
            find_kyou_query: find_kyou_query,
        })
    }

    function floating_action_button_style() {
        return {
            bottom: '60px',
            right: '10px',
            height: '50px',
            width: '50px',
        }
    }

    function delete_item(index: number): void {
        editing_items.value.splice(index, 1)
    }

    function move_item(index: number, direction: 'up' | 'down'): void {
        const target_index = direction === 'up' ? index - 1 : index + 1
        if (target_index < 0 || target_index >= editing_items.value.length) {
            return
        }
        const items = editing_items.value
        const moved_item = items[index]
        items[index] = items[target_index]
        items[target_index] = moved_item
    }

    // クエリエディタの適用は編集中の行に書き戻すだけ。
    // 永続化はこのダイアログ→ハブダイアログ→設定画面の「適用」で確定する
    function apply_edited_query(query: FindKyouQuery): void {
        if (current_editing_index.value < 0 || current_editing_index.value >= editing_items.value.length) {
            return
        }
        editing_items.value[current_editing_index.value].find_kyou_query = query
    }

    function onSave(): void {
        emits('requested_apply_saved_find_querys', editing_items.value)
        hide()
    }

    function onCancel(): void {
        hide()
    }

    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const find_query_editor_dialog = ref<InstanceType<typeof FindQueryEditorDialog> | null>(null)
    const mi_find_query_editor_dialog = ref<InstanceType<typeof MiFindQueryEditorDialog> | null>(null)
    // クエリ編集は種別に応じたエディタダイアログを開き、適用は編集中の行にだけ反映する
    // (ここで親へ流すとこのダイアログのキャンセルが効かなくなる)
    function open_query_editor(index: number): void {
            current_editing_index.value = index
            current_editing_query.value = editing_items.value[index].find_kyou_query
            if (props.query_type === 'rykv') {
                    find_query_editor_dialog.value?.show(editing_items.value[index].find_kyou_query)
            } else {
                    mi_find_query_editor_dialog.value?.show(editing_items.value[index].find_kyou_query)
            }
    }
    function onAppliedQuery(query: FindKyouQuery): void {
            apply_edited_query(query)
    }

    return {
        help_dialog,
        find_query_editor_dialog,
        mi_find_query_editor_dialog,
        open_query_editor,
        onAppliedQuery,
        is_show_dialog,
        ui,
        editing_items,
        current_editing_index,
        current_editing_query,
        title,
        show,
        hide,
        add_item,
        delete_item,
        move_item,
        apply_edited_query,
        floating_action_button_style,
        onSave,
        onCancel,
    }
}
