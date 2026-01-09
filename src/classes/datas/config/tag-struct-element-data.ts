'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class TagStructElementData implements FoldableStructModel {
    name: string

    id: string | null

    tag_name: string

    check_when_inited: boolean

    is_force_hide: boolean

    children: Array<TagStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    is_dir: boolean

    is_open_default: boolean

    constructor() {
        this.name = ""
        this.id = ""
        this.tag_name = ""
        this.check_when_inited = false
        this.is_force_hide = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
        this.is_dir = false
        this.is_open_default = false
    }

}


