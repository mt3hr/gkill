import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";
import moment from "moment";
import DnoteKeyGetterDictionary from "../serialize/dnote-key-getter-dictionary";

export default class RelatedMonthGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedMonthGetter {
        return new RelatedMonthGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [moment(loaded_kyou.related_time).format("YYYY/MM")]
    }

    to_json() {
        return {
            type: "RelatedMonthGetter",
        }
    }
}