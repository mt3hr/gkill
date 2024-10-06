'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class URLog extends InfoBase {

    url: string

    title: string

    description: string

    favicon_image: string

    thumbnail_image: string

    attached_histories: Array<URLog>

    async load_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async clone(): Promise<URLog> {
        throw new Error('Not implemented')
    }

    generate_info_identifer(): InfoIdentifier {
        const info_identifer = new InfoIdentifier()
        info_identifer.id = this.id
        info_identifer.create_time = this.create_time
        info_identifer.update_time = this.update_time
        return info_identifer
    }

    constructor() {
        super()
        this.url = ""
        this.title = ""
        this.description = ""
        this.favicon_image = ""
        this.thumbnail_image = ""
        this.attached_histories = new Array<URLog>()
    }

}


