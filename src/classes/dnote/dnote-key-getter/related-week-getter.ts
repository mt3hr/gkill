import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class RelatedWeekGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedWeekGetter {
        return new RelatedWeekGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [this.getISOWeek(loaded_kyou.related_time).toString()]
    }

    to_json() {
        return {
            type: "RelatedWeekGetter",
        }
    }

    getISOWeek(date: Date): number {
        const tempDate = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()))
        const day = tempDate.getUTCDay() || 7
        tempDate.setUTCDate(tempDate.getUTCDate() + 4 - day)
        const yearStart = new Date(Date.UTC(tempDate.getUTCFullYear(), 0, 1))
        const weekNo = Math.ceil(((tempDate.getTime() - yearStart.getTime()) / 86400000 + 1) / 7)
        return weekNo
    }

}