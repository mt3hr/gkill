<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { DnoteEmits } from './dnote-emits';
import { DnoteLocationListView } from './dnote-location-list-view.vue';
import { DnoteNlogsListView } from './dnote-nlogs-list-view.vue';
import { DnotePeoplesListView } from './dnote-peoples-list-view.vue';
import { DnoteProps } from './dnote-props';

// ˄

export class Dnote {
    // ˅
    
    // ˄

    private start_date: Ref<Date>;

    private end_date: Ref<Date>;

    private date_kmemo0000: Ref<Array<Kmemo>>;

    private awake_timeiss: Ref<Array<TimeIs>>;

    private sleep_timeiss: Ref<Array<TimeIs>>;

    private work_timeiss: Ref<Array<TimeIs>>;

    private lantanas: Ref<Array<Lantana>>;

    private calclutated_total_awake_time: Ref<string>;

    private calclutated_total_sleep_time: Ref<string>;

    private calclutated_total_work_time: Ref<string>;

    private calclutated_kyou_record_count: Ref<Number>;

    private calclutated_tabaco_record_count: Ref<Number>;

    private calclutated_average_lantana_mood: Ref<Number>;

    private props: DnoteProps;

    private emits: DnoteEmits;

    private nlog_list_view: DnoteNlogsListView;

    private location_list_view: DnoteLocationListView;

    private people_list_view: DnotePeoplesListView;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>