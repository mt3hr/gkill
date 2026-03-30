<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("TAG_STRUCT_TITLE") }}
        </v-card-title>
        <div class="tag_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('TAG_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.tag_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="onDblclickedItem"
                @contextmenu_item="show_tag_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_tag_struct_element_dialog">{{
                        i18n.global.t("ADD_TAG_TITLE") }}</v-btn>
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
                        i18n.global.t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            v-on="errorMessageRelayHandlers"
            ref="add_new_folder_dialog" />
        <AddNewTagStructElementDialog :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :is_open="true"
            v-on="errorMessageRelayHandlers"
            @requested_add_tag_struct_element="add_tag_struct_element" ref="add_new_tag_struct_element_dialog" />
        <EditTagStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_update_tag_struct="update_tag_struct" ref="edit_tag_struct_element_dialog" />
        <TagStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            ref="tag_struct_context_menu"
            @requested_edit_tag="(id: string) => show_edit_tag_struct_dialog(id)"
            @requested_delete_tag="(id: string) => show_confirm_delete_tag_struct_dialog(id)" />
        <ConfirmDeleteTagStructDialog ref="confirm_delete_tag_struct_dialog" :application_config="application_config"
            :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_delete_tag="(id: string) => delete_tag_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditTagStructViewEmits } from './edit-tag-struct-view-emits'
import type { EditTagStructViewProps } from './edit-tag-struct-view-props'
import AddNewTagStructElementDialog from '../dialogs/add-new-tag-struct-element-dialog.vue'
import EditTagStructElementDialog from '../dialogs/edit-tag-struct-element-dialog.vue'
import FoldableStruct from './foldable-struct.vue'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import TagStructContextMenu from './tag-struct-context-menu.vue'
import ConfirmDeleteTagStructDialog from '../dialogs/confirm-delete-tag-struct-dialog.vue'
import { useEditTagStructView } from '@/classes/use-edit-tag-struct-view'

const props = defineProps<EditTagStructViewProps>()
const emits = defineEmits<EditTagStructViewEmits>()

const {
    // Template refs
    foldable_struct,
    edit_tag_struct_element_dialog,
    add_new_folder_dialog,
    add_new_tag_struct_element_dialog,
    tag_struct_context_menu,
    confirm_delete_tag_struct_dialog,

    // State
    cloned_application_config,

    // Business logic
    reload_cloned_application_config,
    show_tag_contextmenu,
    show_edit_tag_struct_dialog,
    update_tag_struct,
    apply,
    show_add_new_tag_struct_element_dialog,
    show_add_new_folder_dialog,
    add_folder_struct_element,
    add_tag_struct_element,
    show_confirm_delete_tag_struct_dialog,
    delete_tag_struct,

    // Template event handlers
    onDblclickedItem,
    onRequestedCloseDialog,

    // Event relay objects
    errorMessageRelayHandlers,
} = useEditTagStructView({ props, emits })

defineExpose({ reload_cloned_application_config })
</script>
<style lang="css" scoped>
.tag_struct_root {
    max-height: unset;
    overflow-y: scroll;
}
</style>
