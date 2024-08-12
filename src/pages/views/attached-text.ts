// ˅
'use strict';

import { AttachedTextContextMenu } from './attached-text-context-menu';
import { AttachedTextProps } from './attached-text-props';
import { ConfirmDeleteTextDialog } from '../dialogs/confirm-delete-text-dialog';
import { EditTextDialog } from '../dialogs/edit-text-dialog';
import { KyouViewEmits } from './kyou-view-emits';

// ˄

export class AttachedText {
    // ˅
    
    // ˄

    private cloned_text: Text;

    private props: AttachedTextProps;

    private editTextDialog: EditTextDialog;

    private confirmDeleteTextDialog: ConfirmDeleteTextDialog;

    private contextmenu: AttachedTextContextMenu;

    private emits: KyouViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄
