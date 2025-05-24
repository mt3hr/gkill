'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { AddKmemoRequest } from '@/classes/api/req_res/add-kmemo-request'
import { Kmemo } from '@/classes/datas/kmemo'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_cache from '@/classes/delete-gkill-cache'

export class KFTLKmemoRequest extends KFTLRequest {

    private kmemo_content: string

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.kmemo_content = ""
    }

    override async do_request(): Promise<Array<GkillError>> {
        let errors: Array<GkillError> = new Array<GkillError>()
        if (this.kmemo_content == "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.skiped_no_content_kmemo
            error.error_message = "内容がないKmemoの保存がスキップされました"
            errors.push(error)
            return errors
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        await super.do_request().then(super_errors => (errors = errors.concat(super_errors)))
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())
        const req = new AddKmemoRequest()
        const now = new Date(Date.now())
        req.tx_id = this.get_tx_id()

        req.kmemo = new Kmemo()
        req.kmemo.content = this.kmemo_content
        req.kmemo.id = this.get_request_id()
        req.kmemo.related_time = time

        req.kmemo.create_app = "gkill_kftl"
        req.kmemo.create_device = gkill_info_res.device
        req.kmemo.create_time = now
        req.kmemo.create_user = gkill_info_res.user_id
        req.kmemo.update_app = "gkill_kftl"
        req.kmemo.update_device = gkill_info_res.device
        req.kmemo.update_time = now
        req.kmemo.update_user = gkill_info_res.user_id
        await delete_gkill_cache(req.kmemo.id)
        await GkillAPI.get_gkill_api().add_kmemo(req).then((res) => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

    add_kmemo_line(kmemo_line: string): void {
        if (this.kmemo_content == "") {
            this.kmemo_content += `${kmemo_line}`
        } else {
            this.kmemo_content += `\n${kmemo_line}`
        }
    }
}


