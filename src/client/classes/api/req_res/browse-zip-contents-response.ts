'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export interface ZipEntry {
    path: string
    is_dir: boolean
    size: number
    is_image: boolean
    file_url: string
}

export class BrowseZipContentsResponse extends GkillAPIResponse {

    entries: ZipEntry[]

    constructor() {
        super()
        this.entries = []
    }

}
