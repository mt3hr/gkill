// ˅
'use strict';

import { AggregateViewEmits } from './aggregate-view-emits';
import { AggregateViewProps } from './aggregate-view-props';

// ˄

export class AggregateView {
    // ˅
    
    // ˄

    private kyous_count: Ref<Number>;

    private nlogs_total_amount: Ref<Number>;

    private nlogs_total_plus_amount: Ref<Number>;

    private nlogs_total_minus_amount: Ref<Number>;

    private timeis_total_time_milli_second: Ref<Number>;

    private git_total_file_count: Ref<Number>;

    private git_total_add_row: Ref<Number>;

    private git_total_remove_row: Ref<Number>;

    private lantanas_average_mood: Ref<Number>;

    private props: AggregateViewProps;

    private emits: AggregateViewEmits;

    private async update_aggregate_view(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
