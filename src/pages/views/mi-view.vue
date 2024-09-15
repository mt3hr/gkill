<template>
    <ConfirmDeleteShareTaskListDialog />
    <KyouCountCalendar />
    <KyouView />
    <ManageShareTaskListDialog />
    <ShareTaskListDialog />
    <ShareTaskListLinkDialog />
    <MiBoardTaskListView v-for="find_mi_query, index in find_mi_queries" />
    <miQueryEditorSidebar />
    <miShareFooter />
</template>
<script setup lang="ts">
import ConfirmDeleteShareTaskListDialog from '../dialogs/confirm-delete-share-task-list-dialog.vue';
import KyouCountCalendar from './kyou-count-calendar.vue';
import KyouView from './kyou-view.vue';
import ManageShareTaskListDialog from '../dialogs/manage-share-task-list-dialog.vue';
import ShareTaskListDialog from '../dialogs/share-task-list-dialog.vue';
import ShareTaskListLinkDialog from '../dialogs/share-task-list-link-dialog.vue';
import MiBoardTaskListView from './mi-board-task-list-view.vue';
import miQueryEditorSidebar from './mi-query-editor-sidebar.vue';
import miShareFooter from './mi-share-footer.vue';
import type { miViewEmits } from './mi-view-emits';
import type { miViewProps } from './mi-view-props';
import type { FindMiQuery } from '@/classes/api/find_query/find-mi-query';
import { ref, type Ref } from 'vue';

const find_mi_queries: Ref<Array<FindMiQuery>> = ref(new Array<FindMiQuery>())

const props = defineProps<miViewProps>();
const emits = defineEmits<miViewEmits>();

</script>
