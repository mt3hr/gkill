<template>
    <EditIDFKyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :kyou="focused_kyou" :last_added_tag="last_added_tag" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_errors="(errors) => emits('received_errors', errors)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
    <KyouListView :application_config="application_config" :gkill_api="gkill_api" :matched_kyous="uploaded_kyous"
        :width="180" :list_height="app_content_height" :kyou_height="180" :query="new FindKyouQuery()"
        :is_focused_list="false" :closable="false" @received_errors="(errors) => emits('received_errors', errors)"
        :last_added_tag="last_added_tag" :is_readonly_mi_check="true" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_messages="(messages) => emits('received_messages', messages)"
        :scroll_top="0 /* //TODO */" :show_checkbox="false" :show_footer="false" :show_content_only="false"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script setup lang="ts">
import type { DecideRelatedTimeUploadedFileViewProps } from './decide-related-time-uploaded-file-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditIDFKyouView from './edit-idf-kyou-view.vue'
import KyouListView from './kyou-list-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import { type Ref, ref } from 'vue'

const enable_context_menu = ref(true)
const enable_dialog = ref(true)

defineProps<DecideRelatedTimeUploadedFileViewProps>()
const emits = defineEmits<KyouViewEmits>()
const focused_kyou: Ref<Kyou> = ref(new Kyou())
</script>
