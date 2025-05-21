<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddRyuuItemView :application_config="application_config" :gkill_api="gkill_api"
            @requested_add_related_kyou_query="(related_kyou_query) => emits('requested_add_related_kyou_query', related_kyou_query)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import { ref, type Ref } from 'vue'
import AddRyuuItemView from '../views/add-ryuu-item-view.vue';
import type AddRyuuItemDialogProps from './add-ryuu-item-dialog-props';
import type AddRyuuItemDialogEmits from './add-ryuu-item-dialog-emits';
const is_show_dialog: Ref<boolean> = ref(false)

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
