import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteAggregateTarget from "../dnote-aggregate-target";
import AverageInfo from "./average-info";
import { i18n } from "@/i18n";

export default class AggregateAverageGitCommitLogCodeCount implements DnoteAggregateTarget {
    from_json(_json: any): DnoteAggregateTarget {
        return new AggregateAverageGitCommitLogCodeCount()
    }
    async append_aggregate_element_value(typed_average_info_git_commit_log_amount: any | null, kyou: Kyou, _find_kyou_query: FindKyouQuery): Promise<any> {
        const cloned_typed_average_info_git_commit_log_amount = typed_average_info_git_commit_log_amount === null ? new AverageInfo() : (typed_average_info_git_commit_log_amount as AverageInfo).clone()
        cloned_typed_average_info_git_commit_log_amount.total_value = cloned_typed_average_info_git_commit_log_amount.total_value === null ? 0 : cloned_typed_average_info_git_commit_log_amount.total_value as number

        let code_count = 0
        if (kyou.typed_git_commit_log) {
            code_count += kyou.typed_git_commit_log.addition + kyou.typed_git_commit_log.deletion

            cloned_typed_average_info_git_commit_log_amount.total_value += code_count
            cloned_typed_average_info_git_commit_log_amount.total_count++
        }
        return cloned_typed_average_info_git_commit_log_amount
    }
    async result_to_string(typed_average_info_git_commit_log_amount: any | null): Promise<string> {
        const cloned_typed_average_info_git_commit_log_amount = typed_average_info_git_commit_log_amount === null ? new AverageInfo() : (typed_average_info_git_commit_log_amount as AverageInfo).clone()
        cloned_typed_average_info_git_commit_log_amount.total_value = cloned_typed_average_info_git_commit_log_amount.total_value === null ? 0 : cloned_typed_average_info_git_commit_log_amount.total_value as number
        return (cloned_typed_average_info_git_commit_log_amount.total_count === 0 ? 0 : (cloned_typed_average_info_git_commit_log_amount.total_value / cloned_typed_average_info_git_commit_log_amount.total_count)) + i18n.global.t("YEN_TITLE")
    }

    to_json(): any {
        return {
            type: "AggregateAverageGitCommitLogCodeCount",
        }
    }
}