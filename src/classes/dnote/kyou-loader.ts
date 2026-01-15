import type DnotePredicate from "./dnote-predicate";
import type { Kyou } from "../datas/kyou";

export default async function load_kyous(abort_controller: AbortController, kyous: Array<Kyou>, get_latest_data: boolean, clone: boolean, predicate?: DnotePredicate, target_kyou?: Kyou | null, limit?: number): Promise<Array<Kyou>> {
    let match_count = 0
    const cloned_kyous = new Array<Kyou>()
    for (let i = 0; i < kyous.length; i++) {
        let kyou: Kyou = kyous[i]
        const waitPromises = []
        if (clone) {
            kyou = kyous[i].clone()
            kyou.abort_controller = abort_controller
        }
        if (get_latest_data) {
            await kyou.reload(false, true)
        }
        if (clone || get_latest_data) {
            waitPromises.push(kyou.load_typed_datas())
            waitPromises.push(kyou.load_attached_tags())
            waitPromises.push(kyou.load_attached_texts())
        }
        await Promise.all(waitPromises)
        cloned_kyous.push(kyou)
        if (predicate && target_kyou && limit && (await predicate.is_match(kyou, target_kyou))) {
            match_count++
            if (match_count >= limit) {
                break
            }
        }
    }
    return cloned_kyous
}