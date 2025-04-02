'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class RepTypeStructElementData implements FoldableStructModel {

    seq_in_parent: number

    id: string | null

    rep_type_name: string

    check_when_inited: boolean

    children: Array<RepTypeStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    is_dir: boolean

    is_open_default: boolean

    constructor() {
        this.seq_in_parent = 0
        this.id = ""
        this.rep_type_name = ""
        this.check_when_inited = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
        this.is_dir = false
        this.is_open_default = false
    }

}


