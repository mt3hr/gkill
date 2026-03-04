import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class RelatedDateGetter implements DnoteKeyGetter {

    static from_json(_json: any): RelatedDateGetter {
        return new RelatedDateGetter()
    }

    get_keys(loaded_kyou: Kyou): Array<string> {
        return [`${loaded_kyou.related_time.getFullYear()}/${(loaded_kyou.related_time.getMonth() + 1).toString().padStart(2, '0')}/${loaded_kyou.related_time.getDate().toString().padStart(2, '0')}`
        ]
    }

    to_json() {
        return {
            type: "RelatedDateGetter",
        }
    }
}