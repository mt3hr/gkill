'use strict'

import type { GkillPropsBase } from "../views/gkill-props-base"

export interface EditSavedFindQueryListDialogProps extends GkillPropsBase {
    app_content_height: number
    app_content_width: number
    query_type: 'rykv' | 'mi'
}
