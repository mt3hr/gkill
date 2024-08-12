// ˅
'use strict';

import { EditRepTypeViewEmits } from './edit-rep-type-view-emits';
import { EditRepTypeViewProps } from './edit-rep-type-view-props';
import { RepTypeElement } from './rep-type-element';
import { TagStructElement } from './tag-struct-element';

// ˄

export class EditRepTypeView {
    // ˅
    
    // ˄

    private cloned_application_config: Ref<ApplicationConfig>;

    add_new_rep_type_struct_element_dialog: AddNewRepTypeStructElementDialog;

    edit_rep_type_struct_element_dialog: EditRepTypeStructElementDialog;

    private props: EditRepTypeViewProps;

    private emits: EditRepTypeViewEmits;

    private tagStructElement: TagStructElement;

    private rep_type_root_element: RepTypeElement;

    // ˅
    
    // ˄
}

// ˅

// ˄
