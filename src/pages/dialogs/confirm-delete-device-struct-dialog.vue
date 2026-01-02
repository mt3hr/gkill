<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteDeviceStructView :application_config="application_config" :gkill_api="gkill_api"
            :device_struct="device_struct" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @requested_close_dialog="hide"
            @requested_delete_device="(...id: any[]) => { emits('requested_delete_device', id[0] as string); hide() }"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import ConfirmDeleteDeviceStructView from '../views/confirm-delete-device-struct-view.vue';
import type { ConfirmDeleteDeviceStructDialogEmits } from './confirm-delete-device-struct-dialog-emits.ts';
import type { ConfirmDeleteDeviceStructDialogProps } from './confirm-delete-device-struct-dialog-props.ts';
import { DeviceStruct } from '@/classes/datas/config/device-struct';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteDeviceStructDialogProps>()
const emits = defineEmits<ConfirmDeleteDeviceStructDialogEmits>()
defineExpose({ show, hide })

const device_struct: Ref<DeviceStruct> = ref(new DeviceStruct())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(device_struct_obj: DeviceStruct): Promise<void> {
    device_struct.value = device_struct_obj
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    device_struct.value = new DeviceStruct()
}
</script>
