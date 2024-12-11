'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface KyouViewProps extends KyouViewPropsBase {
    kyou: Kyou
    is_image_view: boolean
    show_content_only: boolean
    show_checkbox: boolean
    show_mi_create_time: boolean
    show_mi_limit_time: boolean
    show_mi_estimate_start_time: boolean
    show_mi_estimate_end_time: boolean
    show_timeis_plaing_end_button: boolean
    is_readonly_mi_check: boolean
    highlight_targets: Array<InfoIdentifier>
    height: number | string
    width: number | string
}
