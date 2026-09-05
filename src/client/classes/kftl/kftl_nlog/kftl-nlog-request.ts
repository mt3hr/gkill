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
import { KFTLNlogBlock } from './kftl-nlog-block'
import { find_existing_anchors } from '../kftl_repeat/kftl-repeat-duplicate'
import type { RepeatSpec } from '../kftl_repeat/kftl-repeat-spec'
import { shift_days } from '../kftl_repeat/kftl-repeat-spec'

/**
 * 支払い1件(品名と金額のペア1組)ぶんのリクエスト。
 *
 * 1つの `ーん` ブロックからは支払いの数だけこのリクエストが出る。
 * request_id をそのまま Nlog の id にしているので、基底が書くタグ・テキストの target_id が
 * その支払いを正しく指す(以前はブロックで1つのリクエストにまとめており、しかも Nlog の id を
 * 毎回採番し直していたので、付けたタグがどの Nlog にも紐づいていなかった)。
 */
export class KFTLNlogRequest extends KFTLRequest {

    shop_name: string

    title: string

    // 金額行をまだ書いていない状態を null で表す
    amount: number | null

    private block: KFTLNlogBlock

    constructor(request_id: string, context: KFTLStatementLineContext, block: KFTLNlogBlock) {
        super(request_id, context)
        this.block = block
        this.shop_name = block.shop_name
        this.title = ""
        this.amount = null
    }

    /**
     * 関連時刻はブロック全体で共有する。
     *
     * `？`行をブロックの中のどこに書いても、そのブロックの全支払いが同じ時刻になる
     */
    override get_related_time(): Date | null {
        if (this.block.related_time != null) {
            return new Date(this.block.related_time.getTime())
        }
        return super.get_related_time()
    }

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        // 末尾の改行が品名行として解釈されただけの空の支払い。
        // エラーにせず、支払いも作らない
        if (this.title === "" && this.amount === null) {
            return new Array<GkillError>()
        }

        let errors = Array<GkillError>()
        if (this.amount === null) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.nlog_title_amount_count_not_equal
            error.error_message = i18n.global.t("KFTL_NLOG_INVALID_RECORD_COUNT_MESSAGE_TITLE")
            errors.push(error)
            return errors
        }

        await super.do_request(gkill_api, application_config).then(super_errors => errors = errors.concat(super_errors))

        if (this.title == "" && this.amount == 0 && this.shop_name == "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.skiped_no_content_nlog
            error.error_message = i18n.global.t("KFTL_NLOG_BLANK_SKIP_SAVE_MESSAGE_TITLE")
            errors.push(error)
        }

        const time = this.get_related_time() ? this.get_related_time()! : new Date(Date.now())
        const req = new AddNlogRequest()
        req.tx_id = this.get_tx_id()
        const now = new Date(Date.now())

        req.nlog.id = this.get_request_id()
        req.nlog.shop = this.shop_name
        req.nlog.amount = this.amount
        req.nlog.title = this.title
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
            } else {
                // 成功したものだけ積む。実体は commit_tx のあとに引き直される
                this.add_registered_kyou_id(req.nlog.id)
            }
        })
        return errors
    }

    set_shop_name(shop_name: string): void {
        this.shop_name = shop_name
    }

    set_title(title: string): void {
        this.title = title
    }

    set_amount(amount: number): void {
        this.amount = amount
    }


    /**
     * 繰り返しの指定は支出ブロックで共有する。
     *
     * 「？？」をブロックの中のどこに書いても、そのブロックの**全支払いが1つの繰り返しグループ**
     * になる（店名・関連時刻と同じ扱い）。get_related_time がブロックを見ているのと同じ形。
     */
    override get_repeat_spec(): RepeatSpec | null {
        return this.block.repeat_spec
    }

    override set_repeat_spec(spec: RepeatSpec): void {
        this.block.repeat_spec = spec
    }

    /** ブロック共有の関連時刻を基準にする。**呼ぶと確定させる。** */
    override anchor_time_for_repeat(): Date | null {
        if (this.block.related_time === null) {
            this.block.related_time = super.get_related_time()
        }
        return this.block.related_time
    }

    /**
     * **ブロックも複製する。** 共有したままだと、回ごとにずらしたはずの関連時刻を
     * 最後の1回が上書きし、全レコードが同じ日時になる（関連時刻の実体はブロック側にある）。
     */
    override clone_for_repeat(new_request_id: string, day_shift: number): KFTLRequest {
        const block_copy = new KFTLNlogBlock(this.block.block_target_id)
        block_copy.shop_name = this.block.shop_name
        block_copy.related_time = this.block.related_time === null ? null : shift_days(this.block.related_time, day_shift)
        block_copy.repeat_spec = null // 複製を再展開しない
        const cloned = new KFTLNlogRequest(new_request_id, this.get_context(), block_copy)
        this.copy_base_state_for_repeat(cloned, day_shift)
        cloned.shop_name = this.shop_name
        cloned.title = this.title
        cloned.amount = this.amount
        return cloned
    }

    /**
     * 支払いごとに見る。**ブロック単位で見てはいけない** ――
     * 同じ買い物でも品名が違えば別の記録なので、片方だけ既にある状態がふつうに起きる。
     */
    override async find_existing_for_repeat(gkill_api: GkillAPI, _application_config: ApplicationConfig | null, from: Date, to: Date): Promise<Set<number>> {
        return find_existing_anchors(gkill_api, from, to, false,
            (data_type) => data_type === "nlog",
            (kyou) => (kyou.typed_nlog !== null && kyou.typed_nlog.title === this.title && kyou.typed_nlog.shop === this.shop_name) ? kyou.typed_nlog.related_time : null)
    }

}
