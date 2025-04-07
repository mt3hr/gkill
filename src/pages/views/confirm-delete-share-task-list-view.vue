<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("DELETE_SHARE_MI_TITLE") }}
        </v-card-title>
        <div>{{ $t("CONFIRM_DELETE_MESSAGE") }}</div>
        <div>{{ share_mi_task_list_info.share_title }}</div>
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary"
                    @click="() => { emits('requested_delete_share_task_link_info', share_mi_task_list_info); emits('requested_close_dialog') }">削除</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteShareTaskListViewEmits } from './confirm-delete-share-task-list-view-emits'
import type { ConfirmDeleteShareTaskListViewProps } from './confirm-delete-share-task-list-view-props'

defineProps<ConfirmDeleteShareTaskListViewProps>()
const emits = defineEmits<ConfirmDeleteShareTaskListViewEmits>()
</script>
