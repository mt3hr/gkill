'use strict'

import type { AggregateLocation } from "@/classes/api/dnote/aggregate-location"
import type { GkillPropsBase } from "./gkill-props-base"

export interface AggregateLocationViewProps extends GkillPropsBase {
    last_added_tag: string
    aggregate_location: AggregateLocation
}
