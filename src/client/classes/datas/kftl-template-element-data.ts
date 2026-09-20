'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class KFTLTemplateElementData implements FoldableStructModel {

    name: string

    id: string | null

    title: string

    template: string

    // 利用者が設定画面で書く運用メモ。MCP（AI連携）が読む。空文字は未記入
    description: string

    children: Array<KFTLTemplateElementData> | null

    key: string

    is_checked: boolean // 使わない
    indeterminate: boolean // 使わない

    is_dir: boolean

    constructor() {
        this.name = ""
        this.id = ""
        this.title = ""
        this.template = ""
        this.description = ""
        this.children = null
        this.is_checked = false
        this.indeterminate = false
        this.key = ""
        this.is_dir = false
    }

}


