'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type DnoteListQuery from "./dnote-list-query"

export interface ConfirmDeleteDnoteListQueryViewProps extends GkillPropsBase {
    dnote_list_query: DnoteListQuery
}
