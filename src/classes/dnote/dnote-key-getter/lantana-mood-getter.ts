import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class LantanaMoodGetter implements DnoteKeyGetter {

    static from_json(_json: any): LantanaMoodGetter {
        return new LantanaMoodGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        if (loaded_kyou.typed_lantana) {
            return [loaded_kyou.typed_lantana.mood.toString()]
        }
        return []
    }

    to_json() {
        return {
            type: "LantanaMoodGetter",
        }
    }
}