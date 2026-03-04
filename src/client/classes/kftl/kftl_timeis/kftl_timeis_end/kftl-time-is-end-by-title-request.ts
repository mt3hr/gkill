'use strict'

import { KFTLRequest } from '../../kftl-request'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'

import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import type { TimeIs } from '@/classes/datas/time-is'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import generate_get_plaing_timeis_kyous_query from '@/classes/api/generate-get-plaing-timeis-kyous-query'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { i18n } from '@/i18n'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

export class KFTLTimeIsEndByTitleRequest extends KFTLRequest {

    private title: string
    private error_when_target_does_not_exist: boolean

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        this.title = ""
        this.error_when_target_does_not_exist = false
    }

    set_title(title: string): void {
        this.title = title
    }

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        if (this.title == "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_setted_timeis_end_title
            error.error_message = i18n.global.t("KFTL_TIMEIS_END_REQUIRE_END_TITLE_MESSAGE_TITLE")
            errors.push(error)
            return errors
        }
        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())

        // 対象のtimeisを取得する
        let target_timeis: TimeIs | null = null
        const get_plaing_timeis_query = generate_get_plaing_timeis_kyous_query(null)
        const get_plaing_timeis_req = new GetKyousRequest()
        get_plaing_timeis_req.query = get_plaing_timeis_query

        await gkill_api.delete_updated_gkill_caches()
        const get_plaing_timeis_res = await gkill_api.get_kyous(get_plaing_timeis_req)
        if (get_plaing_timeis_res.errors && get_plaing_timeis_res.errors.length !== 0) {
            errors = errors.concat(get_plaing_timeis_res.errors)
            return errors
        }

        const awaitPromisses = Array<Promise<any>>()
        for (let i = 0; i < get_plaing_timeis_res.kyous.length; i++) {
            const timeis = get_plaing_timeis_res.kyous[i]
            awaitPromisses.push(timeis.load_typed_timeis())
        }
        await Promise.all(awaitPromisses)

        for (let i = 0; i < get_plaing_timeis_res.kyous.length; i++) {
            const timeis_kyou = get_plaing_timeis_res.kyous[i]
            if (timeis_kyou.typed_timeis) {
                if (timeis_kyou.typed_timeis.title === this.title) {
                    target_timeis = timeis_kyou.typed_timeis
                    break
                }
            }
        }

        if (!target_timeis) {
            if (!this.error_when_target_does_not_exist) {
                return errors
            }
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_end_timeis_target
            error.error_message = i18n.global.t("KFTL_TIMEIS_END_TARGET_NOT_FOUND_MESSAGE_TITLE")
            return [error]
        }

        // end_timeをいれてUPDATEする
        await delete_gkill_kyou_cache(target_timeis.id)
        const update_timeis_req = new UpdateTimeisRequest()
        update_timeis_req.tx_id = this.get_tx_id()
        update_timeis_req.timeis = target_timeis.clone()
        update_timeis_req.timeis.end_time = time
        update_timeis_req.timeis.update_app = "gkill_kftl"
        update_timeis_req.timeis.update_device = application_config.device
        update_timeis_req.timeis.update_time = time
        update_timeis_req.timeis.update_user = application_config.user_id

        await gkill_api.update_timeis(update_timeis_req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

    set_error_when_target_does_not_exist(error_when_target_does_not_exist: boolean) {
        this.error_when_target_does_not_exist = error_when_target_does_not_exist
    }
}