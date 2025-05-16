'use strict'

import { KFTLRequest } from "../../kftl-request"
import type { KFTLStatementLineContext } from "../../kftl-statement-line-context"
import { GkillError } from "@/classes/api/gkill-error"
import { UpdateTimeisRequest } from "@/classes/api/req_res/update-timeis-request"
import { GkillAPI } from "@/classes/api/gkill-api"
import type { TimeIs } from "@/classes/datas/time-is"
import { GetGkillInfoRequest } from "@/classes/api/req_res/get-gkill-info-request"
import { GkillErrorCodes } from "@/classes/api/message/gkill_error"
import generate_get_plaing_timeis_kyous_query from "@/classes/api/generate-get-plaing-timeis-kyous-query"
import { GetKyousRequest } from "@/classes/api/req_res/get-kyous-request"

export class KFTLTimeIsEndByTagRequest extends KFTLRequest {

    private target_tag_names: Array<string>
    private error_when_target_does_not_exist: boolean

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
        this.target_tag_names = new Array<string>()
        this.error_when_target_does_not_exist = false
    }

    set_target_tag_names(target_tag_names: Array<string>): void {
        this.target_tag_names = target_tag_names
    }

    add_target_tag_name(target_tag_name: string): void {
        this.target_tag_names.push(target_tag_name)
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        if (this.target_tag_names.length == 0) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_setted_timeis_end_tag
            error.error_message = "TimeIs終了タグを指定してください"
            errors.push(error)
            return errors
        }
        await super.do_request().then(super_errors => errors = errors.concat(super_errors))
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())


        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        // 対象のtimeisを取得する
        let target_timeis: TimeIs | null = null
        const get_plaing_timeis_query = generate_get_plaing_timeis_kyous_query(null)
        const get_plaing_timeis_req = new GetKyousRequest()
        get_plaing_timeis_req.query = get_plaing_timeis_query
        const get_plaing_timeis_res = await GkillAPI.get_gkill_api().get_kyous(get_plaing_timeis_req)
        if (get_plaing_timeis_res.errors && get_plaing_timeis_res.errors.length !== 0) {
            errors = errors.concat(get_plaing_timeis_res.errors)
            return errors
        }

        const awaitPromisses = Array<Promise<any>>()
        for (let i = 0; i < get_plaing_timeis_res.kyous.length; i++) {
            const timeis = get_plaing_timeis_res.kyous[i]
            awaitPromisses.push(timeis.load_typed_timeis())
            awaitPromisses.push(timeis.load_attached_tags())
        }
        await Promise.all(awaitPromisses)

        for (let i = 0; i < get_plaing_timeis_res.kyous.length; i++) {
            const timeis_kyou = get_plaing_timeis_res.kyous[i]
            const attached_tag_names = Array<string>()
            for (let j = 0; j < timeis_kyou.attached_tags.length; j++) {
                attached_tag_names.push(timeis_kyou.attached_tags[j].tag)
            }

            let is_match_tags = timeis_kyou.attached_tags.length !== 0
            for (let j = 0; j < this.target_tag_names.length; j++) {
                const target_tag_name = this.target_tag_names[j]
                if (!attached_tag_names.includes(target_tag_name)) {
                    is_match_tags = false
                    break
                }
            }
            if (is_match_tags) {
                target_timeis = timeis_kyou.typed_timeis
                break
            }
        }

        if (!target_timeis) {
            if (!this.error_when_target_does_not_exist) {
                return errors
            }
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_end_timeis_target
            error.error_message = "終了対象のTimeIsが存在しませんでした"
            return [error]
        }

        // end_timeをいれてUPDATEする
        const update_timeis_req = new UpdateTimeisRequest()
        update_timeis_req.timeis = target_timeis
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