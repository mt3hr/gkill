'use strict'

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response'
import { KFTLRequest } from '../../kftl-request'
import type { KFTLRequestMap } from '../../kftl-request-map'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'

export class KFTLTimeIsStartRequest extends KFTLRequest {

    private title: string

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        this.title = ""
    }

    async apply_this_line_to_request_map(requet_map: KFTLRequestMap): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_label_name(context: KFTLStatementLineContext): Promise<string> {
        throw new Error('Not implemented')
    }

    async set_title(title: string): Promise<void> {
        throw new Error('Not implemented')
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        if (this.title == "") {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "内容がないtimeisの保存がスキップされました"
            errors.push(error)
            return errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        gkill_info_req.session_id = GkillAPI.get_instance().get_session_id()
        const gkill_info_res = await GkillAPI.get_instance().get_gkill_info(gkill_info_req)

        await super.do_request().then(super_errors => errors = errors.concat(super_errors))
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddTimeisRequest()
        req.timeis.id = this.get_request_id()
        req.timeis.start_time = time
        req.timeis.title = this.title
        req.timeis.create_app = "gkill_kftl"
        req.timeis.create_device = gkill_info_res.device
        req.timeis.create_time = time
        req.timeis.create_user = gkill_info_res.user_id
        req.timeis.update_app = "gkill_kftl"
        req.timeis.update_device = gkill_info_res.device
        req.timeis.update_time = time
        req.timeis.update_user = gkill_info_res.user_id
        await GkillAPI.get_instance().add_timeis(req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

}


