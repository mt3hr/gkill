// ˅
'use strict';

import { FoldableStructEmits } from './foldable-struct-emits';
import { FoldableStructProps } from './foldable-struct-props';
import { GkillPropsBase } from './gkill-props-base';

// ˄

export class FoldableStruct implements GkillPropsBase {
    // ˅
    
    // ˄

    private cloned_is_open: Ref<boolean>;

    private check: Ref<CheckState>;

    private struct_list: Ref<any>;

    private props: FoldableStructProps;

    private emits: FoldableStructEmits;

    async get_selected_items(): Promise<Array<string>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
