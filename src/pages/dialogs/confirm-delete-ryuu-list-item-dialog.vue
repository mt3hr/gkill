<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteRelatedKyouQueryView :application_config="application_config" :gkill_api="gkill_api"
            :related_kyou_query="related_kyou_query" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_close_dialog="hide"
            @requested_delete_related_kyou_query="(id) => { emits('requested_delete_related_kyou_query', id); hide() }"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query';
import ConfirmDeleteRelatedKyouQueryView from '../views/confirm-delete-related-kyou-query-view.vue';
import type { ConfirmDeleteRyuuListItemDialogProps } from './confirm-delete-ryuu-list-item-dialog-props';
import type { ConfirmDeleteRyuuListItemDialogEmits } from './confirm-delete-ryuu-list-item-dialog-emits';

defineProps<ConfirmDeleteRyuuListItemDialogProps>()
const emits = defineEmits<ConfirmDeleteRyuuListItemDialogEmits>()
defineExpose({ show, hide })

const related_kyou_query: Ref<RelatedKyouQuery> = ref(new RelatedKyouQuery())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(related_kyou_query_: RelatedKyouQuery): Promise<void> {
    related_kyou_query.value = related_kyou_query_
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    related_kyou_query.value = new RelatedKyouQuery()
}
</script>
