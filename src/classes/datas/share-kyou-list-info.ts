'use strict'

import { FindKyouQuery } from "../api/find_query/find-kyou-query"

export class ShareKyouListInfo {
    user_id: string
    device: string
    share_title: string
    is_share_detail: boolean
    share_id: string
    find_query_json: FindKyouQuery
    clone(): ShareKyouListInfo {
        const share_kyou_list_info = new ShareKyouListInfo()
        share_kyou_list_info.user_id = this.user_id
        share_kyou_list_info.device = this.device
        share_kyou_list_info.share_title = this.share_title
        share_kyou_list_info.is_share_detail = this.is_share_detail
        share_kyou_list_info.share_id = this.share_id
        share_kyou_list_info.find_query_json = this.find_query_json
        return share_kyou_list_info
    }
    constructor() {
        this.user_id = ""
        this.device = ""
        this.share_title = ""
        this.is_share_detail = false
        this.share_id = ""
        this.find_query_json = new FindKyouQuery()
    }
}
