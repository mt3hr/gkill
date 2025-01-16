'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { AddNlogRequest } from '@/classes/api/req_res/add-nlog-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'

export class KFTLNlogRequest extends KFTLRequest {

    shop_name: string

    titles: Array<string>

    amounts: Array<Number>

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.shop_name = ""
        this.titles = new Array<string>()
        this.amounts = new Array<Number>()
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        await super.do_request().then(super_errors => errors = errors.concat(super_errors))
        if (this.titles.length != this.amounts.length) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "メモと金額の個数が一致していません"
            errors.push(error)
            return errors
        }
        for (let i = 0; i < this.titles.length; i++) {
            const memo = this.titles[i]
            const amount = this.amounts[i]
            if (memo == "" && amount == 0 && this.shop_name == "") {
                const error = new GkillError()
                error.error_code = "//TODO"
                error.error_message = "内容がないnlogの保存がスキップされました"
                errors.push(error)
            }
            const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
            const req = new AddNlogRequest()
            
            req.nlog.id = GkillAPI.get_gkill_api().generate_uuid()
            req.nlog.shop = this.shop_name
            req.nlog.amount = amount
            req.nlog.title = memo
            req.nlog.related_time = time

            req.nlog.create_app = "gkill_kftl"
            req.nlog.create_device = gkill_info_res.device
            req.nlog.create_time = time
            req.nlog.create_user = gkill_info_res.user_id
            req.nlog.update_app = "gkill_kftl"
            req.nlog.update_device = gkill_info_res.device
            req.nlog.update_time = time
            req.nlog.update_user = gkill_info_res.user_id

            await GkillAPI.get_gkill_api().add_nlog(req).then(res => {
                if (res.errors && res.errors.length !== 0) {
                    errors = errors.concat(res.errors)
                }
            })
        }
        return errors
    }

    set_shop_name(shop_name: string): void {
        this.shop_name = shop_name
    }

    add_title(title: string): void {
        this.titles.push(title)
    }

    add_amount(amount: Number): void {
        this.amounts.push(amount)
    }

}


