// ˅
'use strict';

import { ApplicationConfigViewEmits } from './application-config-view-emits';
import { ApplicationConfigViewProps } from './application-config-view-props';
import { EditDeviceStructDialog } from '../dialogs/edit-device-struct-dialog';
import { EditKFTLTemplateDialog } from '../dialogs/edit-kftl-template-dialog';
import { EditRepStructDialog } from '../dialogs/edit-rep-struct-dialog';
import { EditRepTypeDialog } from '../dialogs/edit-rep-type-dialog';
import { EditTagStructDialog } from '../dialogs/edit-tag-struct-dialog';

// ˄

export class ApplicationConfigView {
    // ˅
    
    // ˄

    private cloned_application_config: Ref<ApplicationConfig>;

    private google_map_api_key: Ref<string>;

    private number_of_rykv_columns: Ref<Number>;

    private default_board_name_of_mi: Ref<string>;

    private is_enable_browser_cache: Ref<boolean>;

    private is_enable_hot_reload_rykv: Ref<boolean>;

    private props: ApplicationConfigViewProps;

    private emits: ApplicationConfigViewEmits;

    private tag_struct_dialog: EditTagStructDialog;

    private rep_struct_dialog: EditRepStructDialog;

    private device_struct_dialog: EditDeviceStructDialog;

    private rep_type_dialog: EditRepTypeDialog;

    private kftl_template_dialog: EditKFTLTemplateDialog;

    async apply_application_config(): Promise<GkillError> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
