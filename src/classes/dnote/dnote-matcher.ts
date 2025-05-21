import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";
import type DnotePredicate from "./dnote-predicate";
import load_kyous from "./kyou-loader";

export class DnoteMatcher {
    private dnote_predicate: DnotePredicate

    constructor(dnote_predicate: DnotePredicate) {
        this.dnote_predicate = dnote_predicate
    }

    public async match(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<Array<Kyou>> {
        // 渡されたデータの全項目を取得
        const cloned_kyous = await load_kyous(abort_controller, kyous, !kyou_is_loaded)

        // predicateにマッチしたKyouを抽出
        const match_kyous = new Array<Kyou>()
        for (let i = 0; i < cloned_kyous.length; i++) {
            if (await this.dnote_predicate.is_match(cloned_kyous[i])) {
                match_kyous.push(cloned_kyous[i])
            }
        }

        const cloned_match_kyous = new Array<Kyou>()
        for (let i = 0; i < match_kyous.length; i++) {
            const kyou = match_kyous[i]
            cloned_match_kyous.push(kyou.clone())
        }
        return cloned_match_kyous 
    }
}
