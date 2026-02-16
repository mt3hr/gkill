<template>
    <v-dialog persistent @click:outside="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <EditRyuuItemView v-model="model_value" :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, type Ref } from 'vue'
import EditRyuuItemView from '../views/edit-ryuu-item-view.vue';
import type EditRyuuItemDialogEmits from './edit-ryuu-item-dialog-emits';
import type EditRyuuItemDialogProps from './edit-ryuu-item-dialog-props';
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

const model_value = defineModel<RelatedKyouQuery>()
defineExpose({ show, hide })
defineProps<EditRyuuItemDialogProps>()
const emits = defineEmits<EditRyuuItemDialogEmits>()

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
