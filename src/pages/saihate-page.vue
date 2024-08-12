// ˅
'use strict';

import { KFTLDialog } from './dialogs/kftl-dialog';
import { LantanaDialog } from './dialogs/lantana-dialog';

// ˄

export class SaihatePage {
    // ˅
    
    // ˄

    private gkill_api: GkillAPI;

    private application_config: ApplicationConfig;

    private actual_height: Ref<Number>;

    private element_height: Ref<Number>;

    private browser_url_bar_height: Ref<Number>;

    private app_title_bar_height: Ref<Number>;

    private app_content_height: Ref<Number>;

    private app_content_width: Ref<Number>;

    private kftl_dialog: KFTLDialog;

    private lantana_dialog: LantanaDialog;

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

    // ˅
    
    // ˄
}

// ˅

// ˄
