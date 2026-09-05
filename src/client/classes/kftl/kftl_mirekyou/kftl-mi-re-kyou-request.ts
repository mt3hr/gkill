'use strict'

import { KFTLRequest } from '../kftl-request'
import type { KFTLRequestMap } from '../kftl-request-map'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillAPI } from '@/classes/api/gkill-api'
import { AddMiReKyouRequest } from '@/classes/api/req_res/add-mi-re-kyou-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { KFTLPrototypeRequest } from '../kftl_prototype/kftl-prototype-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { i18n } from '@/i18n'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { schedule_anchor_of, is_mirekyou_data_type } from '../kftl_repeat/kftl-repeat-duplicate'
import { find_existing_anchors } from '../kftl_repeat/kftl-repeat-duplicate'
import { shift_days } from '../kftl_repeat/kftl-repeat-spec'

/**
 * 同じレコードで書いたKyouをタスク化する(MiReKyou)リクエスト。
 *
 * request_mapのキーはMiReKyou自身のid(request_id)で、タスク化する対象のidはtarget_idに別に持つ。
 * 対象のidをキーにすると直前のKmemoリクエストを踏み潰してKFTLRequestMap.setがthrowする。
 */
export class KFTLMiReKyouRequest extends KFTLRequest {

    private target_id: string

    private board_name: string

    private limit_time: Date | null

    private estimate_start_time: Date | null

    private estimate_end_time: Date | null

    private request_map: KFTLRequestMap | null

    constructor(request_id: string, target_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.target_id = target_id
        this.board_name = ""
        this.limit_time = null
        this.estimate_start_time = null
        this.estimate_end_time = null
        this.request_map = null
    }

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        // タスク化する対象は「同じレコードで書いたKyou」。レコードにKyou本体が無い
        // (タグだけ書いてプロトタイプのまま終わった場合を含む)と、対象が存在しないMiReKyouになる。
        // そういうMiReKyouは検索でターゲット解決に失敗して結果から落ちるので、
        // 画面に出ないのに消せない行がリポジトリに残ってしまう。書く前に弾く
        const target_request = this.request_map ? this.request_map.get(this.target_id) : undefined
        if (!target_request || KFTLPrototypeRequest.is_prototype_request(target_request)) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_mi_rekyou_target
            error.error_message = i18n.global.t("NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE")
            return [error]
        }

        let errors = new Array<GkillError>()

        const board_name = this.board_name != "" ? this.board_name : application_config.mi_default_board
        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))
        const id = this.get_request_id()
        const now = new Date(Date.now())

        const mi_re_kyou_req = new AddMiReKyouRequest()
        mi_re_kyou_req.tx_id = this.get_tx_id()

        mi_re_kyou_req.mirekyou.id = id
        mi_re_kyou_req.mirekyou.is_deleted = false
        mi_re_kyou_req.mirekyou.target_id = this.target_id
        mi_re_kyou_req.mirekyou.board_name = board_name
        mi_re_kyou_req.mirekyou.limit_time = this.limit_time
        mi_re_kyou_req.mirekyou.estimate_start_time = this.estimate_start_time
        mi_re_kyou_req.mirekyou.estimate_end_time = this.estimate_end_time
        mi_re_kyou_req.mirekyou.is_checked = false
        mi_re_kyou_req.mirekyou.related_time = this.get_related_time() ?? now

        mi_re_kyou_req.mirekyou.create_app = "gkill_kftl"
        mi_re_kyou_req.mirekyou.create_device = application_config.device
        mi_re_kyou_req.mirekyou.create_time = now
        mi_re_kyou_req.mirekyou.create_user = application_config.user_id
        mi_re_kyou_req.mirekyou.update_app = "gkill_kftl"
        mi_re_kyou_req.mirekyou.update_device = application_config.device
        mi_re_kyou_req.mirekyou.update_time = now
        mi_re_kyou_req.mirekyou.update_user = application_config.user_id

        await delete_gkill_kyou_cache(mi_re_kyou_req.mirekyou.id)
        await gkill_api.add_mirekyou(mi_re_kyou_req).then(res => {
            if (res.errors && res.errors.length !== 0) {
                errors = errors.concat(res.errors)
            } else {
                // 成功したものだけ積む。実体は commit_tx のあとに引き直される
                this.add_registered_kyou_id(mi_re_kyou_req.mirekyou.id)
            }
        })
        return errors
    }

    /**
     * 宙ぶらりんのMiReKyouを書かないための検査に使う。
     * 開始行のapply_this_line_to_request_mapから注入される
     */
    set_request_map(request_map: KFTLRequestMap): void {
        this.request_map = request_map
    }

    // タスク化する対象のKyouのid
    get_target_id(): string {
        return this.target_id
    }

    // 板名行を書かなかったときは空のまま。do_request が既定の板へフォールバックするので、
    // 「ユーザが新しい板名を入力した」ことにはならない
    get_mi_board_name(): string {
        return this.board_name
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
        this.estimate_end_time = estimate_end_time
    }


    /** 板名の解決はタスクと同じ。既存判定と書き込みで同じ値を使う。 */
    resolved_board_name(application_config: ApplicationConfig | null): string {
        if (this.board_name !== "") {
            return this.board_name
        }
        return application_config !== null ? application_config.mi_default_board : ""
    }

    override anchor_time_for_repeat(): Date | null {
        return schedule_anchor_of(this.estimate_start_time, this.estimate_end_time, this.limit_time)
    }

    /**
     * 3つの予定日時をずらす。対象（target_id）は同じ記録を指したままで、
     * 「同じ記録を繰り返しタスク化する」という意味になる。
     */
    override clone_for_repeat(new_request_id: string, day_shift: number): KFTLRequest {
        const cloned = new KFTLMiReKyouRequest(new_request_id, this.target_id, this.get_context())
        this.copy_base_state_for_repeat(cloned, day_shift)
        cloned.board_name = this.board_name
        cloned.request_map = this.request_map
        cloned.estimate_start_time = this.estimate_start_time === null ? null : shift_days(this.estimate_start_time, day_shift)
        cloned.estimate_end_time = this.estimate_end_time === null ? null : shift_days(this.estimate_end_time, day_shift)
        cloned.limit_time = this.limit_time === null ? null : shift_days(this.limit_time, day_shift)
        return cloned
    }

    /** リポストタスクはタイトルを持たないので、対象の記録と板名で見る。 */
    override async find_existing_for_repeat(gkill_api: GkillAPI, application_config: ApplicationConfig | null, from: Date, to: Date): Promise<Set<number>> {
        const board_name = this.resolved_board_name(application_config)
        return find_existing_anchors(gkill_api, from, to, true,
            is_mirekyou_data_type,
            (kyou) => {
                if (kyou.typed_mirekyou === null || kyou.typed_mirekyou.target_id !== this.target_id || kyou.typed_mirekyou.board_name !== board_name) {
                    return null
                }
                return schedule_anchor_of(kyou.typed_mirekyou.estimate_start_time, kyou.typed_mirekyou.estimate_end_time, kyou.typed_mirekyou.limit_time)
            })
    }


    get_estimate_start_time(): Date | null {
        return this.estimate_start_time
    }

    get_estimate_end_time(): Date | null {
        return this.estimate_end_time
    }

    get_limit_time(): Date | null {
        return this.limit_time
    }

}
