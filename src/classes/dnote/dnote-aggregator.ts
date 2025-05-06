import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";
import type DnoteAgregateTarget from "./dnote-agregate-target";
import type DnotePredicate from "./dnote-predicate";
import load_kyous from "./kyou-loader";

export class DnoteAgregator {
    private dnote_predicate: DnotePredicate
    private dnote_agregate_target: DnoteAgregateTarget

    constructor(dnote_predicate: DnotePredicate, agregate_target: DnoteAgregateTarget) {
        this.dnote_predicate = dnote_predicate
        this.dnote_agregate_target = agregate_target
    }

    public async agregate(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<{ result_string: string, match_kyous: Array<Kyou> }> {
        // 渡されたデータの全項目を取得
        const cloned_kyous = await load_kyous(abort_controller, kyous, !kyou_is_loaded)

        // predicateにマッチしたKyouを抽出
        const match_kyous = new Array<Kyou>()
        for (let i = 0; i < cloned_kyous.length; i++) {
            if (await this.dnote_predicate.is_match(cloned_kyous[i])) {
                match_kyous.push(cloned_kyous[i])
            }
        }

        // 抽出されたKyouを集計
        let agregated_value: any | null = null
        for (let i = 0; i < match_kyous.length; i++) {
            const kyou = match_kyous[i]
            agregated_value = await this.dnote_agregate_target.append_agregate_element_value(agregated_value, kyou, find_kyou_query)
        }

        const cloned_match_kyous = new Array<Kyou>()
        for (let i = 0; i < match_kyous.length; i++) {
            const kyou = match_kyous[i]
            cloned_match_kyous.push(kyou.clone())
        }

        // 集計結果を返却
        const result_string = await this.dnote_agregate_target.result_to_string(agregated_value)
        return { result_string: result_string, match_kyous: cloned_match_kyous }
    }
}
