import type DnotePredicate from "./dnote-predicate";
import type { Kyou } from "../datas/kyou";

/**
 * 1件ぶんの読み込み。clone / 最新版取り直し / 付随データ取得をまとめる。
 */
async function prepare_kyou(abort_controller: AbortController, source: Kyou, get_latest_data: boolean, clone: boolean): Promise<Kyou> {
    let kyou: Kyou = source
    if (clone) {
        kyou = source.clone()
        kyou.abort_controller = abort_controller
    }
    if (get_latest_data) {
        await kyou.reload(false, false)
    }
    if (clone || get_latest_data) {
        await Promise.all([
            kyou.load_typed_datas(),
            kyou.load_attached_tags(),
            kyou.load_attached_texts(),
        ])
    }
    return kyou
}

// 同時に走らせる件数。サーバのgoroutineプールを埋め尽くさない程度に抑える。
const LOAD_CONCURRENCY = 8

export default async function load_kyous(abort_controller: AbortController, kyous: Array<Kyou>, get_latest_data: boolean, clone: boolean, predicate?: DnotePredicate, target_kyou?: Kyou | null, limit?: number): Promise<Array<Kyou>> {
    // limit付き(ryuuの前後1件探し)は「先頭から順に見て、条件に合う件数がlimitに達したら止める」
    // という意味なので、先回りして読むと余計な取得が発生する。ここは逐次のまま。
    const use_limit = Boolean(predicate && target_kyou && limit)
    if (!use_limit) {
        // Dnote/Ryuuの一覧はこちら。1件ずつ待つと件数×RTTかかる
        // (1,000件 × RTT20ms で約20秒) ので、一定数ずつ並列で読む。
        const cloned_kyous = new Array<Kyou>()
        for (let start = 0; start < kyous.length; start += LOAD_CONCURRENCY) {
            const chunk = kyous.slice(start, start + LOAD_CONCURRENCY)
            const prepared = await Promise.all(chunk.map(source => prepare_kyou(abort_controller, source, get_latest_data, clone)))
            cloned_kyous.push(...prepared)
        }
        return cloned_kyous
    }

    let match_count = 0
    const cloned_kyous = new Array<Kyou>()
    for (let i = 0; i < kyous.length; i++) {
        const kyou = await prepare_kyou(abort_controller, kyous[i], get_latest_data, clone)
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
