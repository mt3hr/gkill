<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditRepTypeStructView :application_config="application_config" :gkill_api="gkill_api"
            :rep_type_struct="application_config.rep_type_struct"
            @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" 
            @requested_close_dialog="hide"
            />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditRepTypeDialogEmits } from './edit-rep-type-dialog-emits'
import type { EditRepTypeDialogProps } from './edit-rep-type-dialog-props'
import EditRepTypeStructView from '../views/edit-rep-type-struct-view.vue'

defineProps<EditRepTypeDialogProps>()
const emits = defineEmits<EditRepTypeDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
