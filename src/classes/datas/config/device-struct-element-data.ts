'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class DeviceStructElementData implements FoldableStructModel {

    seq_in_parent: number

    id: string | null

    device_name: string

    check_when_inited: boolean

    children: Array<DeviceStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    is_dir: boolean

    is_open_default: boolean

    constructor() {
        this.seq_in_parent = 0
        this.id = ""
        this.device_name = ""
        this.check_when_inited = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
        this.is_dir = false
        this.is_open_default = false
    }

}


