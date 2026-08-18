'use strict'

import type { GkillPropsBase } from "@/pages/views/gkill-props-base"
import type { RudbeckiaPageKind } from "@/pages/views/rudbeckia-page-kind"
import type { KyouChangeBus } from "@/classes/kyou-change-bus"

export interface RudbeckiaPageDialogProps extends GkillPropsBase {
    /** 中に描く画面 */
    kind: RudbeckiaPageKind
    /**
     * 位置・サイズの保存キーを決める番号（種類ごとの採番）。
     * 同じキーで2枚出すと `${key}:pos` / `:size` を奪い合うので、必ず分ける
     */
    slot_index: number
    /** 中央からずらす段数（種類をまたいだ採番）。重なりを避けるためだけに使う */
    cascade_index: number
    /** ポートの内容領域。ダイアログの既定サイズを決めるためだけに使う */
    app_content_height: number
    app_content_width: number
    application_config_load_failed: boolean
    /** 画面間の変更通知バス。ポートのページが1つだけ作って全ウィンドウへ配る */
    kyou_change_bus: KyouChangeBus | null
    /** このウィンドウの id。自分が出した通知を受けないための印になる */
    origin_id: string
}
