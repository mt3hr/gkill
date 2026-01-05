'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddLantanaRequest } from '@/classes/api/req_res/add-lantana-request'

import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

export class KFTLLantanaRequest extends KFTLRequest {

    private mood: Number

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.mood = 0
    }

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))

        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddLantanaRequest()
        const now = new Date(Date.now())
        req.tx_id = this.get_tx_id()

        req.lantana.id = this.get_request_id()
        req.lantana.mood = this.mood
        req.lantana.related_time = time

        req.lantana.create_app = "gkill_kftl"
        req.lantana.create_device = application_config.device
        req.lantana.create_time = now
        req.lantana.create_user = application_config.user_id
        req.lantana.update_app = "gkill_kftl"
        req.lantana.update_device = application_config.device
        req.lantana.update_time = now
        req.lantana.update_user = application_config.user_id

        await delete_gkill_kyou_cache(req.lantana.id)
        await gkill_api.add_lantana(req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

    set_mood(mood: Number): void {
        this.mood = mood
    }
}


