'use strict'

export class FileData {

    file_name: string

    data_base64: string

    last_modified: Date

    constructor() {
        this.file_name = ""
        this.data_base64 = ""
        this.last_modified = new Date(0)
    }

}


