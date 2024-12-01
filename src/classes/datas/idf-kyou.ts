'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class IDFKyou extends InfoBase {

    file_name: string

    file_url: string

    is_image: boolean

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

    clone(): IDFKyou {
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
        this.file_name = ""
        this.file_url = ""
        this.is_image = false
    }

}


