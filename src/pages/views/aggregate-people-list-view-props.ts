'use strict'

import type { AggregatePeople } from "@/classes/api/dnote/aggregate-people"
import type { GkillPropsBase } from "./gkill-props-base"

export interface AggregatePeopleListViewProps extends GkillPropsBase {
    last_added_tag: string
    aggregate_peoples: Array<AggregatePeople>
}