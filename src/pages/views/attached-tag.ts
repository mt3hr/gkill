// ˅
'use strict';

import { AttachedTagContextMenu } from './attached-tag-context-menu';
import { AttachedTagProps } from './attached-tag-props';
import { ConfirmDeleteTagDialog } from '../dialogs/confirm-delete-tag-dialog';
import { EditTagDialog } from '../dialogs/edit-tag-dialog';
import { KyouViewEmits } from './kyou-view-emits';

// ˄

export class AttachedTag {
    // ˅
    
    // ˄

    private cloned_tag: Tag;

    private props: AttachedTagProps;

    private editTagDialog: EditTagDialog;

    private confirmDeleteTagDialog: ConfirmDeleteTagDialog;

    private contextmenu: AttachedTagContextMenu;

    private emits: KyouViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄
