import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";
import type AgregatedItem from "./aggregate-grouping-list-result-record";
import type DnoteAgregateTarget from "./dnote-agregate-target";
import type DnoteKeyGetter from "./dnote-key-getter";
import type DnotePredicate from "./dnote-predicate";
import load_kyous from "./kyou-loader";

export class DnoteListAggregator {
    private dnote_key_getter: DnoteKeyGetter
    private dnote_predicate: DnotePredicate
    private dnote_aggregate_target: DnoteAgregateTarget

    constructor(dnote_predicate: DnotePredicate, dnote_key_getter: DnoteKeyGetter, dnote_aggregate_target: DnoteAgregateTarget) {
        this.dnote_predicate = dnote_predicate
        this.dnote_key_getter = dnote_key_getter
        this.dnote_aggregate_target = dnote_aggregate_target
    }

    public async aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<Array<AgregatedItem>> {
        // 渡されたデータの全項目を取得
        const cloned_kyous = await load_kyous(abort_controller, kyous, !kyou_is_loaded)

        // predicateにマッチしたKyouを抽出
        const match_kyous = new Array<Kyou>()
        for (let i = 0; i < cloned_kyous.length; i++) {
            if (await this.dnote_predicate.is_match(cloned_kyous[i])) {
                match_kyous.push(cloned_kyous[i])
            }
        }

        const aggregated_result_list = new Array<AgregatedItem>()
        for (let i = 0; i < match_kyous.length; i++) {
            const kyou = match_kyous[i]
            const keys = this.dnote_key_getter.get_keys(kyou)
            for (let j = 0; j < keys.length; j++) {
                const key = keys[j]
                // すでに同じキーの集計結果がある場合は、値を追加する
                const existing_result = aggregated_result_list.find(result => result.title === key)
                if (existing_result) {
                    existing_result.value = await this.dnote_aggregate_target.append_agregate_element_value(existing_result.value, kyou, find_kyou_query)
                    existing_result.match_kyous.push(kyou.clone())
                } else {
                    // 新しいキーの場合は、新しい集計結果を作成する
                    const aggregated_value = await this.dnote_aggregate_target.append_agregate_element_value(null, kyou, find_kyou_query)
                    aggregated_result_list.push({ title: key, value: aggregated_value, match_kyous: [kyou.clone()] })
                }
            }
        }
        if (aggregated_result_list.length > 0 && typeof aggregated_result_list[0].value === "number") {
            aggregated_result_list.sort((a, b) => Math.abs(b.value) - Math.abs(a.value))
        } else {
            aggregated_result_list.sort((a, b) => b.value - a.value)
        }
        for (let i = 0; i < aggregated_result_list.length; i++) {
            aggregated_result_list[i].value = await this.dnote_aggregate_target.result_to_string(aggregated_result_list[i].value)
        }
        return aggregated_result_list
    }
}