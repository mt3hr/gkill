import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import moment from "moment";

export default class RelatedTimeWeekPredicate implements DnotePredicate {
    private week: number
    constructor(week: number) {
        this.week = week
    }
    static from_json(json: any): DnotePredicate {
        const week = json.week as number
        return new RelatedTimeWeekPredicate(week)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const week = moment(loaded_kyou.related_time).week()
        if (week === this.week) {
            return true
        }
        return false
    }
    to_json(): any {
        return {
            type: "RelatedTimeWeekPredicate",
            week: this.week,
        }
    }
}