<template>
    <v-card variant="flat">
        <v-card-title>
            {{ i18n.global.t("EDIT_MI_BOARD_STRUCT_TITLE") }}
        </v-card-title>
        <div class="mi_board_struct_root">
            <!-- 板はフォルダ分けしないのでフラットな一覧になる。
                 is_editable はドラッグ&ドロップ並べ替えと、モバイルの長押しメニューの門番なので外さないこと
                 (デスクトップの右クリックは is_editable に依存しない) -->
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('BOARD_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.mi_board_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @contextmenu_item="show_mi_board_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="onRequestedCloseDialog">{{
                        i18n.global.t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <MiBoardStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            ref="mi_board_struct_context_menu"
            @requested_move_up_mi_board="(id: string) => move_mi_board_struct_up(id)"
            @requested_move_down_mi_board="(id: string) => move_mi_board_struct_down(id)"
            @requested_delete_mi_board="(id: string) => show_confirm_delete_mi_board_struct_dialog(id)" />
        <ConfirmDeleteMiBoardStructDialog ref="confirm_delete_mi_board_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_delete_mi_board="(id: string) => delete_mi_board_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditMiBoardStructViewEmits } from './edit-mi-board-struct-view-emits'
import type { EditMiBoardStructViewProps } from './edit-mi-board-struct-view-props'
import FoldableStruct from './foldable-struct.vue'
import MiBoardStructContextMenu from './mi-board-struct-context-menu.vue'
import ConfirmDeleteMiBoardStructDialog from '../dialogs/confirm-delete-mi-board-struct-dialog.vue'
import { useEditMiBoardStructView } from '@/classes/use-edit-mi-board-struct-view'

const props = defineProps<EditMiBoardStructViewProps>()
const emits = defineEmits<EditMiBoardStructViewEmits>()

const {
    // Template refs
    foldable_struct,
    mi_board_struct_context_menu,
    confirm_delete_mi_board_struct_dialog,

    // State
    cloned_application_config,

    // Business logic
    reload_cloned_application_config,
    show_mi_board_contextmenu,
    apply,
    show_confirm_delete_mi_board_struct_dialog,
    delete_mi_board_struct,
    move_mi_board_struct_up,
    move_mi_board_struct_down,

    // Template event handlers
    onRequestedCloseDialog,

    // Event relay objects
    errorMessageRelayHandlers,
} = useEditMiBoardStructView({ props, emits })

defineExpose({ reload_cloned_application_config })
</script>
<style lang="css" scoped>
.mi_board_struct_root {
    max-height: unset;
    overflow-y: scroll;
}
</style>
