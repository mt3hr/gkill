<template>
    <KyouListView :application_config="application_config" :gkill_api="gkill_api" :last_added_tag="last_added_tag"
        :kyou_height="180" :list_height="app_content_height.valueOf() / 2" :width="500" :matched_kyous="matched_kyous"
        :query="query" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import KyouListView from './kyou-list-view.vue'
import type { miBoardTaskListViewEmits } from './mi-board-task-list-view-emits'
import type { miBoardTaskListViewProps } from './mi-board-task-list-view-props'

const props = defineProps<miBoardTaskListViewProps>()
const emits = defineEmits<miBoardTaskListViewEmits>()
</script>
<style scoped></style>
