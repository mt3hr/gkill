import type { Kyou } from "../datas/kyou"

export default class AgregatedItem {
    title: string = ""
    value: any = ""
    match_kyous: Array<Kyou> = new Array<Kyou>()
}