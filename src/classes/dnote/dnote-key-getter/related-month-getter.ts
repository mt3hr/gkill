import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class RelatedMonthGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedMonthGetter {
        return new RelatedMonthGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [loaded_kyou.related_time.getFullYear() + "/" + String(loaded_kyou.related_time.getMonth() + 1).padStart(2, "0")]
    }

    to_json() {
        return {
            type: "RelatedMonthGetter",
        }
    }
}