// ˅
'use strict';

import { EditKFTLTemplateViewEmits } from './edit-kftl-template-view-emits';
import { EditKFTLTemplateViewProps } from './edit-kftl-template-view-props';
import { KFTLTemplateElement } from './kftl-template-element';
import { TagStructElement } from './tag-struct-element';

// ˄

export class EditKFTLTemplateView {
    // ˅
    
    // ˄

    private cloned_application_config: Ref<ApplicationConfig>;

    add_new_kftl_template_struct_element_dialog: AddNewKFTLTemplateStructElementDialog;

    edit_kftl_template_struct_element_dialog: EditKFTLTemplateStructElementDialog;

    private props: EditKFTLTemplateViewProps;

    private emits: EditKFTLTemplateViewEmits;

    private tagStructElement: TagStructElement;

    private kftl_template_root_element: KFTLTemplateElement;

    // ˅
    
    // ˄
}

// ˅

// ˄
