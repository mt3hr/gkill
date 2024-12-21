'use strict'

import { GitCommitLog } from '@/classes/datas/git-commit-log'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetGitCommitLogResponse extends GkillAPIResponse {

    git_commit_log_histories: Array<GitCommitLog>

    constructor() {
        super()
        this.git_commit_log_histories = new Array<GitCommitLog>()
    }

}


