<template>
    <KyouView v-for="kyou, index in match_kyous" :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[]" :is_image_view="false" :kyou="kyou" :last_added_tag="last_added_tag"
        :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
        :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import type { KyouListViewEmits } from './kyou-list-view-emits'
import type { KyouListViewProps } from './kyou-list-view-props'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { GkillError } from '@/classes/api/gkill-error'
import { Kyou } from '@/classes/datas/kyou'
import { type Ref, ref } from 'vue'
import KyouView from './kyou-view.vue'

const props = defineProps<KyouListViewProps>()
const emits = defineEmits<KyouListViewEmits>()
const cloned_find_query: Ref<FindKyouQuery> = ref(await props.query.clone())
const match_kyous: Ref<Array<Kyou>> = ref(props.matched_kyous.concat())
const is_loading: Ref<boolean> = ref(false)
const scroll_distance_from_top_px: Ref<Number> = ref(0)
const checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())

async function reload(): Promise<Array<GkillError>> {
    throw new Error('Not implemented')
}
async function scroll_to_kyou(kyou: Kyou): Promise<boolean> {
    throw new Error('Not implemented')
}
async function scroll_to_time(time: Date): Promise<boolean> {
    throw new Error('Not implemented')
}
</script>
