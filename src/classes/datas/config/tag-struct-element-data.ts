'use strict'

export class TagStructElementData {

    id: string

    tag_name: string

    check_when_inited: boolean

    is_force_hide: boolean

    children: Array<TagStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    constructor() {
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


