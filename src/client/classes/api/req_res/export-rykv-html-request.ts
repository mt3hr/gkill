'use strict'

import { GkillAPIRequest } from "../gkill-api-request"
import type { FindKyouQuery } from "../find_query/find-kyou-query"

/** メディア（画像・動画・添付ファイル）の埋め込み方 */
export type ExportMediaMode = 'thumbnail' | 'full' | 'none'

/**
 * メディアの予算の既定。**Go 側（req_res.ExportDefaultMediaBudgetBytes）と同じ値にすること。**
 * ずれても例外は出ず、ダイアログの初期値とサーバの既定が食い違うだけになる。
 */
export const EXPORT_DEFAULT_MEDIA_BUDGET_BYTES = 500 * 1024 * 1024
/** 1件あたりの上限の既定。Go 側の ExportDefaultMediaItemLimitBytes と対 */
export const EXPORT_DEFAULT_MEDIA_ITEM_LIMIT_BYTES = 100 * 1024 * 1024

/** 出力の形。sidecar は HTML と同名フォルダの組（ZIP で受け取る） */
export type ExportOutputFormat = 'single' | 'sidecar'

export class ExportRykvHTMLRequest extends GkillAPIRequest {

    title: string

    /**
     * 出力する列の検索条件。画面に出ている順。
     *
     * **サイドバーが生成するのと同じ形のまま送ること**（チェック済みの全タグと「no tags」、
     * 全 rep）。tags / reps の既定は非 null の空配列 = 0件指定であって「絞らない」ではないので、
     * 欠けた条件を焼くと、出力先のサイドバーが初めに生成した条件と食い違った瞬間に列が空になる。
     */
    columns: Array<FindKyouQuery>

    include_attached_tags: boolean

    include_attached_texts: boolean

    include_attached_notifications: boolean

    include_attached_timeiss: boolean

    include_gps_log: boolean

    include_plugin_content: boolean

    include_dnote: boolean

    include_map: boolean

    include_sidebar: boolean

    media_mode: ExportMediaMode

    /**
     * メディアと添付ファイルに使える生バイトの合計。
     *
     * base64 は素のバイト数の 4/3 になるので、出来上がる HTML はこれより大きくなる。
     * 収まらなかったものは「含まれていません」として焼かれ、警告が返る。
     */
    media_budget_bytes: number

    /** メディア・添付ファイル1件あたりの上限。1件で合計を食い潰すのを防ぐ */
    media_item_limit_bytes: number

    output_format: ExportOutputFormat

    constructor() {
        super()
        this.title = ""
        this.columns = []
        this.include_attached_tags = true
        this.include_attached_texts = true
        this.include_attached_notifications = true
        this.include_attached_timeiss = true
        this.include_gps_log = false
        this.include_plugin_content = false
        this.include_dnote = true
        this.include_map = true
        this.include_sidebar = false
        this.media_mode = 'thumbnail'
        this.media_budget_bytes = EXPORT_DEFAULT_MEDIA_BUDGET_BYTES
        this.media_item_limit_bytes = EXPORT_DEFAULT_MEDIA_ITEM_LIMIT_BYTES
        this.output_format = 'single'
    }

}
