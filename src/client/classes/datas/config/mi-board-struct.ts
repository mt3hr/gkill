'use strict'

export class MiBoardStruct {

    id: string | null

    user_id: string

    device: string

    board_name: string

    parent_folder_id: string | null

    seq: Number

    check_when_inited: boolean

    ignore_check_rep_rykv: boolean

    constructor() {
        this.id = ""
        this.user_id = ""
        this.device = ""
        this.board_name = ""
        this.parent_folder_id = null
        this.seq = 0
        this.check_when_inited = false
        this.ignore_check_rep_rykv = false
    }

}


