<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { ConfirmDeleteShareTaskListDialog } from '../dialogs/confirm-delete-share-task-list-dialog.vue';
import { KyouCountCalendar } from './kyou-count-calendar.vue';
import { KyouView } from './kyou-view.vue';
import { ManageShareTaskListDialog } from '../dialogs/manage-share-task-list-dialog.vue';
import { ShareTaskListDialog } from '../dialogs/share-task-list-dialog.vue';
import { ShareTaskListLinkDialog } from '../dialogs/share-task-list-link-dialog.vue';
import { miBoardTaskListView } from './mi-board-task-list-view.vue';
import { miQueryEditorSidebar } from './mi-query-editor-sidebar.vue';
import { miShareFooter } from './mi-share-footer.vue';
import { miViewEmits } from './mi-view-emits';
import { miViewProps } from './mi-view-props';

// ˄

export class miView {
    // ˅
    
    // ˄

    private props: miViewProps;

    private emits: miViewEmits;

    private sidebar: miQueryEditorSidebar;

    private footer: miShareFooter;

    private calendar: KyouCountCalendar;

    private miBoardTaskListView: Array<miBoardTaskListView>;

    private mi_detail_view: KyouView;

    private shareTaskListDialog: ShareTaskListDialog;

    private shareTaskListLinkDialog: ShareTaskListLinkDialog;

    private manageShareTaskListDialog: ManageShareTaskListDialog;

    private confirmDeleteShareTaskListDialog: ConfirmDeleteShareTaskListDialog;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>