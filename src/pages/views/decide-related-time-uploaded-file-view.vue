<template>
    <EditIDFKyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="[]"
        :kyou="focused_kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" @requested_reload_kyou="(kyou) => { }"
        @requested_reload_list="() => { }" />
    <KyouListView :application_config="application_config" :gkill_api="gkill_api" :matched_kyous="cloned_uploaded_kyous"
        :query="new FindKyouQuery()" @received_errors="(errors) => emits('received_errors', errors)"
        :last_added_tag="last_added_tag" @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => { }" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => { }" />
</template>
<script setup lang="ts">
import type { DecideRelatedTimeUploadedFileViewProps } from './decide-related-time-uploaded-file-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditIDFKyouView from './edit-idf-kyou-view.vue'
import KyouListView from './kyou-list-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import { type Ref, ref } from 'vue'

const props = defineProps<DecideRelatedTimeUploadedFileViewProps>()
const emits = defineEmits<KyouViewEmits>()
const focused_kyou: Ref<Kyou> = ref(new Kyou())

const cloned_uploaded_kyous: Ref<Array<Kyou>> = ref(await props.uploaded_kyous.concat())
</script>
