'use strict'

import type { MiReKyou } from "@/classes/datas/mi-re-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface MiReKyouViewProps extends KyouViewPropsBase {
    mirekyou: MiReKyou
    is_readonly_mi_check: boolean
    height: number | string
    width: number | string
    // 参照先の KyouView へそのまま渡す。一覧の行の中でも参照先が原寸の画像や動画のメタデータを読みに行かないように
    is_image_request_to_thumb_size: boolean
}
