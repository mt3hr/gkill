// ˅
'use strict';

import { AddRepViewEmits } from './add-rep-view-emits';
import { AddRepViewProps } from './add-rep-view-props';

// ˄

export class AddRepView {
    // ˅
    
    // ˄

    cloned_server_config: ServerConfig;

    user_id: Ref<string>;

    device: Ref<string>;

    type: Ref<string>;

    file: Ref<string>;

    use_to_write: Ref<boolean>;

    is_execute_idf_when_reload: Ref<boolean>;

    is_enable: Ref<boolean>;

    private props: AddRepViewProps;

    private emits: AddRepViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄
