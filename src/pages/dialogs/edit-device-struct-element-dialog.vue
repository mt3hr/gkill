<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditDeviceStructElementView :application_config="application_config" :gkill_api="gkill_api"
            :struct_obj="device_struct" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_update_device_struct="(...device_struct :any[]) => emits('requested_update_device_struct', device_struct [0] as DeviceStruct)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditDeviceStructElementDialogEmits } from './edit-device-struct-element-dialog-emits.ts'
import type { EditDeviceStructElementDialogProps } from './edit-device-struct-element-dialog-props.ts'
import EditDeviceStructElementView from '../views/edit-device-struct-element-view.vue'
import { DeviceStruct } from '@/classes/datas/config/device-struct';
import type { GkillError } from '@/classes/api/gkill-error.js'
import type { GkillMessage } from '@/classes/api/gkill-message.js'

defineProps<EditDeviceStructElementDialogProps>()
const emits = defineEmits<EditDeviceStructElementDialogEmits>()
defineExpose({ show, hide })

const device_struct: Ref<DeviceStruct> = ref(new DeviceStruct())
const is_show_dialog: Ref<boolean> = ref(false)

async function show(device_struct_obj: DeviceStruct): Promise<void> {
    device_struct.value = device_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
