<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("EDIT_DEVICE_STRUCT_TITLE") }}
        </v-card-title>
        <div class="device_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api"
                :folder_name="i18n.global.t('DEVICE_TITLE')" :is_open="true"
                :struct_obj="cloned_application_config.device_struct" :is_editable="true" :is_root="true"
                :is_show_checkbox="false"
                @dblclicked_item="onDblclickedItem"
                @contextmenu_item="show_device_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_device_struct_element_dialog">{{
                        i18n.global.t("ADD_DEVICE_TITLE") }}</v-btn>
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
        <AddNewDeviceStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true"
            v-on="errorMessageRelayHandlers"
            @requested_add_device_struct_element="add_device_struct_element"
            ref="add_new_device_struct_element_dialog" />
        <EditDeviceStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_update_device_struct="update_device_struct" ref="edit_device_struct_element_dialog" />
        <DeviceStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            ref="device_struct_context_menu"
            @requested_edit_device="(...id: any[]) => show_edit_device_struct_dialog(id[0] as string)"
            @requested_delete_device="(...id: any[]) => show_confirm_delete_device_struct_dialog(id[0] as string)" />
        <ConfirmDeleteDeviceStructDialog ref="confirm_delete_device_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            v-on="errorMessageRelayHandlers"
            @requested_delete_device="(...id: any[]) => delete_device_struct(id[0] as string)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditDeviceStructViewEmits } from './edit-device-struct-view-emits'
import type { EditDeviceStructViewProps } from './edit-device-struct-view-props'
import AddNewDeviceStructElementDialog from '../dialogs/add-new-device-struct-element-dialog.vue'
import EditDeviceStructElementDialog from '../dialogs/edit-device-struct-element-dialog.vue'
import FoldableStruct from './foldable-struct.vue'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import DeviceStructContextMenu from './device-struct-context-menu.vue'
import ConfirmDeleteDeviceStructDialog from '../dialogs/confirm-delete-device-struct-dialog.vue'
import { useEditDeviceStructView } from '@/classes/use-edit-device-struct-view'

const props = defineProps<EditDeviceStructViewProps>()
const emits = defineEmits<EditDeviceStructViewEmits>()

const {
    // Template refs
    foldable_struct,
    edit_device_struct_element_dialog,
    add_new_folder_dialog,
    add_new_device_struct_element_dialog,
    device_struct_context_menu,
    confirm_delete_device_struct_dialog,

    // State
    cloned_application_config,

    // Business logic
    reload_cloned_application_config,
    show_device_contextmenu,
    show_edit_device_struct_dialog,
    update_device_struct,
    apply,
    show_add_new_device_struct_element_dialog,
    show_add_new_folder_dialog,
    add_folder_struct_element,
    add_device_struct_element,
    show_confirm_delete_device_struct_dialog,
    delete_device_struct,

    // Template event handlers
    onDblclickedItem,
    onRequestedCloseDialog,

    // Event relay objects
    errorMessageRelayHandlers,
} = useEditDeviceStructView({ props, emits })

defineExpose({ reload_cloned_application_config })
</script>
<style lang="css" scoped>
.device_struct_root {
    max-height: unset;
    overflow-y: scroll;
}
</style>
