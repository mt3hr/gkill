import type { Kyou } from "@/classes/datas/kyou";
import type DnotePredicate from "../dnote-predicate";
import PredicateDictonary from "../serialize/dnote-predicate-dictionary";

export default class DataTypePrefixPredicate implements DnotePredicate {
    private data_type_prefix: string
    constructor(data_type_prefix: string) {
        this.data_type_prefix = data_type_prefix
    }
    static from_json(json: any): DnotePredicate {
        const data_type_prefix = json.data_type_prefix as string
        return new DataTypePrefixPredicate(data_type_prefix)
    }
    async is_match(loaded_kyou: Kyou): Promise<boolean> {
        const data_type = loaded_kyou.data_type
        if (data_type.startsWith(this.data_type_prefix)) {
            return true
        }
        return false
    }
    predicate_struct_to_json(): any {
        return {
            type: "DataTypePrefixPredicate",
            data_type_prefix: this.data_type_prefix,
        }
    }
}