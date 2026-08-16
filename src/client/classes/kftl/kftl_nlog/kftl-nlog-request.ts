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
import type { KFTLNlogBlock } from './kftl-nlog-block'

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

}
