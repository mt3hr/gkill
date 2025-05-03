import type { Kyou } from "@/classes/datas/kyou";
import type DnoteKeyGetter from "../dnote-key-getter";
import DnoteKeyGetterDictionary from "../serialize/dnote-key-getter-dictionary";

export default class TagGetter implements DnoteKeyGetter {

    static from_json(_json: any): TagGetter {
        return new TagGetter()
    }

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