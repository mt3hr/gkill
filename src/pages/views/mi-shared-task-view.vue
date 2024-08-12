<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { KyouCountCalendar } from './kyou-count-calendar.vue';
import { miBoardTaskListView } from './mi-board-task-list-view.vue';
import { miSharedTaskViewEmits } from './mi-shared-task-view-emits';
import { miSharedTaskViewProps } from './mi-shared-task-view-props';

// ˄

export class miSharedTaskView {
    // ˅
    
    // ˄

    private shared_board_task_list_view: miBoardTaskListView;

    private calendar: KyouCountCalendar;

    private props: miSharedTaskViewProps;

    private emits: miSharedTaskViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>