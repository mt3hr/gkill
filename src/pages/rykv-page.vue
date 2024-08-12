// ˅
'use strict';

import { ApplicationConfigDialog } from './dialogs/application-config-dialog';
import { UploadFileDialog } from './dialogs/upload-file-dialog';
import { rykvView } from './views/rykv-view';

// ˄

export class rykvPage {
    // ˅
    
    // ˄

    private actual_height: Ref<Number>;

    private element_height: Ref<Number>;

    private browser_url_bar_height: Ref<Number>;

    private app_title_bar_height: Ref<Number>;

    private application_config: ApplicationConfig;

    private gkill_api: GkillAPI;

    private app_content_height: Ref<Number>;

    private app_content_width: Ref<Number>;

    private rykvView: rykvView;

    private uploadFileDialog: UploadFileDialog;

    private application_config_dialog: ApplicationConfigDialog;

    async resize(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async reload_application_config(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async apply_application_config(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async print_message(message: string): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
