'use strict';

export interface ApplicationConfigStructContextMenuEmits {
    (e: 'requested_show_edit_dialog', struct_obj: Object): void
    (e: 'requested_show_delete_dialog', struct_obj: Object): void
}
