'use strict'

import { InfoBase } from './info-base'
import { GkillError } from '../api/gkill-error'
import type { GitCommitLog } from './git-commit-log'
import type { IDFKyou } from './idf-kyou'
import type { Kmemo } from './kmemo'
import type { Lantana } from './lantana'
import type { Mi } from './mi'
import type { Nlog } from './nlog'
import type { ReKyou } from './re-kyou'
import type { TimeIs } from './time-is'
import type { URLog } from './ur-log'
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

    async load_attached_datas(): Promise<Array<GkillError>> {
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

    async clear_attached_histories(): Promise<Array<GkillError>> {
        this.attached_histories = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        this.attached_tags = []
        this.attached_texts = []
        this.attached_timeis_kyou = []
        return new Array<GkillError>()
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

        if (!res.rekyous|| res.rekyous.length < 1) {
            const error = new GkillError()
            error.error_code = "//TODO"
            error.error_message = "ReKyouが見つかりませんでした"
            return [error]
        }

        this.typed_rekyou= res.rekyous[0]

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


