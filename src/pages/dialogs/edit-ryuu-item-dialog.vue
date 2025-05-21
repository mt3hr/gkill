<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditRyuuItemView v-model="model_value" :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import { ref, type Ref } from 'vue'
import EditRyuuItemView from '../views/edit-ryuu-item-view.vue';
import type EditRyuuItemDialogEmits from './edit-ryuu-item-dialog-emits';
import type EditRyuuItemDialogProps from './edit-ryuu-item-dialog-props';
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query';
const is_show_dialog: Ref<boolean> = ref(false)

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
