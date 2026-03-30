<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_REP_TYPE_STRUCT_TITLE") }}
        </v-card-title>
        <div class="rep_type_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('REP_TYPE_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.rep_type_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="onDblclickedItem"
                @contextmenu_item="show_rep_type_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_rep_type_struct_element_dialog">{{
                        i18n.global.t("ADD_REP_TYPE_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_folder_dialog">{{ i18n.global.t("ADD_FOLDER_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ i18n.global.t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="onRequestedCloseDialog">{{
                        i18n.global.t("CANCEL_TITLE")
                        }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            v-on="errorMessageRelayHandlers"
            ref="add_new_folder_dialog" />
        <AddNewRepTypeStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true"
            v-on="errorMessageRelayHandlers"
            @requested_add_rep_type_struct_element="add_rep_type_struct_element"
            ref="add_new_rep_type_struct_element_dialog" />
        <EditRepTypeStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_update_rep_type_struct="update_rep_type_struct" ref="edit_rep_type_struct_element_dialog" />
        <RepTypeStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            ref="rep_type_struct_context_menu"
            @requested_edit_rep_type="(id: string) => show_edit_rep_type_struct_dialog(id)"
            @requested_delete_rep_type="(id: string) => show_confirm_delete_rep_type_struct_dialog(id)" />
        <ConfirmDeleteRepTypeStructDialog ref="confirm_delete_rep_type_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_delete_rep_type="(id: string) => delete_rep_type_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditRepTypeStructViewEmits } from './edit-rep-type-struct-view-emits.js'
import type { EditRepTypeStructViewProps } from './edit-rep-type-struct-view-props.js'
import AddNewRepTypeStructElementDialog from '../dialogs/add-new-rep-type-struct-element-dialog.vue'
import EditRepTypeStructElementDialog from '../dialogs/edit-rep-type-struct-element-dialog.vue'
import FoldableStruct from './foldable-struct.vue'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import RepTypeStructContextMenu from './rep-type-struct-context-menu.vue'
import ConfirmDeleteRepTypeStructDialog from '../dialogs/confirm-delete-rep-type-struct-dialog.vue'
import { useEditRepTypeStructView } from '@/classes/use-edit-rep-type-struct-view'

const props = defineProps<EditRepTypeStructViewProps>()
const emits = defineEmits<EditRepTypeStructViewEmits>()

const {
    // Template refs
    foldable_struct,
    edit_rep_type_struct_element_dialog,
    add_new_folder_dialog,
    add_new_rep_type_struct_element_dialog,
    rep_type_struct_context_menu,
    confirm_delete_rep_type_struct_dialog,

    // State
    cloned_application_config,

    // Business logic
    reload_cloned_application_config,
    show_rep_type_contextmenu,
    show_edit_rep_type_struct_dialog,
    update_rep_type_struct,
    apply,
    show_add_new_rep_type_struct_element_dialog,
    show_add_new_folder_dialog,
    add_folder_struct_element,
    add_rep_type_struct_element,
    show_confirm_delete_rep_type_struct_dialog,
    delete_rep_type_struct,

    // Template event handlers
    onDblclickedItem,
    onRequestedCloseDialog,

    // Event relay objects
    errorMessageRelayHandlers,
} = useEditRepTypeStructView({ props, emits })

defineExpose({ reload_cloned_application_config })
</script>
<style lang="css" scoped>
.rep_type_struct_root {
    max-height: unset;
    overflow-y: scroll;
}
</style>
