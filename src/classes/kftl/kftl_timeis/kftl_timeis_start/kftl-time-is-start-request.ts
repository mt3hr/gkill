'use strict'

import { KFTLRequest } from '../../kftl-request'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

export class KFTLTimeIsStartRequest extends KFTLRequest {

    private title: string

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        this.title = ""
    }

    async get_label_name(_context: KFTLStatementLineContext): Promise<string> {
        return "開始"
    }

    async set_title(title: string): Promise<void> {
        this.title = title
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        if (this.title == "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.skiped_no_content_timeis
            error.error_message = "内容がないtimeisの保存がスキップされました"
            errors.push(error)
            return errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        await super.do_request().then(super_errors => errors = errors.concat(super_errors))
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddTimeisRequest()
        const now = new Date(Date.now())

        req.timeis.id = this.get_request_id()
        req.timeis.start_time = time
        req.timeis.title = this.title
        req.timeis.create_app = "gkill_kftl"
        req.timeis.create_device = gkill_info_res.device
        req.timeis.create_time = now
        req.timeis.create_user = gkill_info_res.user_id
        req.timeis.update_app = "gkill_kftl"
        req.timeis.update_device = gkill_info_res.device
        req.timeis.update_time = now
        req.timeis.update_user = gkill_info_res.user_id
        await GkillAPI.get_gkill_api().add_timeis(req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

}


