<template>
    <DnoteLocationListView />
    <DnoteNlogsListView />
    <DnotePeoplesListView />
</template>
<script setup lang="ts">
import type { DnoteEmits } from './dnote-emits';
import type { DnoteProps } from './dnote-props';

import type { Kmemo } from '@/classes/datas/kmemo';
import type { Lantana } from '@/classes/datas/lantana';
import { TimeIs } from '@/classes/datas/time-is';
import { computed, ref, type Ref } from 'vue';

import DnoteLocationListView from './dnote-location-list-view.vue';
import DnoteNlogsListView from './dnote-nlogs-list-view.vue';
import DnotePeoplesListView from './dnote-peoples-list-view.vue';
import moment from 'moment';

const props = defineProps<DnoteProps>();
const emits = defineEmits<DnoteEmits>();
const start_date = computed(() => { props.query.calendar_start_date ? props.query.calendar_start_date : moment().toDate() });
const end_date = computed(() => { props.query.calendar_end_date ? props.query.calendar_end_date : moment().toDate() });
const date_kmemo0000: Ref<Array<Kmemo>> = ref(new Array<Kmemo>());
const awake_timeiss: Ref<Array<TimeIs>> = ref(new Array<TimeIs>());
const sleep_timeiss: Ref<Array<TimeIs>> = ref(new Array<TimeIs>());
const work_timeiss: Ref<Array<TimeIs>> = ref(new Array<TimeIs>());
const lantanas: Ref<Array<Lantana>> = ref(new Array<Lantana>());
const calclutated_total_awake_time: Ref<string> = ref("");
const calclutated_total_sleep_time: Ref<string> = ref("");
const calclutated_total_work_time: Ref<string> = ref("");
const calclutated_kyou_record_count: Ref<Number> = ref(0);
const calclutated_tabaco_record_count: Ref<Number> = ref(0);
const calclutated_average_lantana_mood: Ref<Number> = ref(0);
</script>