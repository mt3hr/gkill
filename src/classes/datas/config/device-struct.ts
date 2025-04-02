'use strict'

export class DeviceStruct {

    id: string

    user_id: string

    device: string

    device_name: string

    parent_folder_id: string | null

    seq: Number

    check_when_inited: boolean

    is_dir: boolean

    is_open_default: boolean

    constructor() {
        this.id = ""
        this.user_id = ""
        this.device = ""
        this.device_name = ""
        this.parent_folder_id = ""
        this.seq = 0
        this.check_when_inited = false
        this.is_dir = false
        this.is_open_default = false
    }

}


