'use strict'

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response'
import { KFTLRequest } from '../kftl-request'
import type { KFTLRequestMap } from '../kftl-request-map'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'

export class KFTLTimeIsRequest extends KFTLRequest {

    private title: string

    private start_time: Date

    private end_time: Date | null

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        this.title = ""
        this.start_time = new Date(0)
        this.end_time = null
    }

    async set_title(title: string): Promise<void> {
        this.title = title
    }

    async set_start_time(start_time: Date): Promise<void> {
        this.start_time = start_time
    }

    async set_end_time(end_time: Date | null): Promise<void> {
        this.end_time = end_time
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

        const timeis_req = new AddTimeisRequest()
        timeis_req.session_id = GkillAPI.get_instance().get_session_id()
        timeis_req.timeis.id = this.get_request_id()
        const related_time = this.get_related_time()
        await this.set_start_time(related_time ? related_time : new Date(Date.now()))
        timeis_req.timeis.start_time = this.start_time
        timeis_req.timeis.end_time = this.end_time
        timeis_req.timeis.title = this.title
        timeis_req.timeis.create_app = "gkill_kftl"
        timeis_req.timeis.create_device = gkill_info_res.device
        timeis_req.timeis.create_time = time
        timeis_req.timeis.create_user = gkill_info_res.user_id
        timeis_req.timeis.update_app = "gkill_kftl"
        timeis_req.timeis.update_device = gkill_info_res.device
        timeis_req.timeis.update_time = time
        timeis_req.timeis.update_user = gkill_info_res.user_id

        await GkillAPI.get_instance().add_timeis(timeis_req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })

        return errors
    }
}


