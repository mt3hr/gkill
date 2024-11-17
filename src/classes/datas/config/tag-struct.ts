'use strict'

export class TagStruct {
    id: string
    user_id: string
    device: string
    tag_name: string
    parent_folder_id: string | null
    seq: Number
    check_when_inited: boolean
    is_force_hide: boolean
    async clone(): Promise<TagStruct> {
        throw new Error('Not implemented')
    }
    constructor() {
        this.id = ""
        this.user_id = ""
        this.device = ""
        this.tag_name = ""
        this.parent_folder_id = null
        this.seq = 0
        this.check_when_inited = false
        this.is_force_hide = false
    }
}
