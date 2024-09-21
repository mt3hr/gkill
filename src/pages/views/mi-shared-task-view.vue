<template>
    <KyouCountCalendar :application_config="application_config" :gkill_api="gkill_api" :kyous="match_kyous"
        @requested_focus_time="(time) => focused_time = time" />
    <miBoardTaskListView :application_config="application_config" :gkill_api="gkill_api" :is_show_close_button="false"
        :last_added_tag="''" :matched_kyous="match_kyous" :query="find_mi_kyou_query"
        @requested_focus_kyou="(kyou) => update_focused_kyou(kyou)" />
</template>
<script setup lang="ts">
import type { miSharedTaskViewEmits } from './mi-shared-task-view-emits'
import type { miSharedTaskViewProps } from './mi-shared-task-view-props'

import KyouCountCalendar from './kyou-count-calendar.vue'
import miBoardTaskListView from './mi-board-task-list-view.vue'
import type { Kyou } from '@/classes/datas/kyou'
import { type Ref, ref } from 'vue'
import { FindMiQuery } from '@/classes/api/find_query/find-mi-query'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'

const props = defineProps<miSharedTaskViewProps>()
const emits = defineEmits<miSharedTaskViewEmits>()

const find_mi_query: Ref<FindMiQuery> = ref(new FindMiQuery())
const find_mi_kyou_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
const match_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
const focused_time: Ref<Date> = ref(new Date())

async function update_focused_kyou(kyou: Kyou): Promise<void> {
    throw new Error('Not implemented')
}
</script>
