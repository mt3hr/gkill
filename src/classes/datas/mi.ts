'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'

export class Mi extends InfoBase {

    title: string

    is_checked: boolean

    board_name: string

    limit_time: Date

    estimate_start_time: Date

    estimate_end_time: Date

    attached_histories: Array<Mi>

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

    async clone(): Promise<Mi> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
        this.title = ""

        this.is_checked = false

        this.board_name = ""

        this.limit_time = new Date(0)

        this.estimate_start_time = new Date(0)

        this.estimate_end_time = new Date(0)

        this.attached_histories = new Array<Mi>()
    }

}


