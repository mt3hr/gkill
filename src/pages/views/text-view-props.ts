'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"
import { Text } from "@/classes/datas/text"

export interface TextViewProps extends GkillPropsBase {
    text: Text
    last_added_tag: string
    kyou: Kyou
}
