'use strict'

import { FindKyouQuery } from "../api/find_query/find-kyou-query"

export class ShareKyousInfo {
    share_id: string
    user_id: string
    device: string
    share_title: string
    find_query_json: FindKyouQuery
    view_type: string
    is_share_time_only: boolean
    is_share_with_tags: boolean
    is_share_with_texts: boolean
    is_share_with_timeiss: boolean
    is_share_with_locations: boolean
    clone(): ShareKyousInfo {
        const share_kyou_list_info = new ShareKyousInfo()
        share_kyou_list_info.share_id = this.share_id
        share_kyou_list_info.user_id = this.user_id
        share_kyou_list_info.device = this.device
        share_kyou_list_info.share_title = this.share_title
        share_kyou_list_info.find_query_json = this.find_query_json
        share_kyou_list_info.view_type = this.view_type
        share_kyou_list_info.is_share_time_only = this.is_share_time_only
        share_kyou_list_info.is_share_with_tags = this.is_share_with_tags
        share_kyou_list_info.is_share_with_texts = this.is_share_with_texts
        share_kyou_list_info.is_share_with_timeiss = this.is_share_with_timeiss
        share_kyou_list_info.is_share_with_locations = this.is_share_with_locations
        return share_kyou_list_info
    }
    constructor() {
        this.share_id = ""
        this.user_id = ""
        this.device = ""
        this.share_title = ""
        this.find_query_json = new FindKyouQuery()
        this.view_type = "rykv"
        this.is_share_time_only = false
        this.is_share_with_tags = false
        this.is_share_with_texts = false
        this.is_share_with_timeiss = false
        this.is_share_with_locations = false
    }
}
