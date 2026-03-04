import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class RelatedTimeWeekPredicate implements DnotePredicate {
    private week: number
    constructor(week: number) {
        this.week = week
    }
    static from_json(json: any): DnotePredicate {
        const week = json.value as number
        return new RelatedTimeWeekPredicate(week)
    }
    async is_match(loaded_kyou: Kyou, _: Kyou | null): Promise<boolean> {
        const week = this.getISOWeek(loaded_kyou.related_time)
        if (week === this.week) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "RelatedTimeWeekPredicate",
            value: this.week,
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