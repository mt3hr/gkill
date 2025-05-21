import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class RelatedTimeBeforePredicate implements DnotePredicate {
    private related_time: Date
    constructor(related_time: Date) {
        this.related_time = related_time
    }
    static from_json(json: any): DnotePredicate {
        const related_time = new Date(json.value) as Date
        return new RelatedTimeBeforePredicate(related_time)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const related_time = loaded_kyou.related_time
        if (related_time.getTime() < this.related_time.getTime()) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "RelatedTimeBeforePredicate",
            value: this.related_time.getTime(),
        }
    }
}