import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";

export default interface DnoteKyouFilter {
    filter_kyous(kyous: Array<Kyou>, find_kyou_query: FindKyouQuery): Promise<Array<Kyou>>
    to_json(): any
}