'use strict'

import { log_unless_aborted } from '@/classes/abort-error'
import delete_gkill_kyou_cache from '../delete-gkill-cache'
import { InfoBase } from './info-base'
import { GkillError } from '../api/gkill-error'
import { GitCommitLog } from './git-commit-log'
import { IDFKyou } from './idf-kyou'
import { Kmemo } from './kmemo'
import { KC } from './kc'
import { Lantana } from './lantana'
import { Mi } from './mi'
import { Nlog } from './nlog'
import { ReKyou } from './re-kyou'
import { MiReKyou } from './mi-re-kyou'
import { TimeIs } from './time-is'
import { URLog } from './ur-log'
import { InfoIdentifier } from './info-identifier'
import { GkillAPI } from '../api/gkill-api'
import { GetKyouRequest } from '../api/req_res/get-kyou-request'
import { GetKmemoRequest } from '../api/req_res/get-kmemo-request'
import { GetKCRequest } from '../api/req_res/get-kc-request'
import { GetURLogRequest } from '../api/req_res/get-ur-log-request'
import { GetNlogRequest } from '../api/req_res/get-nlog-request'
import { GetTimeisRequest } from '../api/req_res/get-timeis-request'
import { GetMiRequest } from '../api/req_res/get-mi-request'
import { GetLantanaRequest } from '../api/req_res/get-lantana-request'
import { GetGitCommitLogRequest } from '../api/req_res/get-git-commit-log-request'
import { GetReKyouRequest } from '../api/req_res/get-re-kyou-request'
import { GetMiReKyouRequest } from '../api/req_res/get-mi-re-kyou-request'
import { GetIDFKyouRequest } from '../api/req_res/get-idf-kyou-request'
import { GkillErrorCodes } from '../api/message/gkill_error'
import { i18n } from '@/i18n'
import type { FindKyouQuery } from '../api/find_query/find-kyou-query'
import { MiSortType } from '../api/find_query/mi-sort-type'

// data_typeのプレフィックスから「どの型別データを読むか」を1回で決めるための表。
//
// **長いプレフィックスから順に**照合するので、"mirekyou" は "mi" より必ず先に当たる。
// CLAUDE.mdの「mirekyou_* は "mi" で始まるためMiより先に判定する」を、
// ifを並べる順番（書き換えで簡単に壊れる）ではなく構造で保証するための並べ替え。
// 並びは五十音順ならぬアルファベット順。'mi' が 'mirekyou' より前にあるが、
// 下の sort が長い順へ並べ替えるので判定は 'mirekyou' が先になる
const typed_data_prefixes: ReadonlyArray<string> = [
    'git',
    'idf',
    'kc',
    'kmemo',
    'lantana',
    'mi',
    'mirekyou',
    'nlog',
    'rekyou',
    'timeis',
    'urlog',
].sort((a, b) => b.length - a.length)

// resolve_typed_data_prefix は data_type が該当する型別データのプレフィックスを返します。
// どれにも当たらないときは null（＝プラグインKyou）です。
function resolve_typed_data_prefix(data_type: string): string | null {
    for (const prefix of typed_data_prefixes) {
        if (data_type.startsWith(prefix)) {
            return prefix
        }
    }
    return null
}

export class Kyou extends InfoBase {
    is_deleted: boolean
    image_source: string
    // InfoBase の attached_* と同じ理由で遅延確保する(検索応答には含まれない)
    _attached_histories: Array<Kyou> | null
    get attached_histories(): Array<Kyou> {
        if (!this._attached_histories) {
            this._attached_histories = new Array<Kyou>()
        }
        return this._attached_histories
    }
    set attached_histories(value: Array<Kyou>) {
        this._attached_histories = value
    }
    typed_kmemo: Kmemo | null
    typed_kc: KC | null
    typed_urlog: URLog | null
    typed_nlog: Nlog | null
    typed_timeis: TimeIs | null
    typed_mi: Mi | null
    typed_lantana: Lantana | null
    typed_idf_kyou: IDFKyou | null
    typed_git_commit_log: GitCommitLog | null
    typed_rekyou: ReKyou | null
    typed_mirekyou: MiReKyou | null
    // typed_plugin は既存のデータ型に該当しないプラグインKyouの場合にセットされる。
    // rep_name でプラグインを特定し、GetContentHTML でView HTMLを取得する。
    typed_plugin: { rep_name: string } | null
    is_typed_data_loaded: boolean

    async load_attached_histories(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        if (this.data_type.startsWith("git")) {
            return []
        }
        const req = new GetKyouRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_histories = res.kyou_histories
        return new Array<GkillError>()
    }

    async load_all(query?: FindKyouQuery, force_attached = false): Promise<Array<GkillError>> {
        const await_promises = new Array<Promise<Array<GkillError>>>()
        try {
            await_promises.push(this.load_typed_datas(query))
            // load_attached_histories はここでは呼ばない。
            // 直後の load_attached_datas が同じものを読むので、
            // 両方書くと 1件につき /api/get_kyou が2回飛ぶ。
            await_promises.push(this.load_attached_datas(force_attached))
            return await Promise.all(await_promises).then((errors_list) => {
                const errors = new Array<GkillError>()
                errors_list.forEach(e => {
                    errors.push(...e)
                })
                return errors
            })
        } catch (err: unknown) {
            // 中断（画面を離れた・後発の検索に差し替わった）は正常なので出さない
            log_unless_aborted(err)
        }
        return []
    }

    async clear_all(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        errors = errors.concat(await this.clear_attached_datas())
        errors = errors.concat(await this.clear_typed_datas())
        errors = errors.concat(await this.clear_attached_histories())
        return errors
    }

    async load_typed_datas(query?: FindKyouQuery): Promise<Array<GkillError>> {
        if (this.is_typed_data_loaded) {
            return []
        }
        // ReKyou/MiReKyouが参照先を取りに行っている間に置かれる空のKyou。
        // idが空のまま先へ進むとdata_typeも空なので既知のプレフィックスに一つも当たらず、
        // 末尾のフォールバックでプラグインKyouと誤判定される。その結果rep_nameが空のまま
        // Content HTMLを取りに行き、サーバから「プラグインが見つかりません」が返ってしまう。
        // data_typeではなくidで見るのは、プラグインのdata_typeがプラグイン側の申告を
        // そのまま使っており空になりうるため（空のdata_typeを持つ本物のプラグインKyouを潰さない）。
        // is_typed_data_loadedは立てない。中身の入ったKyouに差し替わったときに読み直させる
        if (this.id === "") {
            return []
        }
        // 種別の判定は1回だけ。以前は11本のifを並べたうえで、
        // 末尾のプラグイン判定で同じ11個のstartsWithをもう一度評価していた。
        let errors = new Array<GkillError>()
        switch (resolve_typed_data_prefix(this.data_type)) {
            case 'kmemo':
                errors = errors.concat(await this.load_typed_kmemo(query))
                break
            case 'kc':
                errors = errors.concat(await this.load_typed_kc(query))
                break
            case 'urlog':
                errors = errors.concat(await this.load_typed_urlog(query))
                break
            case 'nlog':
                errors = errors.concat(await this.load_typed_nlog(query))
                break
            case 'timeis':
                errors = errors.concat(await this.load_typed_timeis(query))
                break
            case 'mirekyou':
                errors = errors.concat(await this.load_typed_mirekyou(query))
                break
            case 'mi':
                errors = errors.concat(await this.load_typed_mi(query))
                break
            case 'lantana':
                errors = errors.concat(await this.load_typed_lantana(query))
                break
            case 'idf':
                errors = errors.concat(await this.load_typed_idf_kyou(query))
                break
            case 'git':
                errors = errors.concat(await this.load_typed_git_commit_log(query))
                break
            case 'rekyou':
                errors = errors.concat(await this.load_typed_rekyou(query))
                break
            default:
                // 既知のプレフィックスに一つも当たらない場合のみプラグインKyouとして扱う
                // （APIエラーでload失敗した既知data_typeをプラグインとして誤判定しないため）
                this.typed_plugin = { rep_name: this.rep_name }
                break
        }
        this.is_typed_data_loaded = true
        return errors
    }

    async load_attached_datas(force = false): Promise<Array<GkillError>> {
        const await_promises = new Array<Promise<Array<GkillError>>>()
        try {
            await_promises.push(this.load_attached_tags(force))
            await_promises.push(this.load_attached_texts(force))
            await_promises.push(this.load_attached_notifications(force))
            await_promises.push(this.load_attached_timeis(force))
            await_promises.push(this.load_attached_histories())
            return await Promise.all(await_promises).then((errors_list) => {
                const errors = new Array<GkillError>()
                errors_list.forEach(e => {
                    errors.push(...e)
                })
                return errors
            })
        } catch (err: unknown) {
            // 中断（画面を離れた・後発の検索に差し替わった）は正常なので出さない
            log_unless_aborted(err)
        }
        return []
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        this.attached_histories = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        return super.clear_attached_datas()
    }

    async load_typed_kmemo(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetKmemoRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kmemo(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.kmemo_histories || res.kmemo_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_kmemo
            error.error_message = i18n.global.t('NOT_FOUND_KMEMO_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_kmemo: Kmemo | null = null
        res.kmemo_histories.forEach(kmemo => {
            if (Math.floor(kmemo.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_kmemo = kmemo
            }
        })
        this.typed_kmemo = match_kmemo

        return new Array<GkillError>()
    }

    async load_typed_kc(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetKCRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kc(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.kc_histories || res.kc_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_kc
            error.error_message = i18n.global.t('NOT_FOUND_KC_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_kc: KC | null = null
        res.kc_histories.forEach(kc => {
            if (Math.floor(kc.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_kc = kc
            }
        })
        this.typed_kc = match_kc

        return new Array<GkillError>()
    }

    async load_typed_urlog(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetURLogRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_urlog(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.urlog_histories || res.urlog_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_urlog
            error.error_message = i18n.global.t('NOT_FOUND_URLOG_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_urlog: URLog | null = null
        res.urlog_histories.forEach(urlog => {
            if (Math.floor(urlog.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_urlog = urlog
            }
        })
        this.typed_urlog = match_urlog

        return new Array<GkillError>()
    }

    async load_typed_nlog(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetNlogRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_nlog(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.nlog_histories || res.nlog_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_nlog
            error.error_message = i18n.global.t('NOT_FOUND_NLOG_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_nlog: Nlog | null = null
        res.nlog_histories.forEach(nlog => {
            if (Math.floor(nlog.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_nlog = nlog
            }
        })
        this.typed_nlog = match_nlog

        return new Array<GkillError>()
    }

    async load_typed_timeis(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetTimeisRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_timeis(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.timeis_histories || res.timeis_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_timeis
            error.error_message = i18n.global.t('NOT_FOUND_TIMEIS_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_timeis: TimeIs | null = null
        res.timeis_histories.forEach(timeis => {
            if (Math.floor(timeis.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_timeis = timeis
            }
        })
        this.typed_timeis = match_timeis

        return new Array<GkillError>()
    }

    async load_typed_mi(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetMiRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_mi(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.mi_histories || res.mi_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_mi
            error.error_message = i18n.global.t('NOT_FOUND_MI_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_mi: Mi | null = null
        res.mi_histories.forEach(mi => {
            if (Math.floor(mi.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_mi = mi
            }
        })
        if (!match_mi && res.mi_histories.length > 0) {
            match_mi = res.mi_histories.reduce((latest, mi) =>
                mi.update_time > latest.update_time ? mi : latest
            )
        }
        this.typed_mi = match_mi
        return new Array<GkillError>()
    }

    async load_typed_lantana(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetLantanaRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_lantana(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.lantana_histories || res.lantana_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_lantana
            error.error_message = i18n.global.t('NOT_FOUND_LANTANA_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_lantana: Lantana | null = null
        res.lantana_histories.forEach(lantana => {
            if (Math.floor(lantana.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_lantana = lantana
            }
        })
        this.typed_lantana = match_lantana

        return new Array<GkillError>()
    }

    async load_typed_idf_kyou(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetIDFKyouRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_idf_kyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.idf_kyou_histories || res.idf_kyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_idf_kyou
            error.error_message = i18n.global.t('NOT_FOUND_IDFKYOU_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_idf_kyou: IDFKyou | null = null
        res.idf_kyou_histories.forEach(idf_kyou => {
            if (Math.floor(idf_kyou.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_idf_kyou = idf_kyou
            }
        })
        this.typed_idf_kyou = match_idf_kyou

        return new Array<GkillError>()
    }

    async load_typed_git_commit_log(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetGitCommitLogRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_git_commit_log(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.git_commit_log_histories || res.git_commit_log_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_git_commit_log
            error.error_message = i18n.global.t('NOT_FOUND_GITCOMMITLOG_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        this.typed_git_commit_log = res.git_commit_log_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_rekyou(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetReKyouRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_rekyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.rekyou_histories || res.rekyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_rekyou
            error.error_message = i18n.global.t('NOT_FOUND_REKYOU_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_rekyou: ReKyou | null = null
        res.rekyou_histories.forEach(rekyou => {
            if (Math.floor(rekyou.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_rekyou = rekyou
            }
        })
        this.typed_rekyou = match_rekyou

        return new Array<GkillError>()
    }

    async load_typed_mirekyou(_query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetMiReKyouRequest()
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_mirekyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.mirekyou_histories || res.mirekyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_mi_rekyou
            error.error_message = i18n.global.t('NOT_FOUND_MI_REKYOU_ERROR_MESSAGE')
            return [error]
        }

        // 実体化は API レイヤ(gkill-api.ts の get_* )が済ませている。
        // ここでもう一度 new + hydrate すると、履歴の件数ぶん二重に作ることになる。

        let match_mirekyou: MiReKyou | null = null
        res.mirekyou_histories.forEach(mirekyou => {
            if (Math.floor(mirekyou.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_mirekyou = mirekyou
            }
        })
        if (!match_mirekyou && res.mirekyou_histories.length > 0) {
            match_mirekyou = res.mirekyou_histories.reduce((latest, mirekyou) =>
                mirekyou.update_time > latest.update_time ? mirekyou : latest
            )
        }
        this.typed_mirekyou = match_mirekyou
        return new Array<GkillError>()
    }

    async clear_typed_datas(): Promise<Array<GkillError>> {
        this.typed_kmemo = null
        this.typed_kc = null
        this.typed_urlog = null
        this.typed_nlog = null
        this.typed_timeis = null
        this.typed_mi = null
        this.typed_lantana = null
        this.typed_idf_kyou = null
        this.typed_git_commit_log = null
        this.typed_rekyou = null
        this.typed_mirekyou = null
        this.typed_plugin = null
        // フラグを戻さないと次のload_typed_datas()が冒頭で早期returnしてしまい、
        // 種別データが二度と入らないKyouになる
        this.is_typed_data_loaded = false
        return new Array<GkillError>()
    }

    // 最新のメタ情報を取得したうえで、typedデータ（typed_timeis等）も強制的に再取得する。
    // 表示時点でKyouを最新化しておき、終了操作などでの読み込み待ちをなくすために使う。
    //
    // **これは「この場で自分自身を書き換える」引き直しで、通常の引き直しとは別物。**
    // 表示の更新に使う引き直しは `classes/kyou-reload.ts` の `refresh_kyou` を使うこと
    // （新しいインスタンスを返し、飛行中の引き直しへ合流し、スピナーも出る）。
    // ここが in-place なのは、唯一の呼び出し元である TimeIsView が
    // 「親から渡された Kyou そのもの」を温めておく必要があるため。
    // その副作用として親の `is_typed_data_loaded` が途中で倒れるので、
    // KyouView は自分が始めた読み込みだけを追う（`use-kyou-view.ts` の `is_typed_datas_loading`）。
    async reload_with_typed_datas(query?: FindKyouQuery): Promise<Array<GkillError>> {
        // ServiceWorker が get_kyou をキャッシュ優先で返すので、消してから引き直す。
        // これが無いと「最新化した」つもりで古い応答をそのまま読むことがある
        try {
            await delete_gkill_kyou_cache(this.id)
        } catch (_e) {
            // Cache API が使えない環境ではスキップ
        }
        const errors = await this.reload(true, query)
        this.is_typed_data_loaded = false
        return errors.concat(await this.load_typed_datas(query))
    }

    async reload(is_updated_info: boolean, query?: FindKyouQuery): Promise<Array<GkillError>> {
        const req = new GetKyouRequest()
        req.abort_controller = this.abort_controller
        if (!is_updated_info) {
            req.update_time = this.update_time
        }
        req.id = this.id

        const res = await GkillAPI.get_gkill_api().get_kyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        const latest_kyou = res.kyou_histories[0]
        if (!latest_kyou) {
            return new Array<GkillError>()
        }
        this.is_deleted = latest_kyou.is_deleted
        this.id = latest_kyou.id
        this.update_time = latest_kyou.update_time
        this.data_type = latest_kyou.data_type
        this.create_time = latest_kyou.create_time
        this.create_app = latest_kyou.create_app
        this.create_device = latest_kyou.create_device
        this.create_user = latest_kyou.create_user
        this.update_app = latest_kyou.update_app
        this.update_device = latest_kyou.update_device
        this.update_user = latest_kyou.update_user
        this.image_source = latest_kyou.image_source

        if (query) {
            // MiReKyouもMiと同じ並び替え規則に従うが、data_typeの接頭辞だけ変える
            const is_mi_rekyou = this.data_type.startsWith("mirekyou")
            if (is_mi_rekyou) {
                await this.load_typed_mirekyou()
            } else {
                await this.load_typed_mi()
            }
            const typed_task: Mi | MiReKyou | null = is_mi_rekyou ? this.typed_mirekyou : this.typed_mi
            if (typed_task) {
                const prefix = is_mi_rekyou ? "mirekyou" : "mi"
                switch (query.mi_sort_type) {
                    case MiSortType.estimate_start_time:
                        if (typed_task.estimate_start_time) {
                            this.related_time = typed_task.estimate_start_time
                            this.data_type = prefix + "_start"
                        } else {
                            this.related_time = this.create_time
                            this.data_type = prefix + "_create"
                        }
                        break;
                    case MiSortType.estimate_end_time:
                        if (typed_task.estimate_end_time) {
                            this.related_time = typed_task.estimate_end_time
                            this.data_type = prefix + "_end"
                        } else {
                            this.related_time = this.create_time
                            this.data_type = prefix + "_create"
                        }
                        break;
                    case MiSortType.limit_time:
                        if (typed_task.limit_time) {
                            this.related_time = typed_task.limit_time
                            this.data_type = prefix + "_limit"
                        } else {
                            this.related_time = this.create_time
                            this.data_type = prefix + "_create"
                        }
                        break;
                    default:
                        // 作成日時ソート。サーバのoverrideKyousもmi.CreateTimeを入れるので合わせる。
                        // ここをupdate_timeにすると、この行だけget_kyousで来た隣接行と
                        // 表示時刻も並び位置もずれる
                        this.related_time = this.create_time
                        this.data_type = prefix + "_create"
                        break;
                }
            }
        }
        return new Array<GkillError>()
    }

    clone(): Kyou {
        const cloned_kyou = new Kyou()
        cloned_kyou.is_deleted = this.is_deleted
        cloned_kyou.is_checked_kyou = this.is_checked_kyou
        cloned_kyou.id = this.id
        cloned_kyou.rep_name = this.rep_name
        cloned_kyou.related_time = this.related_time
        cloned_kyou.data_type = this.data_type
        cloned_kyou.create_time = this.create_time
        cloned_kyou.create_app = this.create_app
        cloned_kyou.create_device = this.create_device
        cloned_kyou.create_user = this.create_user
        cloned_kyou.update_time = this.update_time
        cloned_kyou.update_app = this.update_app
        cloned_kyou.update_device = this.update_device
        cloned_kyou.update_user = this.update_user
        cloned_kyou.image_source = this.image_source
        cloned_kyou.typed_kmemo = this.typed_kmemo
        cloned_kyou.typed_kc = this.typed_kc
        cloned_kyou.typed_urlog = this.typed_urlog
        cloned_kyou.typed_nlog = this.typed_nlog
        cloned_kyou.typed_timeis = this.typed_timeis
        cloned_kyou.typed_mi = this.typed_mi
        cloned_kyou.typed_lantana = this.typed_lantana
        cloned_kyou.typed_idf_kyou = this.typed_idf_kyou
        cloned_kyou.typed_git_commit_log = this.typed_git_commit_log
        cloned_kyou.typed_rekyou = this.typed_rekyou
        cloned_kyou.typed_mirekyou = this.typed_mirekyou
        cloned_kyou.typed_plugin = this.typed_plugin
        cloned_kyou.is_typed_data_loaded = this.is_typed_data_loaded
        cloned_kyou._attached_tags = this._attached_tags ? this._attached_tags.slice() : null
        cloned_kyou._attached_texts = this._attached_texts ? this._attached_texts.slice() : null
        cloned_kyou._attached_notifications = this._attached_notifications ? this._attached_notifications.slice() : null
        cloned_kyou._attached_timeis_kyou = this._attached_timeis_kyou ? this._attached_timeis_kyou.slice() : null
        cloned_kyou.is_attached_tags_loaded = this.is_attached_tags_loaded
        cloned_kyou.is_attached_texts_loaded = this.is_attached_texts_loaded
        cloned_kyou.is_attached_notifications_loaded = this.is_attached_notifications_loaded
        cloned_kyou.is_attached_timeis_loaded = this.is_attached_timeis_loaded
        return cloned_kyou
    }

    generate_info_identifier(): InfoIdentifier {
        const info_identifier = new InfoIdentifier()
        info_identifier.id = this.id
        info_identifier.create_time = this.create_time
        info_identifier.update_time = this.update_time
        return info_identifier
    }

    constructor() {
        super()
        this.is_deleted = false
        this.image_source = ""
        this._attached_histories = null
        this.typed_kmemo = null
        this.typed_kc = null
        this.typed_urlog = null
        this.typed_nlog = null
        this.typed_timeis = null
        this.typed_mi = null
        this.typed_lantana = null
        this.typed_idf_kyou = null
        this.typed_git_commit_log = null
        this.typed_rekyou = null
        this.typed_mirekyou = null
        this.typed_plugin = null
        this.is_typed_data_loaded = false
    }

}


