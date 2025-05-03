import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class TitleGetter implements DnoteKeyGetter {

    get_keys(loaded_kyou: Kyou): Array<string> {
        if (loaded_kyou.data_type.startsWith("kmemo") && loaded_kyou.typed_kmemo) {
            return [loaded_kyou.typed_kmemo.content]
        } else if (loaded_kyou.data_type.startsWith("timeis") && loaded_kyou.typed_timeis) {
            return [loaded_kyou.typed_timeis.title]
        } else if (loaded_kyou.data_type.startsWith("mi") && loaded_kyou.typed_mi) {
            return [loaded_kyou.typed_mi.title]
        } else if (loaded_kyou.data_type.startsWith("git") && loaded_kyou.typed_git_commit_log) {
            return [loaded_kyou.typed_git_commit_log.commit_message]
        }
        return []
    }

    to_json() {
        return {
            type: "TitleGetter",
        }
    }
}