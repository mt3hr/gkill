// ˅
'use strict';

import { AddTagDialogProps } from './add-tag-dialog-props';
import { AddTagView } from '../views/add-tag-view';
import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class AddTagDialog {
    // ˅
    
    // ˄

    private props: AddTagDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private addTagView: AddTagView;

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
