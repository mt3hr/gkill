// ˅
'use strict';

import { KFTLTemplateDialogEmits } from './kftl-template-dialog-emits';
import { KFTLTemplateDialogProps } from './kftl-template-dialog-props';

// ˄

export class KFTLTemplateDialog {
    // ˅
    
    // ˄

    private child_templates: Ref<Array<KFTLTemplateElement>>;

    private is_show_child_dialog: Ref<boolean>;

    private props: KFTLTemplateDialogProps;

    private kFTLTemplateDialog: Array<KFTLTemplateDialog>;

    private emits: KFTLTemplateDialogEmits;

    private async clicked_template_button(template_element: KFTLTemplateElement): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async emit_clicked_template_element_leaf(template_leaf: KFTLTemplateElement): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

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
