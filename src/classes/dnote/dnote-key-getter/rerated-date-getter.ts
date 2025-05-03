import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";
import moment from "moment";
import DnoteKeyGetterDictionary from "../serialize/dnote-key-getter-dictionary";

export default class RelatedDateGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedDateGetter {
        return new RelatedDateGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [moment(loaded_kyou.related_time).format("YYYY/MM/DD")]
    }

    to_json() {
        return {
            type: "RelatedDateGetter",
        }
    }
}