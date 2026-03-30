import { computed, type Ref, ref } from 'vue'
import type { RyuuItemContextMenuEmits } from '@/pages/views/ryuu-item-context-menu-emits'
import type EditRyuuItemDialog from '@/pages/dialogs/edit-ryuu-item-dialog.vue'
import type ConfirmDeleteRyuuItemDialog from '@/pages/dialogs/confirm-delete-ryuu-item-dialog.vue'
import type { ModelRef } from 'vue'
import type RelatedKyouQuery from '@/classes/dnote/related-kyou-query'

export function useRyuuItemContextMenu(options: {
    emits: RyuuItemContextMenuEmits,
    edit_related_kyou_query_dialog: Ref<InstanceType<typeof EditRyuuItemDialog> | null>,
    confirm_delete_ryuu_item_dialog: Ref<InstanceType<typeof ConfirmDeleteRyuuItemDialog> | null>,
    model_value: ModelRef<RelatedKyouQuery | undefined>,
}) {
    const { emits: _emits, edit_related_kyou_query_dialog, confirm_delete_ryuu_item_dialog, model_value } = options

    // ── State refs ──
    const is_show: Ref<boolean> = ref(false)
    const position_x: Ref<number> = ref(0)
    const position_y: Ref<number> = ref(0)
    const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 2))), position_y.value.valueOf())}px; }`)

    // ── Methods ──
    async function show(e: PointerEvent): Promise<void> {
        position_x.value = e.clientX
        position_y.value = e.clientY
        is_show.value = true
    }

    async function hide(): Promise<void> {
        is_show.value = false
    }

    async function show_edit_related_kyou_query_dialog(): Promise<void> {
        edit_related_kyou_query_dialog.value?.show()
    }

    async function show_confirm_delete_ryuu_item_dialog(): Promise<void> {
        confirm_delete_ryuu_item_dialog.value?.show(model_value.value!)
    }

    // ── Return ──
    return {
        // State
        is_show,
        position_x,
        position_y,
        context_menu_style,

        // Methods
        show,
        hide,
        show_edit_related_kyou_query_dialog,
        show_confirm_delete_ryuu_item_dialog,
    }
}
