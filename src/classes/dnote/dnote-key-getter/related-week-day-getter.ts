import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class RelatedWeekDayGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedWeekDayGetter {
        return new RelatedWeekDayGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [((loaded_kyou.related_time.getDay() + 6) % 7).toString()]
    }

    to_json() {
        return {
            type: "RelatedWeekDayGetter",
        }
    }
}