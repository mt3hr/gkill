'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

export interface MKFLProps extends GkillPropsBase {
    app_content_height: number
    app_content_width: number
    // 打刻メモ帳ダイアログ、またはポート(rudbeckia)のダイアログの中で描かれている。
    // 内包している PlaingTimeIsView へそのまま渡す。
    // `/mkfl` ページでは PlaingTimeIsView のFABがその画面の唯一のFABなので false、
    // ダイアログの中では呼び出し元のページが自前のFABを持っているので true にして重なりを避ける
    is_hosted_in_dialog: boolean
}
