import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAgregateTarget from "../dnote-agregate-target";
import AverageInfo from "./average-info";

export default class AgregateAverageGitCommitLogCodeCount implements DnoteAgregateTarget {
    static from_json(_json: Record<string, unknown>): DnoteAgregateTarget {
        return new AgregateAverageGitCommitLogCodeCount()
    }
    async append_agregate_element_value(typed_average_info_git_commit_log_amount: unknown, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<unknown> {
        const cloned_typed_average_info_git_commit_log_amount = typed_average_info_git_commit_log_amount === null ? new AverageInfo() : (typed_average_info_git_commit_log_amount as AverageInfo).clone()
        cloned_typed_average_info_git_commit_log_amount.total_value = cloned_typed_average_info_git_commit_log_amount.total_value === null ? 0 : cloned_typed_average_info_git_commit_log_amount.total_value as number

        let code_count = 0
        if (kyou.typed_git_commit_log) {
            code_count += kyou.typed_git_commit_log.addition - kyou.typed_git_commit_log.deletion

            cloned_typed_average_info_git_commit_log_amount.total_value += code_count
            cloned_typed_average_info_git_commit_log_amount.total_count++
        }
        return cloned_typed_average_info_git_commit_log_amount
    }
    async result_to_string(typed_average_info_git_commit_log_amount: unknown): Promise<string> {
        const cloned_typed_average_info_git_commit_log_amount = typed_average_info_git_commit_log_amount === null ? new AverageInfo() : (typed_average_info_git_commit_log_amount as AverageInfo).clone()
        cloned_typed_average_info_git_commit_log_amount.total_value = cloned_typed_average_info_git_commit_log_amount.total_value === null ? 0 : cloned_typed_average_info_git_commit_log_amount.total_value as number
        return (cloned_typed_average_info_git_commit_log_amount.total_count === 0 ? 0 : (cloned_typed_average_info_git_commit_log_amount.total_value / cloned_typed_average_info_git_commit_log_amount.total_count)).toString()
    }

    to_json(): Record<string, unknown> {
        return {
            type: "AgregateAverageGitCommitLogCodeCount",
        }
    }
}