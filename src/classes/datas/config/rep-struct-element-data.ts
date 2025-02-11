'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"


export class RepStructElementData implements FoldableStructModel {

    seq_in_parent: number

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

    constructor() {
        this.seq_in_parent = 0
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
    }

}


