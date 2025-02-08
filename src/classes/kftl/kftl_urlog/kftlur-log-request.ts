'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { AddURLogRequest } from '@/classes/api/req_res/add-ur-log-request'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'

export class KFTLURLogRequest extends KFTLRequest {

    private url: string

    private title: string

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        this.url = ""
        this.title = ""
    }

    async set_url(url: string): Promise<void> {
        this.url = url
    }

    async set_title(title: string): Promise<void> {
        this.title = title
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        if (this.url == "" && this.title == "") {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "内容がないurlogの保存がスキップされました"
            errors.push(error)
            return errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        await super.do_request().then(super_errors => errors = errors.concat(super_errors))
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddURLogRequest()
        const now = new Date(Date.now())

        req.urlog.id = this.get_request_id()
        req.urlog.related_time = time
        req.urlog.url = this.url
        req.urlog.title = this.title
        req.urlog.create_app = "gkill_kftl"
        req.urlog.create_device = gkill_info_res.device
        req.urlog.create_time = now
        req.urlog.create_user = gkill_info_res.user_id
        req.urlog.update_app = "gkill_kftl"
        req.urlog.update_device = gkill_info_res.device
        req.urlog.update_time = now
        req.urlog.update_user = gkill_info_res.user_id

        await GkillAPI.get_gkill_api().add_urlog(req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }
}


