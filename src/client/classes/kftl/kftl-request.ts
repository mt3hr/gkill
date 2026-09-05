'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { GkillError } from '../api/gkill-error'
import { AddTagRequest } from '../api/req_res/add-tag-request'
import { AddTextRequest } from '../api/req_res/add-text-request'
import type { ApplicationConfig } from '../datas/config/application-config'
import delete_gkill_kyou_cache from '../delete-gkill-cache'
import { KFTLRequestBase } from './kftl-request-base'
import type { KFTLStatementLineContext } from './kftl-statement-line-context'
import { shift_days, type RepeatSpec } from './kftl_repeat/kftl-repeat-spec'

export type KFTLRequestResultKind = 'registered' | 'updated'

export interface KFTLRequestResult {
    id: string
    kind: KFTLRequestResultKind
}

export abstract class KFTLRequest extends KFTLRequestBase {

    private request_id: string

    private result_kyou_ids: Array<KFTLRequestResult>

    private tags: Array<string>

    private current_text_id: string | null

    private texts_map: Map<string, string>

    private related_time: Date | null

    private context: KFTLStatementLineContext

    /**
     * 「？？」ブロックの指定。同じ参照を共有しているリクエストが1つの繰り返しグループになる
     * （支出ブロックの全支払いなど）。
     */
    private repeat_spec: RepeatSpec | null

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super()
        this.request_id = request_id
        this.result_kyou_ids = new Array<KFTLRequestResult>()
        this.tags = new Array<string>()
        this.current_text_id = ""
        this.texts_map = new Map<string, string>()
        this.related_time = null
        this.context = context
        this.repeat_spec = null
    }

    async do_request(gkill_api: GkillAPI, application_config: ApplicationConfig): Promise<Array<GkillError>> {
        let errors = Array<GkillError>()
        const time = this.get_related_time() != null ? this.get_related_time()! : new Date(Date.now())
        const now = new Date(Date.now())

        for (let i = 0; i < this.tags.length; i++) {
            const tag = this.tags[i]
            const req = new AddTagRequest()
            req.tx_id = this.get_tx_id()

            req.tag.id = gkill_api.generate_uuid()
            req.tag.tag = tag
            req.tag.target_id = this.get_request_id()
            req.tag.related_time = time
            req.tag.create_app = "gkill_kftl"
            req.tag.create_device = application_config.device
            req.tag.create_time = now
            req.tag.create_user = application_config.user_id
            req.tag.update_app = "gkill_kftl"
            req.tag.update_device = application_config.device
            req.tag.update_time = now
            req.tag.update_user = application_config.user_id
            await delete_gkill_kyou_cache(req.tag.id)
            await delete_gkill_kyou_cache(req.tag.target_id)
            await gkill_api.add_tag(req).then((res) => {
                if (res.errors && res.errors.length !== 0) {
                    errors = errors.concat(res.errors)
                }
            })
        }
        for (const text_entry of this.texts_map) {
            const id = text_entry[0]
            const text = text_entry[1]

            const req = new AddTextRequest()
            req.tx_id = this.get_tx_id()

            req.text.id = id
            req.text.target_id = this.get_request_id()
            req.text.text = text
            req.text.related_time = time
            req.text.create_app = "gkill_kftl"
            req.text.create_device = application_config.device
            req.text.create_time = now
            req.text.create_user = application_config.user_id
            req.text.update_app = "gkill_kftl"
            req.text.update_device = application_config.device
            req.text.update_time = now
            req.text.update_user = application_config.user_id
            await delete_gkill_kyou_cache(req.text.id)
            await delete_gkill_kyou_cache(req.text.target_id)
            await gkill_api.add_text(req).then((res) => {
                if (res.errors && res.errors.length !== 0) {
                    errors = errors.concat(res.errors)
                }
            })
        }
        return errors
    }

    get_request_id(): string {
        return this.request_id
    }

    /**
     * 作った / 更新した Kyou の id。
     *
     * tx中の add_* は added_kyou を返せない（TXID指定時は一時リポジトリにしか無い）ので、
     * Kyou本体ではなくidだけ積んでおく。実体は commit_tx のあとに
     * use-kftl-view.ts が get_kyou で引き直す。
     * 積むのは実際に登録・更新が成功した分だけにすること。
     */
    protected add_registered_kyou_id(id: string): void {
        this.result_kyou_ids.push({ id: id, kind: 'registered' })
    }

    protected add_updated_kyou_id(id: string): void {
        this.result_kyou_ids.push({ id: id, kind: 'updated' })
    }

    get_result_kyou_ids(): ReadonlyArray<KFTLRequestResult> {
        return this.result_kyou_ids
    }

    get_tags(): Array<string> {
        return this.tags
    }

    /**
     * このリクエストが設定するタスクの板名。Mi以外は空。
     * 送信前に「まだ実在しない板名」を検出するために使う（use-kftl-view.ts の collect_unknown_mi_boards）
     */
    get_mi_board_name(): string {
        return ""
    }

    set_tags(tags: Array<string>): void {
        this.tags = tags
    }

    get_texts(): Array<string> {
        const texts = Array<string>()
        this.texts_map.forEach(text => {
            texts.push(text)
        });
        return texts
    }

    set_texts(texts: Array<string>): void {
        this.texts_map.clear()
        texts.forEach(text => {
            this.texts_map.set(GkillAPI.get_gkill_api().generate_uuid(), text)
        });
    }

    /**
     * `？`行で明示指定された関連時刻。指定が無ければ null。
     *
     * get_related_time() は未指定でも「今」を返すので、「指定されたかどうか」の判定には使えない。
     * 支出ブロックが「ブロックの前に書かれた関連時刻」を取り込むために使う
     */
    get_raw_related_time(): Date | null {
        return this.related_time
    }

    get_related_time(): Date | null {
        let time = new Date(Date.now())
        if (this.related_time != null) {
            time = new Date(this.related_time.getTime())
        } else {
            for (let i = 0; i < this.context.get_add_second().valueOf(); i++) {
                time.setTime(time.getTime() + 1000)
            }
        }
        return time
    }

    set_related_time(time: Date | null): void {
        this.related_time = time
    }

    add_tag(tag: string): void {
        this.tags.push(tag)
    }

    get_current_text_id(): string | null {
        return this.current_text_id
    }

    set_current_text_id(text_id: string | null): void {
        this.current_text_id = text_id
    }

    add_text_line(text_id: string, text_line: string): void {
        let text = this.texts_map.get(text_id)
        if (!text) {
            text = `${text_line}`
        } else {
            text += `\n${text_line}`
        }
        this.texts_map.set(text_id, text)
    }

    get_tx_id(): string {
        return this.context.get_tx_id()
    }

    get_context(): KFTLStatementLineContext {
        return this.context
    }

    // ─── 繰り返し（「？？」ブロック）────────────────────────────────────────

    get_repeat_spec(): RepeatSpec | null {
        return this.repeat_spec
    }

    /**
     * 既定で受け入れる。繰り返しても意味が無い型
     * （打刻開始のみ・打刻終了の2種・プロトタイプ）だけが override して弾く。
     */
    set_repeat_spec(spec: RepeatSpec): void {
        this.repeat_spec = spec
    }

    /**
     * 繰り返しの基準になる日時。日時欄が1つも無ければ null。
     *
     * **呼ぶと確定させる。** get_related_time() は未設定だと「今」を返すので、
     * 確定させないと複製した側がそれぞれ現在時刻を引き、全回が同じ日時になる。
     *
     * 日時欄を複数持つ型（タスク / リポストタスク）と、開始時刻が主軸の型（打刻）は override する。
     */
    anchor_time_for_repeat(): Date | null {
        if (this.get_raw_related_time() === null) {
            this.set_related_time(this.get_related_time())
        }
        return this.get_raw_related_time()
    }

    /**
     * 繰り返しの1回ぶんを作る。
     *
     * **基底に既定実装を置かない** ―― 置くと新しい型で override を忘れても通ってしまい、
     * 日時のずれない複製が黙って書かれる。型を足したらコンパイルエラーで気づけるようにしてある。
     */
    abstract clone_for_repeat(new_request_id: string, day_shift: number): KFTLRequest

    /**
     * 「もう同じ記録があるか」を調べる（3行目が no のときだけ呼ばれる）。
     * 返すのは既にある記録のアンカー時刻（epoch ミリ秒）の集合。
     */
    abstract find_existing_for_repeat(gkill_api: GkillAPI, application_config: ApplicationConfig | null, from: Date, to: Date): Promise<Set<number>>

    /**
     * 繰り返し複製の土台。**ここで採り直すものを1つでも落とすと静かに壊れる。**
     *
     * テキストIDは set_texts が採り直す。使い回すと同じIDのテキストを回数ぶん書くことになり、
     * append-only なので最後の1件以外が消える。
     */
    protected copy_base_state_for_repeat(cloned: KFTLRequest, day_shift: number): void {
        cloned.set_tags([...this.get_tags()])
        cloned.set_texts([...this.get_texts()])
        const related_time = this.get_raw_related_time()
        cloned.set_related_time(related_time === null ? null : shift_days(related_time, day_shift))
        // 複製に繰り返し指定を持たせない。持たせると展開が再帰する
        cloned.repeat_spec = null
    }
}


