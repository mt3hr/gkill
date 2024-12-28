'use strict'

import type { AggregateAmount } from "@/classes/api/dnote/aggregate-amount"
import type { GkillPropsBase } from "./gkill-props-base"

export interface AggregateAmountListViewProps extends GkillPropsBase {
    last_added_tag: string
    aggregate_ammounts: Array<AggregateAmount>
}