// ˅
'use strict';

import { FindGitCommitLogQuery } from '../find_query/find-git-commit-log-query';
import { GkillAPIRequest } from '../gkill-api-request';

// ˄

export class GetGitCommitLogsRequest extends GkillAPIRequest {
    // ˅
    
    // ˄

    query: FindGitCommitLogQuery;

    constructor() {
        // ˅
        super()
        this.query = new FindGitCommitLogQuery()
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
