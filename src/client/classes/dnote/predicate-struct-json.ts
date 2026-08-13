import type Predicate from "./predicate"
import type PredicateGroupType from "./predicate-group-type"

/**
 * 条件エディタ（edit-dnote-predicate-group.vue）が扱う素の構造と、
 * DnotePredicate のシリアライズ形式（register-dictionary が読む JSON）との相互変換。
 *
 * エディタ側はクラスを持たず `{ logic, predicates }`（グループ）と `{ type, value }`（葉）の
 * 入れ子だけを触るので、保存時・読込時にこの2関数で往復させる。
 * トレンドグラフと相関グラフの両方が同じ往復をするため、ここに1つだけ置く。
 */

/**
 * グループ（入れ子を持つ側）かどうかを判定する。
 * `logic` の有無だけでは、条件が壊れた保存データでグループとして扱ってしまうので
 * `predicates` が配列であることまで確かめる。
 */
export function is_predicate_group(predicate: Predicate | PredicateGroupType): predicate is PredicateGroupType {
    return 'logic' in predicate && Array.isArray(predicate.predicates)
}

/** エディタの構造を、DnotePredicate を組み立てられる JSON へ落とす。 */
export function predicate_struct_to_json(group: PredicateGroupType | Predicate): Record<string, unknown> {
    if (is_predicate_group(group)) {
        return {
            logic: group.logic,
            predicates: group.predicates.map(predicate => predicate_struct_to_json(predicate)),
        }
    }
    return { type: group.type, value: group.value }
}

/** DnotePredicate の JSON を、エディタが触れる素の構造へ戻す。 */
export function predicate_struct_from_json(json: Record<string, unknown>): PredicateGroupType | Predicate {
    if (json.logic && Array.isArray(json.predicates)) {
        return {
            logic: json.logic as PredicateGroupType['logic'],
            predicates: json.predicates.map(predicate => predicate_struct_from_json(predicate as Record<string, unknown>)),
        }
    }
    return { type: json.type as string, value: json.value }
}
