'use strict'

import type { GkillPropsBase } from "../views/gkill-props-base"

export interface KFTLDialogProps extends GkillPropsBase {

    app_content_height: number
    app_content_width: number

    /**
     * 何枚目のメモ帳ウィンドウか（0 始まり）。
     *
     * 位置・サイズの保存キーと、中央からずらす量をこれで決める。
     * 同じキーのまま複数枚出すと、全部が同じ座標に完全に重なり、
     * ドラッグやリサイズで互いの保存値を上書きし合う
     */
    slot_index: number
}
