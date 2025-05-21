import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query";
import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKyouFilter from "../dnote-kyou-filter";

export default class FilterTopKyous implements DnoteKyouFilter {
    private limit_count: number
    constructor(limit_count: number) {
        this.limit_count = limit_count
    }

    static from_json(json: any): DnoteKyouFilter {
        const limit_count = json.value as number
        return new FilterTopKyous(limit_count)
    }

    async filter_kyous(kyous: Array<Kyou>, _find_kyou_query: FindKyouQuery): Promise<Array<Kyou>> {
        const result_kyous = new Array<Kyou>()
        for (let i = 0; i < Math.min(this.limit_count, kyous.length); i++) {
            const kyou = kyous[i]
            result_kyous.push(kyou)
        }
        return result_kyous
    }

    to_json() {
        return {
            type: "FilterTopKyous",
            value: this.limit_count,
        }
    }
}