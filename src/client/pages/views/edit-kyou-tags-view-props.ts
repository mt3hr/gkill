'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditKyouTagsViewProps extends GkillPropsBase {
    /**
     * タグを付ける対象。
     *
     * 追加画面ではまだKyouが無いので `null` を渡す。そのときこのビューは
     * 新しく付けるタグ名を集めるだけになり、既存タグの読み込みも削除も行わない。
     */
    kyou: Kyou | null
    /** 保存中は入力を触らせない */
    is_readonly: boolean
}
