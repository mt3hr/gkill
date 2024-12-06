'use strict'

import type { GkillError } from '../api/gkill-error'
import { InfoBase } from './info-base'
import { InfoIdentifier } from './info-identifier'

export class Mi extends InfoBase {

    title: string

    is_checked: boolean

    checked_time: Date | null

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
        const mi = new Mi()
        mi.is_deleted = this.is_deleted
        mi.id = this.id
        mi.rep_name = this.rep_name
        mi.related_time = this.related_time
        mi.data_type = this.data_type
        mi.create_time = this.create_time
        mi.create_app = this.create_app
        mi.create_device = this.create_device
        mi.create_user = this.create_user
        mi.update_time = this.update_time
        mi.update_app = this.update_app
        mi.update_user = this.update_user
        mi.update_device = this.update_device
        mi.title = this.title
        mi.is_checked = this.is_checked
        mi.board_name = this.board_name
        mi.limit_time = this.limit_time
        mi.estimate_start_time = this.estimate_start_time
        mi.estimate_end_time = this.estimate_end_time
        return mi
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

        this.checked_time = null

        this.attached_histories = new Array<Mi>()
    }

}


