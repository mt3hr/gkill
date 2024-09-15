<template>
    <ShareTaskListView />
</template>
<script lang="ts" setup>
import type { ShareTaskListDialogEmits } from './share-task-list-dialog-emits';
import type { ShareTaskListDialogProps } from './share-task-list-dialog-props';
import ShareTaskListView from '../views/share-task-list-view.vue';
import { ref, type Ref } from 'vue';

const props = defineProps<ShareTaskListDialogProps>();
const emits = defineEmits<ShareTaskListDialogEmits>();
const is_share_time_only: Ref<boolean> = ref(true);
</script>
