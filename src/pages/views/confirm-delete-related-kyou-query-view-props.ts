'use strict'

import type RelatedKyouQuery from "@/classes/dnote/related-kyou-query"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ConfirmDeleteRelatedKyouQueryViewProps extends GkillPropsBase {
    related_kyou_query: RelatedKyouQuery
}
