'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'

export class TimeIs extends InfoBase {

    title: string

    start_time: Date

    end_time: Date

    attached_histories: Array<TimeIs>

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

    async clone(): Promise<TimeIs> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
        this.title = ""
        this.start_time = new Date(0)
        this.end_time = new Date(0)
        this.attached_histories = new Array<TimeIs>()
    }

}


