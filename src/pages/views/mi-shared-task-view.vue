<template>
    <KyouCountCalendar />
    <miBoardTaskListView />
</template>
<script setup lang="ts">
import type { miSharedTaskViewEmits } from './mi-shared-task-view-emits';
import type { miSharedTaskViewProps } from './mi-shared-task-view-props';

import KyouCountCalendar from './kyou-count-calendar.vue';
import miBoardTaskListView from './mi-board-task-list-view.vue';

const props = defineProps<miSharedTaskViewProps>();
const emits = defineEmits<miSharedTaskViewEmits>();

</script>
<style scoped></style>