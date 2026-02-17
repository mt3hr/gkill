<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <EditDeviceStructView :application_config="application_config" :gkill_api="gkill_api"
            :device_struct="application_config.device_struct"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_apply_device_struct="(...device_struct_element_data: any[]) => emits('requested_apply_device_struct', device_struct_element_data[0] as DeviceStructElementData)"
            @requested_reload_application_config="emits('requested_reload_application_config')"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditDeviceStructDialogEmits } from './edit-device-struct-dialog-emits.ts'
import type { EditDeviceStructDialogProps } from './edit-device-struct-dialog-props.ts'
import EditDeviceStructView from '../views/edit-device-struct-view.vue'
import type { GkillError } from '@/classes/api/gkill-error.js'
import type { GkillMessage } from '@/classes/api/gkill-message.js'

defineProps<EditDeviceStructDialogProps>()
const emits = defineEmits<EditDeviceStructDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import type { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data.js'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}

</script>
