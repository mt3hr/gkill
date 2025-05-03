import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import AggregateTargetDictionary from "../serialize/dnote-aggregate-target-dictionary";

export default class AggregateSumGitCommitLogDeletionCodeCount implements DnoteAggregateTarget {
    static from_json(_json: any): DnoteAggregateTarget {
        return new AggregateSumGitCommitLogDeletionCodeCount()
    }
    async append_aggregate_element_value(git_commit_log_code_count: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const typed_git_commit_log_code_count = git_commit_log_code_count === null ? 0 : (git_commit_log_code_count as number)
        let code_count = 0
        if (kyou.typed_git_commit_log) {
            code_count = kyou.typed_git_commit_log.deletion
        }
        return typed_git_commit_log_code_count + code_count
    }
    async result_to_string(git_commit_log_code_count: any | null): Promise<string> {
        return (git_commit_log_code_count === null ? 0 : (git_commit_log_code_count as number)).toString()
    }
    to_json(): any {
        return {
            type: "AggregateSumGitCommitLogDeletionCodeCount",
        }
    }
}