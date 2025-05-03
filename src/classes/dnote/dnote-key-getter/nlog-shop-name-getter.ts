import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class NlogShopNameGetter implements DnoteKeyGetter {

    get_keys(loaded_kyou: Kyou): Array<string> {
        if (loaded_kyou.data_type.startsWith("nlog") && loaded_kyou.typed_nlog) {
            return [loaded_kyou.typed_nlog.shop]
        }
        return []
    }

    to_json() {
        return {
            type: "NlogShopNameGetter",
        }
    }
}