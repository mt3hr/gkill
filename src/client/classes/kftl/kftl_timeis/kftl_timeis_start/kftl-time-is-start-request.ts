'use strict'

import { KFTLRequest } from '../../kftl-request'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddTimeisRequest } from '@/classes/api/req_res/add-timeis-request'

import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { i18n } from '@/i18n'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

export class KFTLTimeIsStartRequest extends KFTLRequest {

    private title: string

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        this.title = ""
    }

    async get_label_name(_context: KFTLStatementLineContext): Promise<string> {
        return i18n.global.t("KFTL_TIMEIS_TIMEIS_START_LABEL_TITLE")
    }

    async set_title(title: string): Promise<void> {
        this.title = title
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
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddTimeisRequest()
        const now = new Date(Date.now())
        req.tx_id = this.get_tx_id()

        req.timeis.id = this.get_request_id()
        req.timeis.start_time = time
        req.timeis.title = this.title
        req.timeis.create_app = "gkill_kftl"
        req.timeis.create_device = application_config.device
        req.timeis.create_time = now
        req.timeis.create_user = application_config.user_id
        req.timeis.update_app = "gkill_kftl"
        req.timeis.update_device = application_config.device
        req.timeis.update_time = now
        req.timeis.update_user = application_config.user_id

        await delete_gkill_kyou_cache(req.timeis.id)
        await gkill_api.add_timeis(req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

}


