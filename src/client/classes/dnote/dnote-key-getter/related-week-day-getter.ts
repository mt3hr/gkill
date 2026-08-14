import type { Kyou } from "@/classes/datas/kyou";
import { format_day_of_week } from "@/classes/format-date-time";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class RelatedWeekDayGetter implements DnoteKeyGetter {

    static from_json(_json: Record<string, unknown>): RelatedWeekDayGetter {
        return new RelatedWeekDayGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        // 0〜6の数字ではなくロケールの曜日名を返す。
        // 見出しはaggregated-list-item.vueが無加工で描くので、ここが表示文字列そのものになる
        return [format_day_of_week(loaded_kyou.related_time)]
    }

    to_json() {
        return {
            type: "RelatedWeekDayGetter",
        }
    }
}
