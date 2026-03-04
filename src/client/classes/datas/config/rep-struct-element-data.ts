'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"


export class RepStructElementData implements FoldableStructModel {
    name: string

    id: string

    rep_name: string

    check_when_inited: boolean

    ignore_check_rep_rykv: boolean

    children: Array<RepStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    parent_folder_id: string | null

    seq: number

    is_dir: boolean

    is_open_default: boolean

    constructor() {
        this.name = ""
        this.id = ""
        this.rep_name = ""
        this.check_when_inited = false
        this.ignore_check_rep_rykv = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
        this.parent_folder_id = ""
        this.seq = 0
        this.is_dir = false
        this.is_open_default = false
    }

}


