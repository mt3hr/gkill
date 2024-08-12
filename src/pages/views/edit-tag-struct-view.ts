// ˅
'use strict';

import { EditTagStructViewEmits } from './edit-tag-struct-view-emits';
import { EditTagStructViewProps } from './edit-tag-struct-view-props';
import { TagStructElement } from './tag-struct-element';

// ˄

export class EditTagStructView {
    // ˅
    
    // ˄

    private cloned_application_config: Ref<ApplicationConfig>;

    add_new_tag_struct_element_dialog: AddNewTagStructElementDialog;

    edit_tag_struct_element_dialog: EditTagStructElementDialog;

    private props: EditTagStructViewProps;

    private emits: EditTagStructViewEmits;

    private tagStructElement: TagStructElement;

    private tag_struct_root_element: TagStructElement;

    // ˅
    
    // ˄
}

// ˅

// ˄
