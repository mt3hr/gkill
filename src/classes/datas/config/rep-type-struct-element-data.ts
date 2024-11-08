'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class RepTypeStructElementData implements FoldableStructModel {

    id: string

    rep_type_name: string

    check_when_inited: boolean

    children: Array<RepTypeStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    constructor() {
        this.id = ""
        this.rep_type_name = ""
        this.check_when_inited = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
    }

}


