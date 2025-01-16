'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetTextHistoryByTextIDRequest } from '../api/req_res/get-text-history-by-tag-id-request'
import { InfoIdentifier } from './info-identifier'
import { MetaInfoBase } from './meta-info-base'

export class Text extends MetaInfoBase {

    text: string

    attached_histories: Array<Text>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetTextHistoryByTextIDRequest()
        
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_text_history_by_text_id(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.text_histories
        return new Array<GkillError>()
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        try {
            return this.load_attached_histories()
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
            return []
        }
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        this.attached_histories = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        return this.clear_attached_histories()
    }

    clone(): Text {
        const text = new Text()
        text.is_deleted = this.is_deleted
        text.id = this.id
        text.target_id = this.target_id
        text.related_time = this.related_time
        text.create_time = this.create_time
        text.create_app = this.create_app
        text.create_device = this.create_device
        text.create_user = this.create_user
        text.update_time = this.update_time
        text.update_app = this.update_app
        text.update_device = this.update_device
        text.update_user = this.update_user
        text.text = this.text
        return text
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
        this.text = ""
        this.attached_histories = new Array<Text>()
    }

}


