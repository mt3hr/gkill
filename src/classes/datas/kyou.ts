'use strict'

import { InfoBase } from './info-base'
import { GkillError } from '../api/gkill-error'
import { GitCommitLog } from './git-commit-log'
import type { IDFKyou } from './idf-kyou'
import { Kmemo } from './kmemo'
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
import { GetURLogRequest } from '../api/req_res/get-ur-log-request'
import { GetNlogRequest } from '../api/req_res/get-nlog-request'
import { GetTimeisRequest } from '../api/req_res/get-timeis-request'
import { GetMiRequest } from '../api/req_res/get-mi-request'
import { GetLantanaRequest } from '../api/req_res/get-lantana-request'
import { GetGitCommitLogRequest } from '../api/req_res/get-git-commit-log-request'
import { GetReKyouRequest } from '../api/req_res/get-re-kyou-request'
import moment from 'moment'

export class Kyou extends InfoBase {
    is_deleted: boolean
    image_source: string
    attached_histories: Array<Kyou>
    typed_kmemo: Kmemo | null
    typed_urlog: URLog | null
    typed_nlog: Nlog | null
    typed_timeis: TimeIs | null
    typed_mi: Mi | null
    typed_lantana: Lantana | null
    typed_idf_kyou: IDFKyou | null
    typed_git_commit_log: GitCommitLog | null
    typed_rekyou: ReKyou | null

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetKyouRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_kyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        this.attached_histories = res.kyou_histories
        return new Array<GkillError>()
    }

    async load_all(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        errors = errors.concat(await this.load_typed_datas())
        errors = errors.concat(await this.load_attached_datas())
        errors = errors.concat(await this.load_attached_histories())
        return errors
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
        let errors = new Array<GkillError>()
        errors = errors.concat(await this.load_attached_tags())
        errors = errors.concat(await this.load_attached_texts())
        errors = errors.concat(await this.load_attached_timeis())
        errors = errors.concat(await this.load_attached_histories())
        return errors
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
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_kmemo(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.kmemo_histories || res.kmemo_histories.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "Kmemoが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.kmemo_histories.length; i++) {
            const kmemo = new Kmemo()
            for (let key in res.kmemo_histories[i]) {
                (kmemo as any)[key] = (res.kmemo_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (kmemo as any)[key]) {
                    (kmemo as any)[key] = moment((kmemo as any)[key]).toDate()
                }
            }
            res.kmemo_histories[i] = kmemo
        }

        this.typed_kmemo = res.kmemo_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_urlog(): Promise<Array<GkillError>> {
        const req = new GetURLogRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_urlog(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.urlog_histories || res.urlog_histories.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "URLogが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.urlog_histories.length; i++) {
            const urlog = new URLog()
            for (let key in res.urlog_histories[i]) {
                (urlog as any)[key] = (res.urlog_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (urlog as any)[key]) {
                    (urlog as any)[key] = moment((urlog as any)[key]).toDate()
                }
            }
            res.urlog_histories[i] = urlog
        }

        this.typed_urlog = res.urlog_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_nlog(): Promise<Array<GkillError>> {
        const req = new GetNlogRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_nlog(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.nlog_histories || res.nlog_histories.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "Nlogが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.nlog_histories.length; i++) {
            const nlog = new Nlog()
            for (let key in res.nlog_histories[i]) {
                (nlog as any)[key] = (res.nlog_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (nlog as any)[key]) {
                    (nlog as any)[key] = moment((nlog as any)[key]).toDate()
                }
            }
            res.nlog_histories[i] = nlog
        }


        this.typed_nlog = res.nlog_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_timeis(): Promise<Array<GkillError>> {
        const req = new GetTimeisRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_timeis(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.timeis_histories || res.timeis_histories.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "TimeIsが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.timeis_histories.length; i++) {
            const timeis = new TimeIs()
            for (let key in res.timeis_histories[i]) {
                (timeis as any)[key] = (res.timeis_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (timeis as any)[key]) {
                    (timeis as any)[key] = moment((timeis as any)[key]).toDate()
                }
            }
            res.timeis_histories[i] = timeis
        }


        this.typed_timeis = res.timeis_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_mi(): Promise<Array<GkillError>> {
        const req = new GetMiRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_mi(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.mi_histories || res.mi_histories.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "Miが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.mi_histories.length; i++) {
            const mi = new Mi()
            for (let key in res.mi_histories[i]) {
                (mi as any)[key] = (res.mi_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (mi as any)[key]) {
                    (mi as any)[key] = moment((mi as any)[key]).toDate()
                }
            }
            res.mi_histories[i] = mi
        }


        this.typed_mi = res.mi_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_lantana(): Promise<Array<GkillError>> {
        const req = new GetLantanaRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_lantana(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.lantana_histories || res.lantana_histories.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "Lantanaが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.lantana_histories.length; i++) {
            const lantana = new Lantana()
            for (let key in res.lantana_histories[i]) {
                (lantana as any)[key] = (res.lantana_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (lantana as any)[key]) {
                    (lantana as any)[key] = moment((lantana as any)[key]).toDate()
                }
            }
            res.lantana_histories[i] = lantana
        }


        this.typed_lantana = res.lantana_histories[0]

        return new Array<GkillError>()
    }

    async load_typed_idf_kyou(): Promise<Array<GkillError>> {
        throw new Error('Not implemented')
    }

    async load_typed_git_commit_log(): Promise<Array<GkillError>> {
        const req = new GetGitCommitLogRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_git_commit_log(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.git_commit_logs || res.git_commit_logs.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "GitCommitLogが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.git_commit_logs.length; i++) {
            const git_commit_log = new GitCommitLog()
            for (let key in res.git_commit_logs[i]) {
                (git_commit_log as any)[key] = (res.git_commit_logs[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (git_commit_log as any)[key]) {
                    (git_commit_log as any)[key] = moment((git_commit_log as any)[key]).toDate()
                }
            }
            res.git_commit_logs[i] = git_commit_log
        }


        this.typed_git_commit_log = res.git_commit_logs[0]

        return new Array<GkillError>()
    }

    async load_typed_rekyou(): Promise<Array<GkillError>> {
        const req = new GetReKyouRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_rekyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }

        if (!res.rekyou_histories || res.rekyou_histories.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "ReKyouが見つかりませんでした"
            return [error]
        }

        // 取得したデータリストの型変換（そのままキャストするとメソッドが生えないため）
        for (let i = 0; i < res.rekyou_histories.length; i++) {
            const rekyou = new ReKyou()
            for (let key in res.rekyou_histories[i]) {
                (rekyou as any)[key] = (res.rekyou_histories[i] as any)[key]

                // 時刻はDate型に変換
                if (key.endsWith("time") && (rekyou as any)[key]) {
                    (rekyou as any)[key] = moment((rekyou as any)[key]).toDate()
                }
            }
            res.rekyou_histories[i] = rekyou
        }

        this.typed_rekyou = res.rekyou_histories[0]

        return new Array<GkillError>()
    }

    async clear_typed_datas(): Promise<Array<GkillError>> {
        this.typed_kmemo = null
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

    async reload(): Promise<Array<GkillError>> {
        const req = new GetKyouRequest()
        req.session_id = GkillAPI.get_instance().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_instance().get_kyou(req)
        if (res.errors && res.errors.length != 0) {
            return res.errors
        }
        const latest_kyou = res.kyou_histories[0]
        this.is_deleted = latest_kyou.is_deleted
        this.id = latest_kyou.id
        this.rep_name = latest_kyou.rep_name
        this.related_time = latest_kyou.related_time
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


