<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditDeviceStructView :application_config="application_config" :gkill_api="gkill_api"
            :device_struct="application_config.device_struct" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="emits('requested_reload_application_config')"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditDeviceStructDialogEmits } from './edit-device-struct-dialog-emits.ts'
import type { EditDeviceStructDialogProps } from './edit-device-struct-dialog-props.ts'
import EditDeviceStructView from '../views/edit-device-struct-view.vue'

defineProps<EditDeviceStructDialogProps>()
const emits = defineEmits<EditDeviceStructDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}

</script>
