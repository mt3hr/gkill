'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddLantanaRequest } from '@/classes/api/req_res/add-lantana-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'

export class KFTLLantanaRequest extends KFTLRequest {

    private mood: Number

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.mood = 0
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        await super.do_request().then(super_errors => errors = errors.concat(super_errors))

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddLantanaRequest()
        const now = new Date(Date.now())

        req.lantana.id = this.get_request_id()
        req.lantana.mood = this.mood
        req.lantana.related_time = time

        req.lantana.create_app = "gkill_kftl"
        req.lantana.create_device = gkill_info_res.device
        req.lantana.create_time = now
        req.lantana.create_user = gkill_info_res.user_id
        req.lantana.update_app = "gkill_kftl"
        req.lantana.update_device = gkill_info_res.device
        req.lantana.update_time = now
        req.lantana.update_user = gkill_info_res.user_id

        await GkillAPI.get_gkill_api().add_lantana(req).then(res => {
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


