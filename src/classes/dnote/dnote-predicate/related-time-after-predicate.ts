import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class RelatedTimeAfterPredicate implements DnotePredicate {
    private related_time: Date
    constructor(related_time: Date) {
        this.related_time = related_time
    }
    static from_json(json: any): DnotePredicate {
        const related_time = new Date(json.value) as Date
        return new RelatedTimeAfterPredicate(related_time)
    }
    async is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean> {
        const related_time = loaded_kyou.related_time
        if (related_time.getTime() > this.related_time.getTime()) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "RelatedTimeAfterPredicate",
            value: this.related_time.getTime(),
        }
    }
}