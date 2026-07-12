'use strict'

import type { IDFKyou } from "@/classes/datas/idf-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface IDFKyouProps extends KyouViewPropsBase {
    idf_kyou: IDFKyou
    height: number | string
    width: number | string
    is_image_request_to_thumb_size: boolean
    // MarkDown内リンクのダブルクリックでKyouDialogを開いてよいか。
    // 未指定なら enable_dialog に従う。enable_dialog は内側KyouViewのdblclick抑止にも使われているため分けている。
    enable_md_link_dialog?: boolean
}
