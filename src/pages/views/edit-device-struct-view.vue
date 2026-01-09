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
                @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_device_struct_dialog(id) }"
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
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{
                        i18n.global.t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="add_new_folder_dialog" />
        <AddNewDeviceStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_add_device_struct_element="add_device_struct_element"
            ref="add_new_device_struct_element_dialog" />
        <EditDeviceStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_device_struct="update_device_struct" ref="edit_device_struct_element_dialog" />
        <DeviceStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="device_struct_context_menu"
            @requested_edit_device="(...id: any[]) => show_edit_device_struct_dialog(id[0] as string)"
            @requested_delete_device="(...id: any[]) => show_confirm_delete_device_struct_dialog(id[0] as string)" />
        <ConfirmDeleteDeviceStructDialog ref="confirm_delete_device_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_delete_device="(...id: any[]) => delete_device_struct(id[0] as string)" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { nextTick, type Ref, ref, watch } from 'vue'
import type { EditDeviceStructViewEmits } from './edit-device-struct-view-emits'
import type { EditDeviceStructViewProps } from './edit-device-struct-view-props'
import AddNewDeviceStructElementDialog from '../dialogs/add-new-device-struct-element-dialog.vue'
import EditDeviceStructElementDialog from '../dialogs/edit-device-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'

import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import DeviceStructContextMenu from './device-struct-context-menu.vue'
import ConfirmDeleteDeviceStructDialog from '../dialogs/confirm-delete-device-struct-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const foldable_struct = ref<InstanceType<typeof FoldableStruct> | null>(null);
const edit_device_struct_element_dialog = ref<InstanceType<typeof EditDeviceStructElementDialog> | null>(null);
const add_new_folder_dialog = ref<InstanceType<typeof AddNewFoloderDialog> | null>(null);
const add_new_device_struct_element_dialog = ref<InstanceType<typeof AddNewDeviceStructElementDialog> | null>(null);
const device_struct_context_menu = ref<InstanceType<typeof DeviceStructContextMenu> | null>(null);
const confirm_delete_device_struct_dialog = ref<InstanceType<typeof ConfirmDeleteDeviceStructDialog> | null>(null);

const props = defineProps<EditDeviceStructViewProps>()
const emits = defineEmits<EditDeviceStructViewEmits>()
defineExpose({ reload_cloned_application_config })

watch(() => props.application_config, () => reload_cloned_application_config())
const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
}

function show_device_contextmenu(e: MouseEvent, id: string | null): void {
    if (id) {
        device_struct_context_menu.value?.show(e, id)
    }
}

function show_edit_device_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: DeviceStructElementData | null = null
    let device_name_walk = (_device: DeviceStructElementData): void => { }
    device_name_walk = (device: DeviceStructElementData): void => {
        const device_children = device.children
        if (device.id === id) {
            target_struct_object = device
        } else if (device_children) {
            device_children.forEach(child_device => {
                if (child_device) {
                    device_name_walk(child_device)
                }
            })
        }
    }
    device_name_walk(cloned_application_config.value.device_struct)

    if (!target_struct_object) {
        return
    }

    edit_device_struct_element_dialog.value?.show(target_struct_object)
}

function update_device_struct(device_struct_obj: DeviceStructElementData): void {
    let device_name_walk = (_device: DeviceStructElementData): boolean => false
    device_name_walk = (device: DeviceStructElementData): boolean => {
        const device_children = device.children
        if (device.id === device_struct_obj.id) {
            return true
        } else if (device_children) {
            for (let i = 0; i < device_children.length; i++) {
                const child_device = device_children[i]
                if (child_device.children) {
                    if (device_name_walk(child_device)) {
                        device_children[i] = device_struct_obj
                        return false
                    }
                }
            }
        }
        return false
    }
    device_name_walk(cloned_application_config.value.device_struct)
}

async function apply(): Promise<void> {
    emits('requested_apply_device_struct', cloned_application_config.value.device_struct)
    nextTick(() => emits('requested_close_dialog'))
}
function show_add_new_device_struct_element_dialog(): void {
    add_new_device_struct_element_dialog.value?.show()
}
function show_add_new_folder_dialog(): void {
    add_new_folder_dialog.value?.show()
}
async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
    const device_struct_element = new DeviceStructElementData()
    device_struct_element.id = folder_struct_element.id
    device_struct_element.is_dir = true
    device_struct_element.check_when_inited = false
    device_struct_element.device_name = folder_struct_element.folder_name
    device_struct_element.children = new Array<DeviceStructElementData>()
    device_struct_element.key = folder_struct_element.folder_name
    cloned_application_config.value.device_struct.children?.push(device_struct_element)
}
async function add_device_struct_element(device_struct_element: DeviceStructElementData): Promise<void> {
    cloned_application_config.value.device_struct.children?.push(device_struct_element)
}
function show_confirm_delete_device_struct_dialog(id: string): void {
    let target_struct_object: DeviceStructElementData | null = null
    let device_name_walk = (_device: DeviceStructElementData): void => { }
    device_name_walk = (device: DeviceStructElementData): void => {
        const device_children = device.children
        if (device.id === id) {
            target_struct_object = device
        } else if (device_children) {
            device_children.forEach(child_device => {
                if (child_device) {
                    device_name_walk(child_device)
                }
            })
        }
    }
    device_name_walk(cloned_application_config.value.device_struct)

    if (!target_struct_object) {
        return
    }
    confirm_delete_device_struct_dialog.value?.show(target_struct_object)
}
function delete_device_struct(id: string): void {
    let device_name_walk = (_device: DeviceStructElementData): boolean => false
    device_name_walk = (device: DeviceStructElementData): boolean => {
        const device_children = device.children
        if (device.id === id) {
            return true
        } else if (device_children) {
            for (let i = 0; i < device_children.length; i++) {
                const child_device = device_children[i]
                if (child_device.children) {
                    if (device_name_walk(child_device)) {
                        device_children.splice(i, 1)
                        return false
                    }
                }
            }
        }
        return false
    }
    device_name_walk(cloned_application_config.value.device_struct)
}
</script>
<style lang="css" scoped>
.device_struct_root {
    max-height: 80vh;
    overflow-y: scroll;
}
</style>