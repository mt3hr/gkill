'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { miSidebarProps } from "./mi-sidebar-props"

export interface miQueryEditorSidebarProps extends miSidebarProps {
    app_title_bar_height: Number
    app_content_height: Number
    app_content_width: Number
}
