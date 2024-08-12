// ˅
'use strict';

import { ConfirmDeleteShareTaskListDialog } from '../dialogs/confirm-delete-share-task-list-dialog';
import { KyouCountCalendar } from './kyou-count-calendar';
import { KyouView } from './kyou-view';
import { ManageShareTaskListDialog } from '../dialogs/manage-share-task-list-dialog';
import { ShareTaskListDialog } from '../dialogs/share-task-list-dialog';
import { ShareTaskListLinkDialog } from '../dialogs/share-task-list-link-dialog';
import { miBoardTaskListView } from './mi-board-task-list-view';
import { miQueryEditorSidebar } from './mi-query-editor-sidebar';
import { miShareFooter } from './mi-share-footer';
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
