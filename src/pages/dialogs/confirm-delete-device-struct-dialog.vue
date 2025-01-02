<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteDeviceStructView :application_config="application_config" :gkill_api="gkill_api"
            :device_struct="device_struct" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_close_dialog="hide"
            @requested_delete_device="(id) => { emits('requested_delete_device', id); hide() }"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ConfirmDeleteDeviceStructView from '../views/confirm-delete-device-struct-view.vue';
import type { ConfirmDeleteDeviceStructDialogEmits } from './confirm-delete-device-struct-dialog-emits.ts';
import type { ConfirmDeleteDeviceStructDialogProps } from './confirm-delete-device-struct-dialog-props.ts';
import { DeviceStruct } from '@/classes/datas/config/device-struct';

defineProps<ConfirmDeleteDeviceStructDialogProps>()
const emits = defineEmits<ConfirmDeleteDeviceStructDialogEmits>()
defineExpose({ show, hide })

const device_struct: Ref<DeviceStruct> = ref(new DeviceStruct())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(device_struct_obj: DeviceStruct): Promise<void> {
    device_struct.value = device_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    device_struct.value = new DeviceStruct()
}
</script>
