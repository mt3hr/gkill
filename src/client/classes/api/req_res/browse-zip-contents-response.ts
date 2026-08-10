'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

export interface ZipEntry {
    path: string
    is_dir: boolean
    size: number
    is_image: boolean
    is_text: boolean
    is_video: boolean
    is_audio: boolean
    is_pdf: boolean
    file_url: string
}

export class BrowseZipContentsResponse extends GkillAPIResponse {

    entries: ZipEntry[]

    constructor() {
        super()
        this.entries = []
    }

}
