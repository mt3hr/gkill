<template>
    <v-dialog v-model="is_show_dialog">
        <EditDeviceStructElementView :application_config="application_config" :folder_name="folder_name"
            :gkill_api="gkill_api" :struct_obj="struct_obj"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditDeviceStructElementDialogEmits } from './edit-device-struct-element-dialog-emits';
import type { EditDeviceStructElementDialogProps } from './edit-device-struct-element-dialog-props';
import EditDeviceStructElementView from '../views/edit-device-struct-element-view.vue';

const props = defineProps<EditDeviceStructElementDialogProps>();
const emits = defineEmits<EditDeviceStructElementDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
