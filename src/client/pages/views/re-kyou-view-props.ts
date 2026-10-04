'use strict'

import type { ReKyou } from "@/classes/datas/re-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface ReKyouViewProps extends KyouViewPropsBase {
    rekyou: ReKyou
    height: number | string
    width: number | string
    // 参照先の KyouView へそのまま渡す。一覧の行の中でも参照先が原寸の画像や動画のメタデータを読みに行かないように
    is_image_request_to_thumb_size: boolean
}
