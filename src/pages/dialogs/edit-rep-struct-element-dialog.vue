<template>
    <v-dialog v-model="is_show_dialog">
        <EditRepStructElementView :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
            :struct_obj="struct_obj" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_rep_struct_element="(rep_struct) => emits('requested_update_rep_struct_element', rep_struct)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditRepStructElementDialogEmits } from './edit-rep-struct-element-dialog-emits'
import type { EditRepStructElementDialogProps } from './edit-rep-struct-element-dialog-props'
import EditRepStructElementView from '../views/edit-rep-struct-element-view.vue'

const props = defineProps<EditRepStructElementDialogProps>()
const emits = defineEmits<EditRepStructElementDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
