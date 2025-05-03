import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";
import moment from "moment";

export default class RelatedWeekDayGetter implements DnoteKeyGetter {

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [moment(loaded_kyou.related_time).weekday().toString()]
    }

    to_json() {
        return {
            type: "RelatedWeekDayGetter",
        }
    }
}