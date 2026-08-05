import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class RelatedWeekGetter implements DnoteKeyGetter {

    static from_json(_json: Record<string, unknown>): RelatedWeekGetter {
        return new RelatedWeekGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [this.get_iso_week(loaded_kyou.related_time).toString()]
    }

    to_json() {
        return {
            type: "RelatedWeekGetter",
        }
    }

    get_iso_week(date: Date): number {
        const temp_date = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
        const day = temp_date.getUTCDay() || 7
        temp_date.setUTCDate(temp_date.getUTCDate() + 4 - day)
        const year_start = new Date(Date.UTC(temp_date.getUTCFullYear(), 0, 1))
        const week_no = Math.ceil(((temp_date.getTime() - year_start.getTime()) / 86400000 + 1) / 7)
        return week_no
    }

}