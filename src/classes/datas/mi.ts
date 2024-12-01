'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class Mi extends InfoBase {

    title: string

    is_checked: boolean

    board_name: string

    limit_time: Date | null

    estimate_start_time: Date | null

    estimate_end_time: Date | null

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

    clone(): Mi {
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
        this.title = ""

        this.is_checked = false

        this.board_name = ""

        this.limit_time = new Date(0)

        this.estimate_start_time = new Date(0)

        this.estimate_end_time = new Date(0)

        this.attached_histories = new Array<Mi>()
    }

}


