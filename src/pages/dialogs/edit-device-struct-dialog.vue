<template>
    <v-dialog v-model="is_show_dialog">
        <EditDeviceStructView :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :struct_obj="struct_obj" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_device_struct_element="(device_struct: any) => emits('requested_update_device_struct_element', device_struct)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import EditDeviceStructView from '../views/edit-device-struct-view.vue'
import type { EditDeviceStructgDialogEmits } from './edit-device-structg-dialog-emits'
import type { EditDeviceStructgDialogProps } from './edit-device-structg-dialog-props'

const props = defineProps<EditDeviceStructgDialogProps>()
const emits = defineEmits<EditDeviceStructgDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
