'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class TagStructElementData implements FoldableStructModel {

    seq_in_parent: number

    id: string | null

    tag_name: string

    check_when_inited: boolean

    is_force_hide: boolean

    children: Array<TagStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    constructor() {
        this.seq_in_parent = 0
        this.id = ""
        this.tag_name = ""
        this.check_when_inited = false
        this.is_force_hide = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
    }

}


