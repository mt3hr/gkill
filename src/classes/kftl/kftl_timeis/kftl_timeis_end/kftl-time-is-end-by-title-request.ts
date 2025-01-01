'use strict'

import { KFTLRequest } from '../../kftl-request'
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { GetPlaingTimeisRequest } from '@/classes/api/req_res/get-plaing-timeis-request'
import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import type { TimeIs } from '@/classes/datas/time-is'

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

    async do_request(): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        if (this.title == "") {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "TimeIs終了タイトルを指定してください"
            errors.push(error)
            return errors
        }
        await super.do_request().then(super_errors => errors = errors.concat(super_errors))
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())

        const gkill_info_req = new GetGkillInfoRequest()
        gkill_info_req.session_id = GkillAPI.get_gkill_api().get_session_id()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        // 対象のtimeisを取得する
        let target_timeis: TimeIs | null = null
        const get_plaing_timeis_req = new GetPlaingTimeisRequest()
        get_plaing_timeis_req.session_id = GkillAPI.get_gkill_api().get_session_id()
        const get_plaing_timeis_res = await GkillAPI.get_gkill_api().get_plaing_timeis(get_plaing_timeis_req)
        if (get_plaing_timeis_res.errors && get_plaing_timeis_res.errors.length !== 0) {
            errors = errors.concat(get_plaing_timeis_res.errors)
            return errors
        }

        const awaitPromisses = Array<Promise<any>>()
        for (let i = 0; i < get_plaing_timeis_res.plaing_timeis_kyous.length; i++) {
            const timeis = get_plaing_timeis_res.plaing_timeis_kyous[i]
            awaitPromisses.push(timeis.load_typed_timeis())
        }
        await Promise.all(awaitPromisses)

        for (let i = 0; i < get_plaing_timeis_res.plaing_timeis_kyous.length; i++) {
            const timeis_kyou = get_plaing_timeis_res.plaing_timeis_kyous[i]
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
            error.error_code = "//TODO"
            error.error_message = "終了対象のTimeIsが存在しませんでした"
            throw new Error(error.error_message)
        }

        // end_timeをいれてUPDATEする
        const update_timeis_req = new UpdateTimeisRequest()
        update_timeis_req.session_id = GkillAPI.get_gkill_api().get_session_id()
        update_timeis_req.timeis = target_timeis.clone()
        update_timeis_req.timeis.end_time = time
        update_timeis_req.timeis.update_app = "gkill_kftl"
        update_timeis_req.timeis.update_device = gkill_info_res.device
        update_timeis_req.timeis.update_time = time
        update_timeis_req.timeis.update_user = gkill_info_res.user_id

        await GkillAPI.get_gkill_api().update_timeis(update_timeis_req).then(res => {
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