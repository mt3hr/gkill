<template>
    <v-dialog v-model="is_show_dialog">
        <AddLantanaView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { LantanaDialogEmits } from './lantana-dialog-emits'
import type { LantanaDialogProps } from './lantana-dialog-props'
import AddLantanaView from '../views/add-lantana-view.vue'

defineProps<LantanaDialogProps>()
const emits = defineEmits<LantanaDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
