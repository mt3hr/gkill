import type { FindKyouQuery } from "../api/find_query/find-kyou-query";
import type { Kyou } from "../datas/kyou";

export default interface DnoteAgregateTarget {
    append_agregate_element_value(agregated_value: any | null, kyou: Kyou, find_kyou_query: FindKyouQuery): Promise<any>
    result_to_string(agregated_value: any | null): Promise<string>
    to_json(): any
}