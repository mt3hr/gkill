<template>
    <v-dialog persistent @click:outside="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <AddRyuuItemView :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_related_kyou_query="(...related_kyou_query :any[]) => emits('requested_add_related_kyou_query', related_kyou_query[0] as RelatedKyouQuery)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, type Ref } from 'vue'
import AddRyuuItemView from '../views/add-ryuu-item-view.vue';
import type AddRyuuItemDialogProps from './add-ryuu-item-dialog-props';
import type AddRyuuItemDialogEmits from './add-ryuu-item-dialog-emits';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type RelatedKyouQuery from '@/classes/dnote/related-kyou-query';
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

defineExpose({ show, hide })
defineProps<AddRyuuItemDialogProps>()
const emits = defineEmits<AddRyuuItemDialogEmits>()

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
