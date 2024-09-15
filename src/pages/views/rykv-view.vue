<template>
    <KyouListView v-for="query, index in querys" />
    <RykvQueryEditorSideBar />
    <KyouView />
    <KyouCountCalendar />
    <GPSLogMap />
    <Dnote />
    <AggregateView />
    <AddMiDialog />
    <AddNlogDialog />
    <EndTimeIsPlaingDialog />
    <StartTimeIsDialog />
    <kftlDialog />
</template>
<script setup lang="ts">

// ˅
import { type Ref, ref } from 'vue';
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { Kyou } from '@/classes/datas/kyou';
import AddMiDialog from '../dialogs/add-mi-dialog.vue';
import AddNlogDialog from '../dialogs/add-nlog-dialog.vue';
import AggregateView from './aggregate-view.vue';
import Dnote from './dnote.vue';
import EndTimeIsPlaingDialog from '../dialogs/end-time-is-plaing-dialog.vue';
import GPSLogMap from './gps-log-map.vue';
import KyouCountCalendar from './kyou-count-calendar.vue';
import KyouListView from './kyou-list-view.vue';
import KyouView from './kyou-view.vue';
import StartTimeIsDialog from '../dialogs/start-time-is-dialog.vue';
import RykvQueryEditorSideBar from './rykv-query-editor-side-bar.vue';
import kftlDialog from '../dialogs/kftl-dialog.vue';
import type { rykvViewEmits } from './rykv-view-emits';
import type { rykvViewProps } from './rykv-view-props';

const querys: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>());
const focused_kyou: Ref<Kyou> = ref(new Kyou());
const focused_list_views_kyous: Array<Kyou> = ref(new Array<Kyou>());
const focused_time: Ref<Date> = ref(new Date());
const focused_column_kyous: Array<Kyou> = ref(new Array<Kyou>());
const focused_column_checked_kyous: Array<Kyou> = ref(new Array<Kyou>());
const is_show_kyou_detail_view: Ref<boolean> = ref(false);
const is_show_kyou_count_calendar: Ref<boolean> = ref(false);
const is_show_gps_log_map: Ref<boolean> = ref(false);
const props = defineProps<rykvViewProps>();
const emits = defineEmits<rykvViewEmits>();

async function add_list_view(): Promise<void> {
    throw new Error('Not implemented');
}
async function close_list_view(query_index: Number): Promise<void> {
    throw new Error('Not implemented');
}
async function update_queries(query_index: Number, by_user: boolean): Promise<void> {
    throw new Error('Not implemented');
}
async function update_kyous(column_index: Number, kyous: Array<Kyou>): Promise<void> {
    throw new Error('Not implemented');
}
</script>