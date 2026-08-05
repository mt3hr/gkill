import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class RelatedTimeWeekPredicate implements DnotePredicate {
    private week: number
    constructor(week: number) {
        this.week = week
    }
    static from_json(json: Record<string, unknown>): DnotePredicate {
        const week = json.value as number
        return new RelatedTimeWeekPredicate(week)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const week = this.get_iso_week(loaded_kyou.related_time)
        if (week === this.week) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): Record<string, unknown> {
        return {
            type: "RelatedTimeWeekPredicate",
            value: this.week,
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