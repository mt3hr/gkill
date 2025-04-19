<template>
    <v-card>
        <v-card-title>
            {{ $t("EDIT_DEVICE_STRUCT_TITLE") }}
        </v-card-title>
        <div class="device_struct_root">
            <FoldableStruct :application_config="application_config" :gkill_api="gkill_api" :folder_name="$t('DEVICE_TITLE')"
                :is_open="true" :struct_obj="cloned_application_config.parsed_device_struct" :is_editable="true"
                :is_root="true" :is_show_checkbox="false"
                @dblclicked_item="(e: MouseEvent, id: string | null) => { if (id) show_edit_device_struct_dialog(id) }"
                @contextmenu_item="show_device_contextmenu" ref="foldable_struct" />
        </div>
        <v-card-action>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_device_struct_element_dialog">{{ $t("ADD_DEVICE_TITLE") }}</v-btn>
                </v-col>
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="show_add_new_folder_dialog">{{ $t("ADD_FOLDER_TITLE") }}</v-btn>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark @click="apply" color="primary">{{ $t("APPLY_TITLE") }}</v-btn>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="secondary" @click="emits('requested_close_dialog')">{{ $t("CANCEL_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-action>
        <AddNewFoloderDialog :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_new_folder="add_folder_struct_element"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="add_new_folder_dialog" />
        <AddNewDeviceStructElementDialog :application_config="application_config" :folder_name="''"
            :gkill_api="gkill_api" :is_open="true" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_device_struct_element="add_device_struct_element"
            ref="add_new_device_struct_element_dialog" />
        <EditDeviceStructElementDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_device_struct="update_device_struct" ref="edit_device_struct_element_dialog" />
        <DeviceStructContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="device_struct_context_menu"
            @requested_edit_device="(id) => show_edit_device_struct_dialog(id)"
            @requested_delete_device="(id) => show_confirm_delete_device_struct_dialog(id)" />
        <ConfirmDeleteDeviceStructDialog ref="confirm_delete_device_struct_dialog"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_device="(id) => delete_device_struct(id)" />
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref, watch } from 'vue'
import type { EditDeviceStructViewEmits } from './edit-device-struct-view-emits'
import type { EditDeviceStructViewProps } from './edit-device-struct-view-props'
import AddNewDeviceStructElementDialog from '../dialogs/add-new-device-struct-element-dialog.vue'
import EditDeviceStructElementDialog from '../dialogs/edit-device-struct-element-dialog.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import FoldableStruct from './foldable-struct.vue'
import { DeviceStruct } from '@/classes/datas/config/device-struct'
import type { FoldableStructModel } from './foldable-struct-model'
import { UpdateDeviceStructRequest } from '@/classes/api/req_res/update-device-struct-request'
import AddNewFoloderDialog from '../dialogs/add-new-foloder-dialog.vue'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import type { FolderStructElementData } from '@/classes/datas/config/folder-struct-element-data'
import DeviceStructContextMenu from './device-struct-context-menu.vue'
import ConfirmDeleteDeviceStructDialog from '../dialogs/confirm-delete-device-struct-dialog.vue'

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

cloned_application_config.value.parse_device_struct()

async function reload_cloned_application_config(): Promise<void> {
    cloned_application_config.value = props.application_config.clone()
    cloned_application_config.value.parse_device_struct()
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
    let target_struct_object: DeviceStruct | null = null

    for (let i = 0; i < cloned_application_config.value.device_struct.length; i++) {
        const device_struct = cloned_application_config.value.device_struct[i]
        if (device_struct.id === id) {
            target_struct_object = device_struct
        }
    }

    if (!target_struct_object) {
        return
    }

    edit_device_struct_element_dialog.value?.show(target_struct_object)
}

function update_device_struct(device_struct_obj: DeviceStruct): void {
    for (let i = 0; i < cloned_application_config.value.device_struct.length; i++) {
        if (cloned_application_config.value.device_struct[i].id === device_struct_obj.id) {
            cloned_application_config.value.device_struct.splice(i, 1, device_struct_obj)
            break
        }
    }
    if (cloned_application_config.value.parsed_device_struct.children) {
        update_seq(cloned_application_config.value.parsed_device_struct.children)
    }
}

function update_seq(_device_struct: Array<FoldableStructModel>): void {
    const exist_ids = new Array<string>()

    // 並び順再決定
    let f = (_struct: FoldableStructModel, _parent: FoldableStructModel, _seq: number) => { }
    let func = (struct: FoldableStructModel, parent: FoldableStructModel, seq: number) => {
        if (struct.id) {
            exist_ids.push(struct.id)
        }

        for (let i = 0; i < cloned_application_config.value.device_struct.length; i++) {
            if (struct.id === cloned_application_config.value.device_struct[i].id) {
                cloned_application_config.value.device_struct[i].seq = seq
                cloned_application_config.value.device_struct[i].parent_folder_id = parent.id
            }
        }
        if (struct.children) {
            for (let i = 0; i < struct.children.length; i++) {
                f(struct.children[i], struct, i)
            }
        }
    }
    f = func
    if (cloned_application_config.value.parsed_device_struct.children) {
        for (let i = 0; i < cloned_application_config.value.parsed_device_struct.children?.length; i++) {
            f(cloned_application_config.value.parsed_device_struct.children[i], cloned_application_config.value.parsed_device_struct, i)
        }
    }

    // 存在しないものを消す
    for (let i = 0; i < cloned_application_config.value.device_struct.length; i++) {
        let exist = false
        for (let j = 0; j < exist_ids.length; j++) {
            if (cloned_application_config.value.device_struct[i].id === exist_ids[j]) {
                exist = true
            }
        }
        if (!exist) {
            cloned_application_config.value.device_struct.splice(i, 1)
            i--
        }
    }

}

async function apply(): Promise<void> {
    const device_struct = foldable_struct.value?.get_foldable_struct()
    if (!device_struct) {
        return
    }

    update_seq(device_struct)

    // 更新する
    const req = new UpdateDeviceStructRequest()
    req.device_struct = cloned_application_config.value.device_struct
    const res = await props.gkill_api.update_device_struct(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('requested_close_dialog')
}
function show_add_new_device_struct_element_dialog(): void {
    add_new_device_struct_element_dialog.value?.show()
}
function show_add_new_folder_dialog(): void {
    add_new_folder_dialog.value?.show()
}
async function add_folder_struct_element(folder_struct_element: FolderStructElementData): Promise<void> {
    const req = new GetGkillInfoRequest()
    const res = await props.gkill_api.get_gkill_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const device_struct = new DeviceStruct()
    device_struct.id = folder_struct_element.id
    device_struct.user_id = res.user_id
    device_struct.device = res.device
    device_struct.check_when_inited = false
    device_struct.parent_folder_id = null
    device_struct.seq = cloned_application_config.value.parsed_device_struct.children ? cloned_application_config.value.parsed_device_struct.children.length : 0
    device_struct.device_name = folder_struct_element.folder_name

    const device_struct_element = new DeviceStructElementData()
    device_struct_element.id = folder_struct_element.id
    device_struct_element.check_when_inited = false
    device_struct_element.device_name = folder_struct_element.folder_name
    device_struct_element.children = new Array<DeviceStructElementData>()
    device_struct_element.key = folder_struct_element.folder_name

    cloned_application_config.value.device_struct.push(device_struct)
    cloned_application_config.value.parsed_device_struct.children?.push(device_struct_element)
}
async function add_device_struct_element(device_struct_element: DeviceStructElementData): Promise<void> {
    const req = new GetGkillInfoRequest()
    const res = await props.gkill_api.get_gkill_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    const device_struct = new DeviceStruct()
    device_struct.id = device_struct_element.id ? device_struct_element.id : ""
    device_struct.user_id = res.user_id
    device_struct.device = res.device
    device_struct.check_when_inited = device_struct_element.check_when_inited
    device_struct.parent_folder_id = null
    device_struct.seq = cloned_application_config.value.parsed_device_struct.children ? cloned_application_config.value.parsed_device_struct.children.length : 0
    device_struct.device_name = device_struct_element.device_name

    cloned_application_config.value.device_struct.push(device_struct)
    cloned_application_config.value.parsed_device_struct.children?.push(device_struct_element)
    await cloned_application_config.value.parse_device_struct()
}
function show_confirm_delete_device_struct_dialog(id: string): void {
    if (!foldable_struct.value) {
        return
    }
    let target_struct_object: DeviceStruct | null = null

    for (let i = 0; i < cloned_application_config.value.device_struct.length; i++) {
        const device_struct = cloned_application_config.value.device_struct[i]
        if (device_struct.id === id) {
            target_struct_object = device_struct
        }
    }

    if (!target_struct_object) {
        return
    }
    confirm_delete_device_struct_dialog.value?.show(target_struct_object)
}
function delete_device_struct(id: string): void {
    for (let i = 0; i < cloned_application_config.value.device_struct.length; i++) {
        if (cloned_application_config.value.device_struct[i].id === id) {
            cloned_application_config.value.device_struct.splice(i, 1)
            break
        }
    }
    foldable_struct.value?.delete_struct(id)
}
</script>
<style lang="css" scoped>
.device_struct_root {
    max-height: 80vh;
    overflow-y: scroll;
}
</style>