'use strict'

import { GkillAPI } from "../api/gkill-api"
import { GkillError } from "../api/gkill-error"
import { generate_plaing_timeis_query } from "../api/find_query/generate-plaing-timeis-query"
import { GetApplicationConfigRequest } from "../api/req_res/get-application-config-request"
import { GetKyousRequest } from "../api/req_res/get-kyous-request"
import { GetNotificationsByTargetIDRequest } from "../api/req_res/get-notifications-by-target-id-request"
import { GetTagsByTargetIDRequest } from "../api/req_res/get-tags-by-target-id-request"
import { GetTextsByTargetIDRequest } from "../api/req_res/get-texts-by-target-id-request"
import type { Kyou } from "./kyou"
import type { Notification } from "./notification"
import type { Tag } from "./tag"
import type { Text } from "./text"

export abstract class InfoBase {
    // AbortControllerは初アクセスまで生成しない。
    // 数十万件の検索応答を実体化するとき、1件ごとの生成コスト(WebIDLラッパ)が
    // 合計で数百msを占めていたため。ほとんどのインスタンスは一度も使わない。
    // ES private(#)はVueのreactive Proxy越しのthisで壊れる。
    // TS privateはVueのUnwrapRefがprivateメンバーを落とすため
    // `Ref<Array<Kyou>> = ref(new Array<Kyou>())` が全部型エラーになる。
    // どちらも使えないのでunderscore公開フィールド。直接触らずゲッターを使うこと。
    _abort_controller: AbortController | null

    get abort_controller(): AbortController {
        if (!this._abort_controller) {
            this._abort_controller = new AbortController()
        }
        return this._abort_controller
    }

    set abort_controller(abort_controller: AbortController) {
        this._abort_controller = abort_controller
    }

    is_deleted: boolean
    id: string
    rep_name: string
    related_time: Date
    data_type: string
    create_time: Date
    create_app: string
    create_device: string
    create_user: string
    update_time: Date
    update_app: string
    update_user: string
    update_device: string
    // 検索応答(get_kyous)には attached_* が1つも含まれないので、30万件の実体化では
    // これらの配列は一度も書かれない。それでもコンストラクタで確保すると1件につき4本、
    // 30万件で120万個の使い捨てになる。`_abort_controller` と同じく
    // underscore 公開フィールド + ゲッターで遅延確保する。
    //
    // TS private は Vue の UnwrapRef が落として `Ref<Array<Kyou>>` への代入が型エラーになり、
    // ES private(#) は reactive Proxy 越しの this で壊れるので、どちらも使えない。
    // 直接 `_` 付きを触らず、必ずゲッター/セッター越しに読み書きすること。
    _attached_tags: Array<Tag> | null
    get attached_tags(): Array<Tag> {
        if (!this._attached_tags) {
            this._attached_tags = new Array<Tag>()
        }
        return this._attached_tags
    }
    set attached_tags(value: Array<Tag>) {
        this._attached_tags = value
    }
    _attached_texts: Array<Text> | null
    get attached_texts(): Array<Text> {
        if (!this._attached_texts) {
            this._attached_texts = new Array<Text>()
        }
        return this._attached_texts
    }
    set attached_texts(value: Array<Text>) {
        this._attached_texts = value
    }
    _attached_notifications: Array<Notification> | null
    get attached_notifications(): Array<Notification> {
        if (!this._attached_notifications) {
            this._attached_notifications = new Array<Notification>()
        }
        return this._attached_notifications
    }
    set attached_notifications(value: Array<Notification>) {
        this._attached_notifications = value
    }
    _attached_timeis_kyou: Array<Kyou> | null
    get attached_timeis_kyou(): Array<Kyou> {
        if (!this._attached_timeis_kyou) {
            this._attached_timeis_kyou = new Array<Kyou>()
        }
        return this._attached_timeis_kyou
    }
    set attached_timeis_kyou(value: Array<Kyou>) {
        this._attached_timeis_kyou = value
    }
    is_checked_kyou: boolean
    is_attached_tags_loaded: boolean
    is_attached_texts_loaded: boolean
    is_attached_notifications_loaded: boolean
    is_attached_timeis_loaded: boolean

    async load_all(): Promise<Array<GkillError>> {
        return await this.load_attached_datas()
    }

    async load_attached_tags(force = false): Promise<Array<GkillError>> {
        if (this.is_attached_tags_loaded && !force) {
            return []
        }
        const errors = new Array<GkillError>()
        const req = new GetTagsByTargetIDRequest()
        req.abort_controller = this.abort_controller

        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_tags_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_tags = res.tags
        this.is_attached_tags_loaded = true
        return errors
    }

    async load_attached_texts(force = false): Promise<Array<GkillError>> {
        if (this.is_attached_texts_loaded && !force) {
            return []
        }
        const errors = new Array<GkillError>()
        const req = new GetTextsByTargetIDRequest()
        req.abort_controller = this.abort_controller

        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_texts_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_texts = res.texts
        this.is_attached_texts_loaded = true
        return errors
    }

    async load_attached_notifications(force = false): Promise<Array<GkillError>> {
        if (this.is_attached_notifications_loaded && !force) {
            return []
        }
        const errors = new Array<GkillError>()
        const req = new GetNotificationsByTargetIDRequest()
        req.abort_controller = this.abort_controller

        req.target_id = this.id
        const res = await GkillAPI.get_gkill_api().get_notifications_by_target_id(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_notifications = res.notifications
        this.is_attached_notifications_loaded = true
        return errors
    }

    async load_attached_timeis(force = false): Promise<Array<GkillError>> {
        if (this.is_attached_timeis_loaded && !force) {
            return []
        }
        const errors = new Array<GkillError>()
        const req = new GetKyousRequest()
        req.abort_controller = this.abort_controller

        // アプリ設定はKyou 1件ごとに取り直すものではない。
        // get_application_config() は毎回ネットワークに出たうえ、
        // 内部の load_all() で get_all_rep_names / get_all_tag_names /
        // get_mi_board_list を引くので、1件につき5〜6往復増えてしまう。
        // 各ページが読み込み後に set_saved_application_config しているので、
        // それがあればそれを使う（共有ページは null を返すので従来どおり取りに行く）。
        const gkill_api = GkillAPI.get_gkill_api()
        const application_config = gkill_api.get_saved_application_config()
            ?? (await gkill_api.get_application_config(new GetApplicationConfigRequest())).application_config
        // 検索条件の組み立ては実行中画面・KFTL終了候補と共通。
        // ApplicationConfigのカスタム検索条件（plaing_timeis_json_data）もそこで適用される
        req.query = generate_plaing_timeis_query(application_config, this.related_time)

        const res = await GkillAPI.get_gkill_api().get_kyous(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        // 1件ずつ await すると件数×RTTかかるので並列で読む
        await Promise.all(res.kyous.map(kyou => kyou.load_typed_timeis()))
        this.attached_timeis_kyou = res.kyous
        this.is_attached_timeis_loaded = true
        return errors
    }

    async load_attached_datas(force = false): Promise<Array<GkillError>> {
        const await_promises = new Array<Promise<Array<GkillError>>>()
        await_promises.push(this.load_attached_tags(force))
        await_promises.push(this.load_attached_texts(force))
        await_promises.push(this.load_attached_notifications(force))
        await_promises.push(this.load_attached_timeis(force))
        return Promise.all(await_promises).then((errors_list) => {
            const errors = new Array<GkillError>()
            errors_list.forEach(e => {
                errors.push(...e)
            })
            return errors
        })
    }

    async clear_attached_tags(): Promise<Array<GkillError>> {
        this.attached_tags = []
        this.is_attached_tags_loaded = false
        return new Array<GkillError>()
    }

    async clear_attached_texts(): Promise<Array<GkillError>> {
        this.attached_texts = []
        this.is_attached_texts_loaded = false
        return new Array<GkillError>()
    }

    async clear_attached_notifications(): Promise<Array<GkillError>> {
        this.attached_notifications = []
        this.is_attached_notifications_loaded = false
        return new Array<GkillError>()
    }

    async clear_attached_timeis(): Promise<Array<GkillError>> {
        this.attached_timeis_kyou = []
        this.is_attached_timeis_loaded = false
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        this.attached_tags = []
        this.attached_texts = []
        this.attached_notifications = []
        this.attached_timeis_kyou = []
        this.is_attached_tags_loaded = false
        this.is_attached_texts_loaded = false
        this.is_attached_notifications_loaded = false
        this.is_attached_timeis_loaded = false
        return new Array<GkillError>()
    }

    abstract clone(): InfoBase

    constructor() {
        this._abort_controller = null
        this.is_deleted = false
        this.id = ""
        this.rep_name = ""
        this.related_time = new Date(0)
        this.data_type = ""
        this.create_time = new Date(0)
        this.create_app = ""
        this.create_device = ""
        this.create_user = ""
        this.update_time = new Date(0)
        this.update_app = ""
        this.update_user = ""
        this.update_device = ""
        this._attached_tags = null
        this._attached_texts = null
        this._attached_notifications = null
        this._attached_timeis_kyou = null
        this.is_checked_kyou = false
        this.is_attached_tags_loaded = false
        this.is_attached_texts_loaded = false
        this.is_attached_notifications_loaded = false
        this.is_attached_timeis_loaded = false
    }
}
