'use strict'

import type { GitCommitLog } from "@/classes/datas/git-commit-log"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface GitCommitLogViewProps extends KyouViewPropsBase {
    git_commit_log: GitCommitLog
    width: number | string
    height: number | string
}
