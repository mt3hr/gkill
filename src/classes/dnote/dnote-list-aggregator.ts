import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";
import type AggregateGroupingListResultRecord from "./aggregate-grouping-list-result-record";
import type DnoteAggregateTarget from "./dnote-aggregate-target";
import type DnoteKeyGetter from "./dnote-key-getter";
import type DnotePredicate from "./dnote-predicate";

export class DnoteListAggregator {
    private dnote_key_getter: DnoteKeyGetter
    private dnote_predicate: DnotePredicate
    private dnote_aggregate_target: DnoteAggregateTarget

    constructor(dnote_predicate: DnotePredicate, dnote_key_getter: DnoteKeyGetter, dnote_aggregate_target: DnoteAggregateTarget) {
        this.dnote_predicate = dnote_predicate
        this.dnote_key_getter = dnote_key_getter
        this.dnote_aggregate_target = dnote_aggregate_target
    }

    public async aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery): Promise<Array<AggregateGroupingListResultRecord>> {
        // 渡されたデータの全項目を取得
        const cloned_kyous = new Array<Kyou>()
        for (let i = 0; i < kyous.length; i++) {
            const kyou = kyous[i].clone()
            kyou.abort_controller = abort_controller
            await kyou.load_typed_datas()
            await kyou.load_attached_tags()
            cloned_kyous.push(kyou)
        }

        // predicateにマッチしたKyouを抽出
        const match_kyous = new Array<Kyou>()
        for (let i = 0; i < cloned_kyous.length; i++) {
            if (await this.dnote_predicate.is_match(cloned_kyous[i])) {
                match_kyous.push(cloned_kyous[i])
            }
        }

        const aggregated_result_list = new Array<AggregateGroupingListResultRecord>()
        for (let i = 0; i < match_kyous.length; i++) {
            const kyou = match_kyous[i]
            const keys = this.dnote_key_getter.get_keys(kyou)
            for (let j = 0; j < keys.length; j++) {
                const key = keys[j]
                // すでに同じキーの集計結果がある場合は、値を追加する
                const existing_result = aggregated_result_list.find(result => result.title === key)
                if (existing_result) {
                    existing_result.value = await this.dnote_aggregate_target.append_aggregate_element_value(existing_result.value, kyou, find_kyou_query)
                    existing_result.match_kyous.push(kyou.clone())
                } else {
                    // 新しいキーの場合は、新しい集計結果を作成する
                    const aggregated_value = await this.dnote_aggregate_target.append_aggregate_element_value(null, kyou, find_kyou_query)
                    aggregated_result_list.push({ title: key, value: aggregated_value, match_kyous: [kyou.clone()] })
                }
            }
        }
        aggregated_result_list.sort((a, b) => b.value - a.value)
        return aggregated_result_list
    }
}