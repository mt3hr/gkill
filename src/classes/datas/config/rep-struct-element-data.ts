'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"


export class RepStructElementData implements FoldableStructModel {

    id: string

    rep_name: string

    check_when_inited: boolean

    ignore_check_rep_rykv: boolean

    children: Array<RepStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    constructor() {
        this.id = ""
        this.rep_name = ""
        this.check_when_inited = false
        this.ignore_check_rep_rykv = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
    }

}


