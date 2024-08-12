// ˅
'use strict';

import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouView } from '../views/kyou-view';
import { KyouViewEmits } from '../views/kyou-view-emits';
import { TextHistoriesDialogProps } from './text-histories-dialog-props';

// ˄

export class TextHistoriesDialog {
    // ˅
    
    // ˄

    private cloned_text: Text;

    private cloned_kyou: Kyou;

    kyou_view: KyouView;

    text_histories_view: TextHistoriesView;

    private props: TextHistoriesDialogProps;

    private kyouView: KyouView;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

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
