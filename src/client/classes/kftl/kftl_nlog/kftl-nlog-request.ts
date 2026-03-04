'use strict'

import { GkillAPI } from '@/classes/api/gkill-api'
import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { AddNlogRequest } from '@/classes/api/req_res/add-nlog-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { i18n } from '@/i18n'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

export class KFTLNlogRequest extends KFTLRequest {

    shop_name: string

    titles: Array<string>

    amounts: Array<number>

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.shop_name = ""
        this.titles = new Array<string>()
        this.amounts = new Array<number>()
    }

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))
        if (this.titles.length != this.amounts.length) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.nlog_title_amount_count_not_equal
            error.error_message = i18n.global.t("KFTL_NLOG_INVALID_RECORD_COUNT_MESSAGE_TITLE")
            errors.push(error)
            return errors
        }
        for (let i = 0; i < this.titles.length; i++) {
            const memo = this.titles[i]
            const amount = this.amounts[i]
            if (memo == "" && amount == 0 && this.shop_name == "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.skiped_no_content_nlog
                error.error_message = i18n.global.t("KFTL_NLOG_BLANK_SKIP_SAVE_MESSAGE_TITLE")
                errors.push(error)
            }
            const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
            const req = new AddNlogRequest()
            req.tx_id = this.get_tx_id()
            const now = new Date(Date.now())

            req.nlog.id = gkill_api.generate_uuid()
            req.nlog.shop = this.shop_name
            req.nlog.amount = amount
            req.nlog.title = memo
            req.nlog.related_time = time

            req.nlog.create_app = "gkill_kftl"
            req.nlog.create_device = application_config.device
            req.nlog.create_time = now
            req.nlog.create_user = application_config.user_id
            req.nlog.update_app = "gkill_kftl"
            req.nlog.update_device = application_config.device
            req.nlog.update_time = now
            req.nlog.update_user = application_config.user_id

            await delete_gkill_kyou_cache(req.nlog.id)
            await gkill_api.add_nlog(req).then(res => {
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

    add_amount(amount: number): void {
        this.amounts.push(amount)
    }

}


