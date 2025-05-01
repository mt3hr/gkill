import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";

export default class NlogShopEqualPredicate implements DnotePredicate {
    private nlog_shop_equal_target: string
    constructor(nlog_shop_equal_target: string) {
        this.nlog_shop_equal_target = nlog_shop_equal_target
    }
    static from_json(json: any): DnotePredicate {
        const nlog_shop_equal_target = json.nlog_shop_equal_target as string
        return new NlogShopEqualPredicate(nlog_shop_equal_target)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const nlog_shop = loaded_kyou.typed_nlog?.shop
        if (nlog_shop) {
            if (nlog_shop === this.nlog_shop_equal_target) {
                return true
            }
        }
        return false
    }
    to_json(): any {
        return {
            type: "NlogShopEqualPredicate",
            nlog_shop_equal_target: this.nlog_shop_equal_target,
        }
    }
}