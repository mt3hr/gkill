<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_KFTL_TEMPLATE_STRUCT_TITLE") }}
        </v-card-title>
        <div class="kftl_template_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('KFTL_TEMPLATE_STRUCT_ELEMENT_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.kftl_template_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="onDblclickedItem"
                @contextmenu_item="show_kftl_template_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_kftl_template_struct_element_dialog">{{
                        i18n.global.t("ADD_KFTL_TEMPLATE_STRUCT_ELEMENT_TITLE") }}</v-btn>
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
        <AddNewKFTLTemplateStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true"
            v-on="errorMessageRelayHandlers"
            @requested_add_kftl_template_struct_element="add_kftl_template_struct_element"
            ref="add_new_kftl_template_struct_element_dialog" />
        <EditKFTLTemplateStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_update_kftl_template_struct="update_kftl_template_struct"
            ref="edit_kftl_template_struct_element_dialog" />
        <KFTLTemplateStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            ref="kftl_template_struct_context_menu"
            @requested_edit_kftl_template="(id: string) => show_edit_kftl_template_struct_dialog(id)"
            @requested_delete_kftl_template="(id: string) => show_confirm_delete_kftl_template_struct_dialog(id)" />
        <ConfirmDeleteKFTLTemplateStructDialog ref="confirm_delete_kftl_template_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_delete_kftl_template="(id: string) => delete_kftl_template_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditKFTLTemplateStructViewEmits } from './edit-kftl-template-struct-view-emits'
import type { EditKFTLTemplateStructViewProps } from './edit-kftl-template-struct-view-props'
import AddNewKFTLTemplateStructElementDialog from '../dialogs/add-new-kftl-template-struct-element-dialog.vue'
import EditKFTLTemplateStructElementDialog from '../dialogs/edit-kftl-template-struct-element-dialog.vue'
import FoldableStruct from './foldable-struct.vue'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import KFTLTemplateStructContextMenu from './kftl-template-struct-context-menu.vue'
import ConfirmDeleteKFTLTemplateStructDialog from '../dialogs/confirm-delete-kftl-template-struct-dialog.vue'
import { useEditKftlTemplateStructView } from '@/classes/use-edit-kftl-template-struct-view'

const props = defineProps<EditKFTLTemplateStructViewProps>()
const emits = defineEmits<EditKFTLTemplateStructViewEmits>()

const {
    // Template refs
    foldable_struct,
    edit_kftl_template_struct_element_dialog,
    add_new_folder_dialog,
    add_new_kftl_template_struct_element_dialog,
    kftl_template_struct_context_menu,
    confirm_delete_kftl_template_struct_dialog,

    // State
    cloned_application_config,

    // Business logic
    reload_cloned_application_config,
    show_kftl_template_contextmenu,
    show_edit_kftl_template_struct_dialog,
    update_kftl_template_struct,
    apply,
    show_add_new_kftl_template_struct_element_dialog,
    show_add_new_folder_dialog,
    add_folder_struct_element,
    add_kftl_template_struct_element,
    show_confirm_delete_kftl_template_struct_dialog,
    delete_kftl_template_struct,

    // Template event handlers
    onDblclickedItem,
    onRequestedCloseDialog,

    // Event relay objects
    errorMessageRelayHandlers,
} = useEditKftlTemplateStructView({ props, emits })

defineExpose({ reload_cloned_application_config })
</script>
<style lang="css" scoped>
.kftl_template_struct_root {
    max-height: unset;
    overflow-y: scroll;
}
</style>
