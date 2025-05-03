import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";
import moment from "moment";
import DnoteKeyGetterDictionary from "../serialize/dnote-key-getter-dictionary";

export default class RelatedWeekDayGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedWeekDayGetter {
        return new RelatedWeekDayGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [moment(loaded_kyou.related_time).weekday().toString()]
    }

    to_json() {
        return {
            type: "RelatedWeekDayGetter",
        }
    }
}