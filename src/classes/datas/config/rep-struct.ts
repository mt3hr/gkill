'use strict'

export class RepStruct {

    id: string

    user_id: string

    device: string

    rep_name: string

    parent_folder_id: string | null

    seq: Number

    check_when_inited: boolean

    ignore_check_rep_rykv: boolean

    constructor() {
        this.id = ""
        this.user_id = ""
        this.device = ""
        this.rep_name = ""
        this.parent_folder_id = null
        this.seq = 0
        this.check_when_inited = false
        this.ignore_check_rep_rykv = false
    }

}


