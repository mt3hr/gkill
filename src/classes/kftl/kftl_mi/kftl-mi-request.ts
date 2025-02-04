'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetApplicationConfigRequest } from '@/classes/api/req_res/get-application-config-request'
import { AddMiRequest } from '@/classes/api/req_res/add-mi-request'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'

export class KFTLMiRequest extends KFTLRequest {

    private title: string

    private board_name: string

    private limit_time: Date | null

    private estimate_start_time: Date | null

    private esitimate_end_time: Date | null

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.title = ""
        this.board_name = ""
        this.limit_time = null
        this.estimate_start_time = null
        this.esitimate_end_time = null
    }

    async do_request(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        if (this.title == "") {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "タイトルが未入力です"
            errors = errors.concat([error])
        }

        const gkill_info_req = new GetGkillInfoRequest()
        const gkill_info_res = await GkillAPI.get_gkill_api().get_gkill_info(gkill_info_req)

        if (this.board_name == "") {
            const req = new GetApplicationConfigRequest()

            const res = await GkillAPI.get_gkill_api().get_application_config(req)
            this.board_name = res.application_config.mi_default_board
        }
        if (errors.length !== 0) {
            return errors
        }

        const req = new GetApplicationConfigRequest()

        const res = await GkillAPI.get_gkill_api().get_application_config(req)
        if (res.errors && res.errors.length !== 0) {
            errors = errors.concat(res.errors)
            return errors
        }

        const board_name = this.board_name != "" ? this.board_name : res.application_config.mi_default_board

        await super.do_request().then(super_errors => errors = errors.concat(super_errors))
        const id = this.get_request_id()
        const time = this.get_related_time() ? this.get_related_time()!! : new Date(Date.now())

        const mi_req = new AddMiRequest()

        mi_req.mi.id = id
        mi_req.mi.title = this.title
        mi_req.mi.board_name = board_name
        mi_req.mi.limit_time = this.limit_time
        mi_req.mi.estimate_start_time = this.estimate_start_time
        mi_req.mi.estimate_end_time = this.esitimate_end_time
        mi_req.mi.is_checked = false

        mi_req.mi.create_app = "gkill_kftl"
        mi_req.mi.create_device = gkill_info_res.device
        mi_req.mi.create_time = time
        mi_req.mi.create_user = gkill_info_res.user_id
        mi_req.mi.update_app = "gkill_kftl"
        mi_req.mi.update_device = gkill_info_res.device
        mi_req.mi.update_time = time
        mi_req.mi.update_user = gkill_info_res.user_id

        await GkillAPI.get_gkill_api().add_mi(mi_req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            }
        })
        return errors
    }

    async set_title(title: string): Promise<void> {
        this.title = title
    }

    async set_board_name(board_name: string): Promise<void> {
        this.board_name = board_name
    }

    async set_limit_time(limit_time: Date | null): Promise<void> {
        this.limit_time = limit_time
    }

    async set_estimate_start_time(estimate_start_time: Date | null): Promise<void> {
        this.estimate_start_time = estimate_start_time
    }

    async set_estimate_end_time(estimate_end_time: Date | null): Promise<void> {
        this.esitimate_end_time = estimate_end_time
    }

}


