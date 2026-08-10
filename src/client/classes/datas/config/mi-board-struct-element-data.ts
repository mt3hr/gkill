'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"


export class MiBoardStructElementData implements FoldableStructModel {

    name: string

    id: string | null

    board_name: string

    check_when_inited: boolean

    children: Array<MiBoardStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    is_dir: boolean

    constructor() {
        this.name = ""
        this.id = ""
        this.board_name = ""
        this.check_when_inited = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
        this.is_dir = false
    }
}
