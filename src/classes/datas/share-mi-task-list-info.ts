'use strict'

export class ShareMiTaskListInfo {
    user_id: string
    device: string
    share_title: string
    is_share_detail: boolean
    share_id: string
    find_query_json: string
    async clone(): Promise<ShareMiTaskListInfo> {
        const share_mi_task_list_info = new ShareMiTaskListInfo
        share_mi_task_list_info.user_id = this.user_id
        share_mi_task_list_info.device = this.device
        share_mi_task_list_info.share_title = this.share_title
        share_mi_task_list_info.is_share_detail = this.is_share_detail
        share_mi_task_list_info.share_id = this.share_id
        share_mi_task_list_info.find_query_json = this.find_query_json
        return share_mi_task_list_info
    }
    constructor() {
        this.user_id = ""
        this.device = ""
        this.share_title = ""
        this.is_share_detail = false
        this.share_id = ""
        this.find_query_json = "{}"
    }
}
