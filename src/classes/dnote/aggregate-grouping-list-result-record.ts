import type { Kyou } from "../datas/kyou"

export default class AggregatedItem {
    title: string = ""
    value: any = ""
    match_kyous: Array<Kyou> = new Array<Kyou>()
}