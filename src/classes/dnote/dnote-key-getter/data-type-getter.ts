import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class DataTypeGetter implements DnoteKeyGetter {

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [loaded_kyou.data_type]
    }

    to_json() {
        return {
            type: "DataTypeGetter",
        }
    }
}