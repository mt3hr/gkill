'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'

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

    async clone(): Promise<IDFKyou> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
        this.file_name = ""
        this.file_url = ""
        this.is_image = false
    }

}


