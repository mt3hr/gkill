<template>
    <ShareTaskListLinkView />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ShareTaskListLinkDialogEmits } from './share-task-list-link-dialog-emits';
import type { ShareTaskListLinkDialogProps } from './share-task-list-link-dialog-props';
import type { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';
import ShareTaskListLinkView from '../views/share-task-list-link-view.vue';

const props = defineProps<ShareTaskListLinkDialogProps>();
const emits = defineEmits<ShareTaskListLinkDialogEmits>();
const cloned_share_mi_task_list_info: Ref<ShareMiTaskListInfo> = ref(await props.share_mi_task_list_info.clone());
const is_share_time_only: Ref<boolean> = ref(!cloned_share_mi_task_list_info.value.is_share_detail);
</script>
