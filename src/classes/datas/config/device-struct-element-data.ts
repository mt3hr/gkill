'use strict'

import type { FoldableStructModel } from "@/pages/views/foldable-struct-model"

export class DeviceStructElementData implements FoldableStructModel {

    id: string | null

    device_name: string

    check_when_inited: boolean

    children: Array<DeviceStructElementData> | null

    key: string

    is_checked: boolean

    indeterminate: boolean

    constructor() {
        this.id = ""
        this.device_name = ""
        this.check_when_inited = false
        this.children = null
        this.key = ""
        this.is_checked = false
        this.indeterminate = false
    }

}


