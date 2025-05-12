<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("DELETE_SHARE_MI_TITLE") }}
        </v-card-title>
        <div>{{ $t("CONFIRM_DELETE_MESSAGE") }}</div>
        <div>{{ share_kyou_list_info.share_title }}</div>
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary"
                    @click="() => { emits('requested_delete_share_kyou_link_info', share_kyou_list_info); emits('requested_close_dialog') }">削除</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteShareKyousListViewEmits } from './confirm-delete-share-task-list-view-emits'
import type { ConfirmDeleteShareKyousListViewProps } from './confirm-delete-share-task-list-view-props'

defineProps<ConfirmDeleteShareKyousListViewProps>()
const emits = defineEmits<ConfirmDeleteShareKyousListViewEmits>()
</script>
