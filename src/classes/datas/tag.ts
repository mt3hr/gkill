'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoIdentifier } from './info-identifier'
import { MetaInfoBase } from './meta-info-base'

export class Tag extends MetaInfoBase {

    tag: string

    attached_histories: Array<Tag>

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

    async clone(): Promise<Tag> {
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
        this.tag = ""
        this.attached_histories = new Array<Tag>()
    }

}


