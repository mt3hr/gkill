'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { SidebarProps } from "./sidebar-props"

export interface rykvQueryEditorSidebarProps extends GkillPropsBase, SidebarProps {
    app_title_bar_height: Number
    app_content_height: Number
    app_content_width: Number
}
