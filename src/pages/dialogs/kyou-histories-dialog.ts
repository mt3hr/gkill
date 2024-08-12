// ˅
'use strict';

import { KyouDialogEmits } from '../views/kyou-dialog-emits';
import { KyouHistoriesDialogProps } from './kyou-histories-dialog-props';
import { KyouHistoriesView } from '../views/kyou-histories-view';
import { KyouView } from '../views/kyou-view';
import { KyouViewEmits } from '../views/kyou-view-emits';

// ˄

export class KyouHistoriesDialog {
    // ˅
    
    // ˄

    private props: KyouHistoriesDialogProps;

    private kyouView: Array<KyouView>;

    private emits: KyouViewEmits;

    private emits: KyouDialogEmits;

    private kyouHistoriesView: KyouHistoriesView;

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
