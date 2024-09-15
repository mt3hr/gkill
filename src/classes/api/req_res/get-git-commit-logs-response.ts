'use strict';

import { GitCommitLog } from '@/classes/datas/git-commit-log';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetGitCommitLogsResponse extends GkillAPIResponse {


    git_commit_logs: Array<GitCommitLog>;

    constructor() {
        super()
        this.git_commit_logs = new Array<GitCommitLog>()
    }


}



