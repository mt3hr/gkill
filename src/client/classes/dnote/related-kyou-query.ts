import { RelatedTimeMatchType } from "@/classes/dnote/related-time-match-type"
import type DnotePredicate from "./dnote-predicate"
import AndPredicate from "./dnote-predicate/and-predicate"
import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"

export default class RelatedKyouQuery {
    id: string = ""
    title: string = ""
    prefix: string = ""
    suffix: string = ""
    predicate: DnotePredicate = new AndPredicate([])
    related_time_match_type: RelatedTimeMatchType = RelatedTimeMatchType.NEAR_RELATED_TIME
    find_kyou_query: FindKyouQuery | null = null
    find_duration_hour: number = 1
}