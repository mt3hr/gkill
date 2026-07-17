import type { Kyou } from "../../datas/kyou"

export default interface DnoteTrendPoint {
    bucket_key: string
    label: string
    value: number
    value_string: string
    match_kyous: Array<Kyou>
}
