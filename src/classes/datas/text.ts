'use strict'

import type { GkillError } from '../api/gkill-error'
import { MetaInfoBase } from './meta-info-base'

export class Text extends MetaInfoBase {

    text: string

    attached_histories: Array<Text>

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

    async clone(): Promise<Text> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
        this.text = ""
        this.attached_histories = new Array<Text>()
    }

}


