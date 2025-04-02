'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class KFTLTemplateStructElementData implements FoldableStructModel {

    seq_in_parent: number

    id: string | null

    title: string

    template: string

    children: Array<KFTLTemplateStructElementData> | null

    key: string

    is_checked: boolean // 使わない

    indeterminate: boolean // 使わない

    is_dir: boolean

    is_open_default: boolean

    constructor() {
        this.seq_in_parent = 0
        this.id = ""
        this.key = ""
        this.title = ""
        this.template = ""
        this.children = null
        this.is_checked = false
        this.indeterminate = false
        this.is_dir = false
        this.is_open_default = false
    }

}


