'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"
import { Text } from "@/classes/datas/text"

export interface AttachedTextContextMenuProps extends GkillPropsBase {
    text: Text
    kyou: Kyou
    last_added_tag: string
}
