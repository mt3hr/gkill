import type { Kyou } from "../datas/kyou"

export default interface DnoteKeyGetter {
    get_keys(loaded_kyou: Kyou): Array<string>
    to_json(): any
}