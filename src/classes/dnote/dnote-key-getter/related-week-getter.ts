import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";
import moment from "moment";
import DnoteKeyGetterDictionary from "../serialize/dnote-key-getter-dictionary";

export default class RelatedWeekGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedWeekGetter {
        return new RelatedWeekGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [moment(loaded_kyou.related_time).week().toString()]
    }

    to_json() {
        return {
            type: "RelatedWeekGetter",
        }
    }
}