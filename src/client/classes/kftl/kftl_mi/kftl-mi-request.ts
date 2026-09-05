'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddMiRequest } from '@/classes/api/req_res/add-mi-request'

import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { i18n } from '@/i18n'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { schedule_anchor_of, is_mi_data_type } from '../kftl_repeat/kftl-repeat-duplicate'
import { find_existing_anchors } from '../kftl_repeat/kftl-repeat-duplicate'
import { shift_days } from '../kftl_repeat/kftl-repeat-spec'

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

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        if (this.title == "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.mi_title_is_blank
            error.error_message = i18n.global.t("KFTL_MI_TITLE_BLANK_SKIP_SAVE_MESSAGE_TITLE")
            errors = errors.concat([error])
        }

        if (this.board_name == "") {
            this.board_name = application_config.mi_default_board
        }
        if (errors.length !== 0) {
            return errors
        }

        const board_name = this.board_name != "" ? this.board_name : application_config.mi_default_board
        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))
        const id = this.get_request_id()
        const now = new Date(Date.now())

        const mi_req = new AddMiRequest()
        mi_req.tx_id = this.get_tx_id()

        mi_req.mi.id = id
        mi_req.mi.title = this.title
        mi_req.mi.board_name = board_name
        mi_req.mi.limit_time = this.limit_time
        mi_req.mi.estimate_start_time = this.estimate_start_time
        mi_req.mi.estimate_end_time = this.esitimate_end_time
        mi_req.mi.is_checked = false

        mi_req.mi.create_app = "gkill_kftl"
        mi_req.mi.create_device = application_config.device
        mi_req.mi.create_time = now
        mi_req.mi.create_user = application_config.user_id
        mi_req.mi.update_app = "gkill_kftl"
        mi_req.mi.update_device = application_config.device
        mi_req.mi.update_time = now
        mi_req.mi.update_user = application_config.user_id

        await delete_gkill_kyou_cache(mi_req.mi.id)
        await gkill_api.add_mi(mi_req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            } else {
                // 成功したものだけ積む。実体は commit_tx のあとに引き直される
                this.add_registered_kyou_id(mi_req.mi.id)
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

    // 板名行を書かなかったときは空のまま。do_request が既定の板へフォールバックするので、
    // 「ユーザが新しい板名を入力した」ことにはならない
    get_mi_board_name(): string {
        return this.board_name
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


    /**
     * 書き込みに使う板名。空なら設定の既定板になる。
     * **既存判定と書き込みで同じ値を使うため**にここへ出してある。
     */
    resolved_board_name(application_config: ApplicationConfig | null): string {
        if (this.board_name !== "") {
            return this.board_name
        }
        return application_config !== null ? application_config.mi_default_board : ""
    }

    /**
     * 予定日時のうち最初に埋まっているものを基準にする。
     * タスクに関連時刻の列は無いので、1つも埋まっていなければ繰り返しの入れ先が無い。
     */
    override anchor_time_for_repeat(): Date | null {
        return schedule_anchor_of(this.estimate_start_time, this.esitimate_end_time, this.limit_time)
    }

    /**
     * 3つの予定日時を**同じ日数だけ**ずらす。
     * 欄どうしの相対差（見積開始の2日後が期限、など）はそのまま保たれる。
     */
    override clone_for_repeat(new_request_id: string, day_shift: number): KFTLRequest {
        const cloned = new KFTLMiRequest(new_request_id, this.get_context())
        this.copy_base_state_for_repeat(cloned, day_shift)
        cloned.title = this.title
        cloned.board_name = this.board_name
        cloned.estimate_start_time = this.estimate_start_time === null ? null : shift_days(this.estimate_start_time, day_shift)
        cloned.esitimate_end_time = this.esitimate_end_time === null ? null : shift_days(this.esitimate_end_time, day_shift)
        cloned.limit_time = this.limit_time === null ? null : shift_days(this.limit_time, day_shift)
        return cloned
    }

    override async find_existing_for_repeat(gkill_api: GkillAPI, application_config: ApplicationConfig | null, from: Date, to: Date): Promise<Set<number>> {
        const board_name = this.resolved_board_name(application_config)
        return find_existing_anchors(gkill_api, from, to, true,
            is_mi_data_type,
            (kyou) => {
                if (kyou.typed_mi === null || kyou.typed_mi.title !== this.title || kyou.typed_mi.board_name !== board_name) {
                    return null
                }
                return schedule_anchor_of(kyou.typed_mi.estimate_start_time, kyou.typed_mi.estimate_end_time, kyou.typed_mi.limit_time)
            })
    }


    get_estimate_start_time(): Date | null {
        return this.estimate_start_time
    }

    get_estimate_end_time(): Date | null {
        return this.esitimate_end_time
    }

    get_limit_time(): Date | null {
        return this.limit_time
    }

}


