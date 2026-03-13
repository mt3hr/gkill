<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t('EDIT_REP_STRUCT_TITLE') }}
        </v-card-title>
        <div class="rep_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('REP_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.rep_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="onDblclickedItem"
                @contextmenu_item="show_rep_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_rep_struct_element_dialog">{{
                        i18n.global.t("ADD_REP_TITLE") }}</v-btn>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
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
        <AddNewRepStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true"
            v-on="addNewRepHandlers" ref="add_new_rep_struct_element_dialog" />
        <EditRepStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="editRepHandlers" ref="edit_rep_struct_element_dialog" />
        <RepStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            v-on="repContextMenuHandlers"
            ref="rep_struct_context_menu" />
        <ConfirmDeleteRepStructDialog ref="confirm_delete_rep_struct_dialog" :application_config="application_config"
            :gkill_api="gkill_api"
            v-on="confirmDeleteHandlers" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditRepStructViewEmits } from './edit-rep-struct-view-emits'
import type { EditRepStructViewProps } from './edit-rep-struct-view-props'
import AddNewRepStructElementDialog from '../dialogs/add-new-rep-struct-element-dialog.vue'
import EditRepStructElementDialog from '../dialogs/edit-rep-struct-element-dialog.vue'
import FoldableStruct from './foldable-struct.vue'
import RepStructContextMenu from './rep-struct-context-menu.vue'
import ConfirmDeleteRepStructDialog from '../dialogs/confirm-delete-rep-struct-dialog.vue'
import { useEditRepStructView } from '@/classes/use-edit-rep-struct-view'

const props = defineProps<EditRepStructViewProps>()
const emits = defineEmits<EditRepStructViewEmits>()

const {
    foldable_struct,
    edit_rep_struct_element_dialog,
    add_new_rep_struct_element_dialog,
    rep_struct_context_menu,
    confirm_delete_rep_struct_dialog,
    cloned_application_config,
    reload_cloned_application_config,
    show_rep_contextmenu,
    apply,
    show_add_new_rep_struct_element_dialog,
    onDblclickedItem,
    onRequestedCloseDialog,
    addNewRepHandlers,
    editRepHandlers,
    repContextMenuHandlers,
    confirmDeleteHandlers,
} = useEditRepStructView({ props, emits })

defineExpose({ reload_cloned_application_config })
</script>
<style lang="css" scoped>
.rep_struct_root {
    max-height: unset;
    overflow-y: scroll;
}
</style>
