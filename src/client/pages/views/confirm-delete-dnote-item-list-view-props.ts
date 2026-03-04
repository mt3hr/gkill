'use strict'

import type DnoteItem from "@/classes/dnote/dnote-item"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ConfirmDeleteDnoteItemListViewProps extends GkillPropsBase {
    dnote_item: DnoteItem
}
