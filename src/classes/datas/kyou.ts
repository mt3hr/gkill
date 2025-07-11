'use strict'

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
import { GetIDFKyouRequest } from '../api/req_res/get-idf-kyou-request'
import { GkillErrorCodes } from '../api/message/gkill_error'

export class Kyou extends InfoBase {
    is_deleted: boolean
    image_source: string
    attached_histories: Array<Kyou>
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

    async load_attached_histories(): Promise<Array<GkillError>> {
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

    async load_all(): Promise<Array<GkillError>> {
        const awaitPromises = new Array<Promise<any>>()
        try {
            awaitPromises.push(this.load_typed_datas())
            this.load_attached_histories()
            this.load_attached_datas()
            return Promise.all(awaitPromises).then((errors_list) => {
                const errors = new Array<GkillError>()
                errors_list.forEach(e => {
                    errors.push(...e)
                })
                return errors
            })
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
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

    async load_typed_datas(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        if (this.data_type.startsWith("kmemo")) {
            const e = await this.load_typed_kmemo()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("kc")) {
            const e = await this.load_typed_kc()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("urlog")) {
            const e = await this.load_typed_urlog()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("nlog")) {
            const e = await this.load_typed_nlog()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("timeis")) {
            const e = await this.load_typed_timeis()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("mi")) {
            const e = await this.load_typed_mi()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("lantana")) {
            const e = await this.load_typed_lantana()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("idf")) {
            const e = await this.load_typed_idf_kyou()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("git")) {
            const e = await this.load_typed_git_commit_log()
            errors = errors.concat(e)
        }
        if (this.data_type.startsWith("rekyou")) {
            const e = await this.load_typed_rekyou()
            errors = errors.concat(e)
        }
        return errors
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        const awaitPromises = new Array<Promise<any>>()
        try {
            awaitPromises.push(this.load_attached_tags())
            awaitPromises.push(this.load_attached_texts())
            awaitPromises.push(this.load_attached_notifications())
            awaitPromises.push(this.load_attached_timeis())
            awaitPromises.push(this.load_attached_histories())
            return Promise.all(awaitPromises).then((errors_list) => {
                const errors = new Array<GkillError>()
                errors_list.forEach(e => {
                    errors.push(...e)
                })
                return errors
            })
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
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

    async load_typed_kmemo(): Promise<Array<GkillError>> {
        const req = new GetKmemoRequest()
        req.rep_name = this.rep_name
        req.abort_controller = this.abort_controller

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kmemo(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.kmemo_histories || res.kmemo_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_kmemo
            error.error_message = "Kmemoが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.kmemo_histories.length; i++) {
            const kmemo = new Kmemo()
            for (const key in res.kmemo_histories[i]) {
                (kmemo as any)[key] = (res.kmemo_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (kmemo as any)[key]) {
                    (kmemo as any)[key] = new Date((kmemo as any)[key])
                }
            }
            res.kmemo_histories[i] = kmemo
        }

        let match_kmemo: Kmemo | null = null
        res.kmemo_histories.forEach(kmemo => {
            if (Math.floor(kmemo.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_kmemo = kmemo
            }
        })
        this.typed_kmemo = match_kmemo

        return new Array<GkillError>()
    }

    async load_typed_kc(): Promise<Array<GkillError>> {
        const req = new GetKCRequest()
        req.abort_controller = this.abort_controller
        req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kc(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.kc_histories || res.kc_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_kc
            error.error_message = "KCが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.kc_histories.length; i++) {
            const kc = new KC()
            for (const key in res.kc_histories[i]) {
                (kc as any)[key] = (res.kc_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (kc as any)[key]) {
                    (kc as any)[key] = new Date((kc as any)[key])
                }
            }
            res.kc_histories[i] = kc
        }

        let match_kc: KC | null = null
        res.kc_histories.forEach(kc => {
            if (Math.floor(kc.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_kc = kc
            }
        })
        this.typed_kc = match_kc

        return new Array<GkillError>()
    }

    async load_typed_urlog(): Promise<Array<GkillError>> {
        const req = new GetURLogRequest()
        req.abort_controller = this.abort_controller
        req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_urlog(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.urlog_histories || res.urlog_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_urlog
            error.error_message = "URLogが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.urlog_histories.length; i++) {
            const urlog = new URLog()
            for (const key in res.urlog_histories[i]) {
                (urlog as any)[key] = (res.urlog_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (urlog as any)[key]) {
                    (urlog as any)[key] = new Date((urlog as any)[key])
                }
            }
            res.urlog_histories[i] = urlog
        }

        let match_urlog: URLog | null = null
        res.urlog_histories.forEach(urlog => {
            if (Math.floor(urlog.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_urlog = urlog
            }
        })
        this.typed_urlog = match_urlog

        return new Array<GkillError>()
    }

    async load_typed_nlog(): Promise<Array<GkillError>> {
        const req = new GetNlogRequest()
        req.abort_controller = this.abort_controller
        req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_nlog(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.nlog_histories || res.nlog_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_nlog
            error.error_message = "Nlogが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.nlog_histories.length; i++) {
            const nlog = new Nlog()
            for (const key in res.nlog_histories[i]) {
                (nlog as any)[key] = (res.nlog_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (nlog as any)[key]) {
                    (nlog as any)[key] = new Date((nlog as any)[key])
                }
            }
            res.nlog_histories[i] = nlog
        }

        let match_nlog: Nlog | null = null
        res.nlog_histories.forEach(nlog => {
            if (Math.floor(nlog.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_nlog = nlog
            }
        })
        this.typed_nlog = match_nlog

        return new Array<GkillError>()
    }

    async load_typed_timeis(): Promise<Array<GkillError>> {
        const req = new GetTimeisRequest()
        req.abort_controller = this.abort_controller
        req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_timeis(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.timeis_histories || res.timeis_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_timeis
            error.error_message = "TimeIsが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.timeis_histories.length; i++) {
            const timeis = new TimeIs()
            for (const key in res.timeis_histories[i]) {
                (timeis as any)[key] = (res.timeis_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (timeis as any)[key]) {
                    (timeis as any)[key] = new Date((timeis as any)[key])
                }
            }
            res.timeis_histories[i] = timeis
        }

        let match_timeis: TimeIs | null = null
        res.timeis_histories.forEach(timeis => {
            if (Math.floor(timeis.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_timeis = timeis
            }
        })
        this.typed_timeis = match_timeis

        return new Array<GkillError>()
    }

    async load_typed_mi(): Promise<Array<GkillError>> {
        const req = new GetMiRequest()
        req.abort_controller = this.abort_controller
        // req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_mi(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.mi_histories || res.mi_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_mi
            error.error_message = "Miが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.mi_histories.length; i++) {
            const mi = new Mi()
            for (const key in res.mi_histories[i]) {
                (mi as any)[key] = (res.mi_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (mi as any)[key]) {
                    (mi as any)[key] = new Date((mi as any)[key])
                }
            }
            res.mi_histories[i] = mi
        }

        let match_mi: Mi | null = null
        res.mi_histories.forEach(mi => {
            if (Math.floor(mi.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_mi = mi
            }
        })
        this.typed_mi = match_mi

        return new Array<GkillError>()
    }

    async load_typed_lantana(): Promise<Array<GkillError>> {
        const req = new GetLantanaRequest()
        req.abort_controller = this.abort_controller
        req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_lantana(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.lantana_histories || res.lantana_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_lantana
            error.error_message = "Lantanaが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.lantana_histories.length; i++) {
            const lantana = new Lantana()
            for (const key in res.lantana_histories[i]) {
                (lantana as any)[key] = (res.lantana_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (lantana as any)[key]) {
                    (lantana as any)[key] = new Date((lantana as any)[key])
                }
            }
            res.lantana_histories[i] = lantana
        }

        let match_lantana: Lantana | null = null
        res.lantana_histories.forEach(lantana => {
            if (Math.floor(lantana.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_lantana = lantana
            }
        })
        this.typed_lantana = match_lantana

        return new Array<GkillError>()
    }

    async load_typed_idf_kyou(): Promise<Array<GkillError>> {
        const req = new GetIDFKyouRequest()
        req.abort_controller = this.abort_controller
        req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_idf_kyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.idf_kyou_histories || res.idf_kyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_idf_kyou
            error.error_message = "IDFKyouが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.idf_kyou_histories.length; i++) {
            const idf_kyou = new IDFKyou()
            for (const key in res.idf_kyou_histories[i]) {
                (idf_kyou as any)[key] = (res.idf_kyou_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (idf_kyou as any)[key]) {
                    (idf_kyou as any)[key] = new Date((idf_kyou as any)[key])
                }
            }
            res.idf_kyou_histories[i] = idf_kyou
        }

        let match_idf_kyou: IDFKyou | null = null
        res.idf_kyou_histories.forEach(idf_kyou => {
            if (Math.floor(idf_kyou.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_idf_kyou = idf_kyou
            }
        })
        this.typed_idf_kyou = match_idf_kyou

        return new Array<GkillError>()
    }

    async load_typed_git_commit_log(): Promise<Array<GkillError>> {
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
            error.error_message = "GitCommitLogが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.git_commit_log_histories.length; i++) {
            const git_commit_log = new GitCommitLog()
            for (const key in res.git_commit_log_histories[i]) {
                (git_commit_log as any)[key] = (res.git_commit_log_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (git_commit_log as any)[key]) {
                    (git_commit_log as any)[key] = new Date((git_commit_log as any)[key])
                }
            }
            res.git_commit_log_histories[i] = git_commit_log
        }

        this.typed_git_commit_log = res.git_commit_log_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_rekyou(): Promise<Array<GkillError>> {
        const req = new GetReKyouRequest()
        req.abort_controller = this.abort_controller
        req.rep_name = this.rep_name

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_rekyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.rekyou_histories || res.rekyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.not_found_rekyou
            error.error_message = "ReKyouが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.rekyou_histories.length; i++) {
            const rekyou = new ReKyou()
            for (const key in res.rekyou_histories[i]) {
                (rekyou as any)[key] = (res.rekyou_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (rekyou as any)[key]) {
                    (rekyou as any)[key] = new Date((rekyou as any)[key])
                }
            }
            res.rekyou_histories[i] = rekyou
        }

        let match_rekyou: ReKyou | null = null
        res.rekyou_histories.forEach(rekyou => {
            if (Math.floor(rekyou.update_time.getTime() / 1000) === Math.floor(this.update_time.getTime() / 1000)) {
                match_rekyou = rekyou
            }
        })
        this.typed_rekyou = match_rekyou

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
        return new Array<GkillError>()
    }

    async reload(content_only: boolean, is_updated_info: boolean): Promise<Array<GkillError>> {
        const req = new GetKyouRequest()
        req.abort_controller = this.abort_controller
        if (!is_updated_info) {
            req.rep_name = this.rep_name
        }

        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_kyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        const latest_kyou = res.kyou_histories[0]
        this.is_deleted = latest_kyou.is_deleted
        this.id = latest_kyou.id
        this.rep_name = latest_kyou.rep_name
        if (!content_only) {
            this.related_time = latest_kyou.related_time
        }
        this.data_type = latest_kyou.data_type
        this.create_time = latest_kyou.create_time
        this.create_app = latest_kyou.create_app
        this.create_device = latest_kyou.create_device
        this.create_user = latest_kyou.create_user
        this.update_time = latest_kyou.update_time
        this.update_app = latest_kyou.update_app
        this.update_device = latest_kyou.update_device
        this.update_user = latest_kyou.update_user
        this.image_source = latest_kyou.image_source
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
        return cloned_kyou
    }

    generate_info_identifer(): InfoIdentifier {
        const info_identifer = new InfoIdentifier()
        info_identifer.id = this.id
        info_identifer.create_time = this.create_time
        info_identifer.update_time = this.update_time
        return info_identifer
    }

    constructor() {
        super()
        this.is_deleted = false
        this.image_source = ""
        this.attached_histories = new Array<Kyou>()
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
    }

}


