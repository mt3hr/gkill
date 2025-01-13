'use strict'

import { GkillAPI } from "../api/gkill-api"
import type { GkillError } from "../api/gkill-error"
import { GetGitCommitLogRequest } from "../api/req_res/get-git-commit-log-request"
import { InfoBase } from "./info-base"

export class GitCommitLog extends InfoBase {

    commit_message: string

    addition: number

    deletion: number

    attached_histories: Array<GitCommitLog>

    async load_attached_histories(): Promise<Array<GkillError>> {
        const req = new GetGitCommitLogRequest()
        req.abort_controller = this.abort_controller
        req.session_id = GkillAPI.get_gkill_api().get_session_id()
        req.id = this.id
        const res = await GkillAPI.get_gkill_api().get_git_commit_log(req)
        if (res.errors && res.errors.length !== 0) {
            return res.errors
        }
        this.attached_histories = res.git_commit_log_histories
        return new Array<GkillError>()
    }

    async load_attached_datas(): Promise<Array<GkillError>> {
        try {
            return this.load_attached_histories()
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
            return []
        }
    }

    async clear_attached_histories(): Promise<Array<GkillError>> {
        this.attached_histories = []
        return new Array<GkillError>()
    }

    async clear_attached_datas(): Promise<Array<GkillError>> {
        let errors = new Array<GkillError>()
        errors = errors.concat(await this.clear_attached_tags())
        errors = errors.concat(await this.clear_attached_texts())
        errors = errors.concat(await this.clear_attached_notifications())
        errors = errors.concat(await this.clear_attached_timeis())
        errors = errors.concat(await this.clear_attached_histories())
        return errors
    }



    clone(): GitCommitLog {
        const git_commit_log = new GitCommitLog()
        git_commit_log.is_deleted = this.is_deleted
        git_commit_log.id = this.id
        git_commit_log.rep_name = this.rep_name
        git_commit_log.related_time = this.related_time
        git_commit_log.data_type = this.data_type
        git_commit_log.create_time = this.create_time
        git_commit_log.create_app = this.create_app
        git_commit_log.create_device = this.create_device
        git_commit_log.create_user = this.create_user
        git_commit_log.update_time = this.update_time
        git_commit_log.update_app = this.update_app
        git_commit_log.update_user = this.update_user
        git_commit_log.update_device = this.update_device
        git_commit_log.commit_message = this.commit_message
        git_commit_log.addition = this.addition
        git_commit_log.deletion = this.deletion
        return git_commit_log
    }

    constructor() {
        super()
        this.commit_message = ""
        this.addition = 0
        this.deletion = 0
        this.attached_histories = new Array<GitCommitLog>()
    }

}


