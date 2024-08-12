// ˅
'use strict';

import { KFTLLineLabel } from './kftl-line-label';
import { KFTLProps } from './kftl-props';
import { KFTLTemplateDialog } from '../dialogs/kftl-template-dialog';
import { KFTLViewEmits } from './kftl-view-emits';
import { KyouListView } from './kyou-list-view';
import { textarea } from '../../../04_クラスモデル_フロント/lang/HTML/textarea';

// ˄

export class KFTLView {
    // ˅
    
    // ˄

    private text_area_content: Ref<string>;

    private text_area_width: Ref<string>;

    private text_area_height: Ref<string>;

    private line_label_width: Ref<string>;

    private line_label_height: Ref<string>;

    private line_label_datas: Ref<Array<LineLabelData>>;

    private line_label_styles: Ref<Array<string>>;

    private invalid_line_numbers: Ref<Array<Number>>;

    private is_requested_submit: Ref<boolean>;

    private find__kyou_query_plaing_timeis: Ref<FindKyouQuery>;

    private props: KFTLProps;

    private kFTLLineLabel: Array<KFTLLineLabel>;

    private kftl_text_area: textarea;

    private kFTLProps: KFTLProps;

    private kFTLProps: KFTLProps;

    private kftl_template_dialog: KFTLTemplateDialog;

    private emits: KFTLViewEmits;

    private plaing_timeis_view: KyouListView;

    private async restore_content_from_localstorage(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async save_content_to_localstorage(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async update_line_labels(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async is_invalid_line(line_index: Number): Promise<boolean> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async submit(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async clear(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async show_kftl_template_dialog(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async resize(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async apply_application_config(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async update_plaing_timeis_kyous(): Promise<GkillError> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    private async request_close_dialog(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
