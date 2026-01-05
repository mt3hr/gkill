'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddKCRequest } from '@/classes/api/req_res/add-kc-request'

import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

export class KFTLKCRequest extends KFTLRequest {

    private title: string
    private num_value: Number

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.title = ""
        this.num_value = 0
    }

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))

        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddKCRequest()
        const now = new Date(Date.now())
        req.tx_id = this.get_tx_id()

        req.kc.id = this.get_request_id()
        req.kc.title = this.title
        req.kc.num_value = this.num_value.valueOf()
        req.kc.related_time = time

        req.kc.create_app = "gkill_kftl"
        req.kc.create_device = application_config.device
        req.kc.create_time = now
        req.kc.create_user = application_config.user_id
        req.kc.update_app = "gkill_kftl"
        req.kc.update_device = application_config.device
        req.kc.update_time = now
        req.kc.update_user = application_config.user_id

        await delete_gkill_kyou_cache(req.kc.id)
        await gkill_api.add_kc(req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

    set_title(title: string): void {
        this.title = title
    }

    set_num_value(num_value: Number): void {
        this.num_value = num_value
    }
}


