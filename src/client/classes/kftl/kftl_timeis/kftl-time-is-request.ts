'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'

import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { i18n } from '@/i18n'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

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

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        if (this.title == "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.skiped_no_content_timeis
            error.error_message = i18n.global.t("KFTL_TIMEIS_BLANK_SKIP_SAVE_MESSAGE_TITLE")
            errors.push(error)
            return errors
        }

        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))
        const related_time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const now = new Date(Date.now())

        const timeis_req = new AddTimeisRequest()
        timeis_req.tx_id = this.get_tx_id()
        timeis_req.timeis.id = this.get_request_id()
        await this.set_start_time(related_time ? related_time : new Date(Date.now()))
        timeis_req.timeis.start_time = this.start_time
        timeis_req.timeis.end_time = this.end_time
        timeis_req.timeis.title = this.title
        timeis_req.timeis.create_app = "gkill_kftl"
        timeis_req.timeis.create_device = application_config.device
        timeis_req.timeis.create_time = now
        timeis_req.timeis.create_user = application_config.user_id
        timeis_req.timeis.update_app = "gkill_kftl"
        timeis_req.timeis.update_device = application_config.device
        timeis_req.timeis.update_time = now
        timeis_req.timeis.update_user = application_config.user_id

        await delete_gkill_kyou_cache(timeis_req.timeis.id)
        await gkill_api.add_timeis(timeis_req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })

        return errors
    }
}


