'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"


export class RepStructElementData implements FoldableStructModel {
    name: string

    id: string

    rep_name: string

    check_when_inited: boolean

    ignore_check_rep_rykv: boolean

    // 利用者が設定画面で書く運用メモ。MCP（AI連携）が読む。空文字は未記入
    description: string

    children: Array<RepStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    is_dir: boolean

    constructor() {
        this.name = ""
        this.id = ""
        this.rep_name = ""
        this.check_when_inited = false
        this.ignore_check_rep_rykv = false
        this.description = ""
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
        this.is_dir = false
    }

}
