import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";
import moment from "moment";

export default class RelatedMonthGetter implements DnoteKeyGetter {

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [moment(loaded_kyou.related_time).format("YYYY/MM")]
    }

    to_json() {
        return {
            type: "RelatedMonthGetter",
        }
    }
}