// ˅
'use strict';

import { ApplicationConfigDialog } from '../dialogs/application-config-dialog';
import { KyouView } from './kyou-view';

// ˄

export class KyouPage {
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

    private kyou: KyouView;

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
