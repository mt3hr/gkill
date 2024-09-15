<template>
    <KyouListView />
</template>
<script setup lang="ts">
import KyouListView from './kyou-list-view.vue';
import type { miBoardTaskListViewEmits } from './mi-board-task-list-view-emits';
import type { miBoardTaskListViewProps } from './mi-board-task-list-view-props';

const props = defineProps<miBoardTaskListViewProps>();
const emits = defineEmits<miBoardTaskListViewEmits>();
</script>
<style scoped></style>
