import type { Kyou } from "../datas/kyou";

export default async function load_kyous(abort_controller: AbortController, kyous: Array<Kyou>, clone: boolean): Promise<Array<Kyou>> {
    const cloned_kyous = new Array<Kyou>()
    for (let i = 0; i < kyous.length; i++) {
        let kyou: Kyou = kyous[i]
        if (clone) {
            kyou = kyous[i].clone()
        }
        kyou.abort_controller = abort_controller
        await kyou.load_typed_datas()
        await kyou.load_attached_tags()
        cloned_kyous.push(kyou)
    }
    return cloned_kyous
}