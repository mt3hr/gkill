'use strict'

import type { AggregateLocation } from "@/classes/api/dnote/aggregate-location"
import type { GkillPropsBase } from "./gkill-props-base"

export interface AggregateLocationListViewProps extends GkillPropsBase {
    last_added_tag: string
    aggregate_locations: Array<AggregateLocation>
}