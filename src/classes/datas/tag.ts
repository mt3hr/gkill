'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { GetTagHistoryByTagIDRequest } from '../api/req_res/get-tag-history-by-tag-id-request'
import { InfoIdentifier } from './info-identifier'
import { MetaInfoBase } from './meta-info-base'

export class Tag extends MetaInfoBase {

    tag: string

    attached_histories: Array<Tag>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetTagHistoryByTagIDRequest()
        
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_tag_histories_by_tag_id(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.tag_histories
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

    clone(): Tag {
        const tag = new Tag()
        tag.is_deleted = this.is_deleted
        tag.id = this.id
        tag.target_id = this.target_id
        tag.related_time = this.related_time
        tag.create_time = this.create_time
        tag.create_app = this.create_app
        tag.create_device = this.create_device
        tag.create_user = this.create_user
        tag.update_time = this.update_time
        tag.update_app = this.update_app
        tag.update_device = this.update_device
        tag.update_user = this.update_user
        tag.tag = this.tag
        return tag
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


