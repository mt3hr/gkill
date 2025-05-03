import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class NlogShopContainsPredicate implements DnotePredicate {
    private nlog_shop_contains_target: string
    constructor(nlog_shop_contains_target: string) {
        this.nlog_shop_contains_target = nlog_shop_contains_target
    }
    static from_json(json: any): DnotePredicate {
        const nlog_shop_contains_target = json.nlog_shop_contains_target as string
        return new NlogShopContainsPredicate(nlog_shop_contains_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const nlog_shop = loaded_kyou.typed_nlog?.shop
        if (nlog_shop) {
            if (nlog_shop?.includes(this.nlog_shop_contains_target)) {
                return true
            }
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "NlogShopContainsPredicate",
            nlog_shop_contains_target: this.nlog_shop_contains_target,
        }
    }
}