// ˅
'use strict';

import { GkillPropsBase } from './gkill-props-base';
import { KyouListViewEmits } from './kyou-list-view-emits';
import { KyouListViewProps } from './kyou-list-view-props';
import { KyouView } from './kyou-view';

// ˄

export class KyouListView implements GkillPropsBase {
    // ˅
    
    // ˄

    private cloned_find_query: FindKyouQuery;

    private is_loading: boolean;

    private scroll_distance_from_top_px: Number;

    private match_kyous: Array<Kyou>;

    private checked_kyous: Array<Kyou>;

    private props: KyouListViewProps;

    private emits: KyouListViewEmits;

    private kyouView: Array<KyouView>;

    async reload(): Promise<Array<GkillError>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async scroll_to_kyou(kyou: Kyou): Promise<boolean> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async scroll_to_time(time: Date): Promise<boolean> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
