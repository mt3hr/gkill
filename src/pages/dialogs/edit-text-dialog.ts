// ˅
'use strict';

import { EditTextDialogProps } from './edit-text-dialog-props';
import { EditTextView } from '../views/edit-text-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class EditTextDialog {
    // ˅
    
    // ˄

    private props: EditTextDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private editTextView: EditTextView;

    async show(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async hide(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
