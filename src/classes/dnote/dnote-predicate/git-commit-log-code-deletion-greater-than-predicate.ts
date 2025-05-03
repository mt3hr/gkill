import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class GitCommitLogCodeDeletionGreaterThanPredicate implements DnotePredicate {
    private git_commit_log_code_count: number
    constructor(git_commit_log_code_count: number) {
        this.git_commit_log_code_count = git_commit_log_code_count
    }
    static from_json(json: any): DnotePredicate {
        const git_commit_log_code_count = json.git_commit_log_code_count as number
        return new GitCommitLogCodeDeletionGreaterThanPredicate(git_commit_log_code_count)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        if (loaded_kyou.typed_git_commit_log) {
            const git_commit_log_code_count = loaded_kyou.typed_git_commit_log.deletion
            if (git_commit_log_code_count) {
                if (git_commit_log_code_count <= this.git_commit_log_code_count) {
                    return true
                }
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "GitCommitLogCodeDeletionGreaterThanPredicate",
            git_commit_log_code_count: this.git_commit_log_code_count,
        }
    }
}