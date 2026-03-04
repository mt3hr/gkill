import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../../dnote-predicate";

export default class EqualTitleTargetKyouPredicate implements DnotePredicate {
    private title: string
    constructor(title: string) {
        this.title = title
    }
    static from_json(json: any): DnotePredicate {
        const title = json.data_type_prefix as string
        return new EqualTitleTargetKyouPredicate(title)
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        if (!target_kyou) {
            return false
        }

        const get_title_func = (kyou: Kyou): string | null => {
            if (kyou.data_type.startsWith("kmemo")) {
                return kyou.typed_kmemo ? kyou.typed_kmemo.content : null
            }
            if (kyou.data_type.startsWith("kc")) {
                return kyou.typed_kc ? kyou.typed_kc.title : null
            }
            if (kyou.data_type.startsWith("urlog")) {
                return kyou.typed_urlog ? kyou.typed_urlog.url : null
            }
            if (kyou.data_type.startsWith("nlog")) {
                return kyou.typed_nlog ? kyou.typed_nlog.title : null
            }
            if (kyou.data_type.startsWith("timeis")) {
                return kyou.typed_timeis ? kyou.typed_timeis.title : null
            }
            if (kyou.data_type.startsWith("mi")) {
                return kyou.typed_mi ? kyou.typed_mi.title : null
            }
            if (kyou.data_type.startsWith("lantana")) {
                return null
            }
            if (kyou.data_type.startsWith("idf")) {
                return kyou.typed_idf_kyou ? kyou.typed_idf_kyou.file_name : null
            }
            if (kyou.data_type.startsWith("git")) {
                return kyou.typed_git_commit_log ? kyou.typed_git_commit_log.commit_message : null
            }
            if (kyou.data_type.startsWith("rekyou")) {
                return null
            }
            return null
        }

        if (get_title_func(loaded_kyou) === get_title_func(target_kyou)) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "EqualTitleTargetKyouPredicate",
            value: this.title,
        }
    }
}