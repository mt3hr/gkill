import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";

export default class TagGetter implements DnoteKeyGetter {

    get_keys(loaded_kyou: Kyou): Array<string> {
        const tags = new Array<string>()
        for (let i = 0; i < loaded_kyou.attached_tags.length; i++) {
            tags.push(loaded_kyou.attached_tags[i].tag)
        }
        return tags
    }

    to_json() {
        return {
            type: "TagGetter",
        }
    }
}