import type { Kyou } from "../datas/kyou"

export default interface AggregateGroupingListResultRecord {
    title: string
    value: any
    match_kyous: Array<Kyou>
}