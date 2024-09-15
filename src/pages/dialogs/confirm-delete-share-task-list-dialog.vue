<template>
    <ConfirmDeleteShareTaskListView />
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ConfirmDeleteShareTaskLinkDialogEmits } from './confirm-delete-share-task-link-dialog-emits';
import type { ConfirmDeleteShareTaskLinkDialogProps } from './confirm-delete-share-task-link-dialog-props';
import type { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';
import ConfirmDeleteShareTaskListView from '../views/confirm-delete-share-task-list-view.vue';

const props = defineProps<ConfirmDeleteShareTaskLinkDialogProps>();
const emits = defineEmits<ConfirmDeleteShareTaskLinkDialogEmits>();

const cloned_share_mi_task_list_info: Ref<ShareMiTaskListInfo> = ref(await props.share_mi_task_list_info.clone());
</script>
