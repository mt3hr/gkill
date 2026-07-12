'use strict'

import type { IDFKyou } from "@/classes/datas/idf-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface IDFKyouProps extends KyouViewPropsBase {
    idf_kyou: IDFKyou
    height: number | string
    width: number | string
    is_image_request_to_thumb_size: boolean
    // trueなら enable_dialog=false でもMarkDown内リンクのダブルクリックでKyouDialogを開く。
    // enable_dialog は内側KyouViewのdblclick抑止にも使われているため分けている (ryuu-item-view.vue)。
    enable_md_link_dialog?: boolean
}
