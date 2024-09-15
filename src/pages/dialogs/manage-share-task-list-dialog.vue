<template>
    <ManageShareTaskListView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ManageShareTaskLinkDialogEmits } from './manage-share-task-link-dialog-emits';
import type { ManageShareTaskLinkDialogProps } from './manage-share-task-link-dialog-props';
import ManageShareTaskListView from '../views/manage-share-task-list-view.vue';
import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';

const props = defineProps<ManageShareTaskLinkDialogProps>();
const emits = defineEmits<ManageShareTaskLinkDialogEmits>();
const cloned_share_mi_task_list_infos: Ref<Array<ShareMiTaskListInfo>> = ref(new Array<ShareMiTaskListInfo>());
</script>
