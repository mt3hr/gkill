<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { AddMiDialog } from '../dialogs/add-mi-dialog';
import { AddNlogDialog } from '../dialogs/add-nlog-dialog';
import { AggregateView } from './aggregate-view';
import { Dnote } from './dnote';
import { EndTimeIsPlaingDialog } from '../dialogs/end-time-is-plaing-dialog';
import { GPSLogMap } from './gps-log-map';
import { KyouCountCalendar } from './kyou-count-calendar';
import { KyouListView } from './kyou-list-view';
import { KyouView } from './kyou-view';
import { StartTimeIsDialog } from '../dialogs/start-time-is-dialog';
import { rykvQueryEditorSideBar } from './rykv-query-editor-side-bar';
import { rykvViewEmits } from './rykv-view-emits';
import { rykvViewProps } from './rykv-view-props';

// ˄

export class rykvView {
    // ˅
    
    // ˄

    private querys: Ref<Array<FindKyouQuery>>;

    private focused_kyou: Ref<Kyou>;

    private focused_list_views_kyous: Array<Kyou>;

    private focused_time: Ref<Date>;

    private focused_column_kyous: Array<Kyou>;

    private focused_column_checked_kyous: Array<Kyou>;

    private is_show_kyou_detail_view: Ref<boolean>;

    private is_show_kyou_count_calendar: Ref<boolean>;

    private is_show_gps_log_map: Ref<boolean>;

    private props: rykvViewProps;

    private emits: rykvViewEmits;

    private kyouListView: Array<KyouListView>;

    private rykvQueryEditorSideBar: rykvQueryEditorSideBar;

    private kyouView: KyouView;

    private kyou_detail_view: KyouView;

    private kyou_count_calendar: KyouCountCalendar;

    private gps_log_map: GPSLogMap;

    private dnote: Array<Dnote>;

    private dnote: Dnote;

    private aggregate_view: AggregateView;

    private addMiDialog: AddMiDialog;

    private addNlogDialog: AddNlogDialog;

    private endTimeIsPlaingDialog: EndTimeIsPlaingDialog;

    private startTimeIsDialog: StartTimeIsDialog;

    async add_list_view(): void {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async close_list_view(query_index: Number): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async update_queries(query_index: Number, by_user: boolean): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async update_kyous(column_index: Number, kyous: Array<Kyou>): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>