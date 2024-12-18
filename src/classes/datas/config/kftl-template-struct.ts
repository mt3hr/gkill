'use strict'

export class KFTLTemplateStruct {

    id: string

    user_id: string

    device: string

    title: string

    template: string | null

    parent_folder_id: string | null

    seq: Number

    constructor() {
        this.id = ""
        this.user_id = ""
        this.device = ""
        this.title = ""
        this.template = ""
        this.parent_folder_id = ""
        this.seq = 0
    }

}


