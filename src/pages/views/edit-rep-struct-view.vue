// ˅
'use strict';

import { EditRepStructViewEmits } from './edit-rep-struct-view-emits';
import { EditRepStructViewProps } from './edit-rep-struct-view-props';
import { RepStructElement } from './rep-struct-element';
import { TagStructElement } from './tag-struct-element';

// ˄

export class EditRepStructView {
    // ˅
    
    // ˄

    private cloned_application_config: Ref<ApplicationConfig>;

    add_new_rep_struct_element_dialog: AddNewRepStructElementDialog;

    edit_rep_struct_element_dialog: EditRepStructElementDialog;

    private props: EditRepStructViewProps;

    private emits: EditRepStructViewEmits;

    private tagStructElement: TagStructElement;

    private rep_struct_root_element: RepStructElement;

    // ˅
    
    // ˄
}

// ˅

// ˄
