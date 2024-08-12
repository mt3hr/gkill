// ˅
'use strict';

import { AddLantanaView } from '../views/add-lantana-view';
import { LantanaDialogEmits } from './lantana-dialog-emits';
import { LantanaDialogProps } from './lantana-dialog-props';
import { LantanaFlowersView } from '../views/lantana-flowers-view';

// ˄

export class LantanaDialog {
    // ˅
    
    // ˄

    private props: LantanaDialogProps;

    private emits: LantanaDialogEmits;

    private lantana_scene: AddLantanaView;

    private lantanaFlowersView: LantanaFlowersView;

    private async add_text_data(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async update_mood(mood: Number): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async update_text_data(text_data: LantanaTextData): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async delete_text_data(text_data: LantanaTextData): Promise<boolean> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async request_close_dialog(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
