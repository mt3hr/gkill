<template>
    <v-dialog v-model="is_show_dialog">
        <AddNewDeviceStructElementView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide"
            @requested_add_device_struct_element="(device_struct_element) => emits('requested_add_device_struct_element', device_struct_element)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewDeviceStructElementDialogEmits } from './add-new-device-struct-element-dialog-emits'
import type { AddNewDeviceStructElementDialogProps } from './add-new-device-struct-element-dialog-props'
import AddNewDeviceStructElementView from '../views/add-new-device-struct-element-view.vue'

const add_new_device_struct_element_view = ref<InstanceType<typeof AddNewDeviceStructElementView> | null>(null);

const props = defineProps<AddNewDeviceStructElementDialogProps>()
const emits = defineEmits<AddNewDeviceStructElementDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    add_new_device_struct_element_view.value?.reset_device_name()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    add_new_device_struct_element_view.value?.reset_device_name()
}
</script>
