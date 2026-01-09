import type { Kyou } from "../datas/kyou";

export default interface DnotePredicate {
    is_match(loaded_kyou: Kyou, target_kyou: Kyou | null): Promise<boolean>
    predicate_struct_to_json(): any
}
