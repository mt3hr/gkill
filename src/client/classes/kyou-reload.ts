'use strict'

import { reactive, toRaw } from 'vue'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'

/**
 * Kyou 1件を最新化する唯一の手順。
 *
 * 以前は rykv / mi / plaing / shared-mi / upload-file / kyou-list-view-dialog の
 * 6箇所に手書きでコピーされていて、手順が完全に一致していたのは rykv の列ループだけだった。
 * 特に `load_all` の `force_attached` を落としている実装では、
 * `Kyou.clone()` が `is_attached_tags_loaded` を引き継ぐせいで
 * `InfoBase.load_attached_tags(false)` が早期 return し、
 * **添付タグを一度も引き直さない**（タグを追加しても表示が変わらない）。
 *
 * 正しい手順は次の4つで、1つでも欠けると引き直しに失敗する。
 *   1. ServiceWorker のPOSTキャッシュを捨てる
 *   2. `reload(true, query)` で最新のメタ情報を取る
 *   3. `is_typed_data_loaded` を倒す（clone が引き継ぐため）
 *   4. `load_all(query, true)` で種別データと添付データを強制的に引き直す
 */

// ── Query resolution ──

export type KyouReloadQuery = FindKyouQuery | undefined

/**
 * 引き直しに使うクエリ。
 * rykv だけは「列のクエリ × リスト要素の data_type」から導出する必要があるので、
 * 値ではなく関数も受け取れるようにしている。
 */
export type KyouReloadQueryResolver = KyouReloadQuery | ((kyou_in_list: Kyou) => KyouReloadQuery)

export interface RefreshKyouInListOptions {
    query?: KyouReloadQueryResolver
    /**
     * 引き直した Kyou をリストへ書き戻す方法。
     *
     * 省略時は元の配列を in-place で splice する。
     * `KyouListViewDialog` / `UploadFileView` は `model_value` が親の配列そのものなので
     * in-place でないと親と縁が切れる。
     * 逆に rykv / mi / plaing / shared-mi は `Ref<Array<...>>` への copy-on-write で
     * 反応性を飛ばしているので、そちらは `replace` を渡すこと。
     */
    replace?: (next_list: Array<Kyou>) => void
    /**
     * 同じ更新から派生した引き直しをまとめるための時刻。`new_reload_batch()` の戻り値。
     * 1回の更新で列・focused・開いているダイアログが独立に引き直すので、
     * それらに同じ値を渡すと1往復に合流する。
     */
    requested_at?: number
}

/**
 * Mi の並び順はクエリの `mi_sort_type` で決まるので、引き直しにも同じ条件を渡す必要がある。
 * mi 以外の data_type では undefined を返す（クエリなしで引く）。
 */
export function build_mi_reload_query(base_query: FindKyouQuery, data_type: string): FindKyouQuery | undefined {
    if (!data_type.startsWith("mi")) {
        return undefined
    }
    const query = base_query.clone()
    query.for_mi = true
    // MiReKyouはmirekyou_*で来るので接頭辞を落として同じ扱いにする
    const suffix = data_type.startsWith("mirekyou_") ? data_type.slice("mirekyou_".length) : data_type.slice("mi_".length)
    switch (suffix) {
        case "start": query.mi_sort_type = MiSortType.estimate_start_time; break
        case "end": query.mi_sort_type = MiSortType.estimate_end_time; break
        case "limit": query.mi_sort_type = MiSortType.limit_time; break
        default: query.mi_sort_type = MiSortType.create_time; break
    }
    return query
}

function resolve_query(resolver: KyouReloadQueryResolver | undefined, kyou_in_list: Kyou): KyouReloadQuery {
    if (typeof resolver === 'function') {
        return resolver(kyou_in_list)
    }
    return resolver
}

// ── In-flight coalescing ──

/**
 * 同じ Kyou に対する引き直しを合流させる。
 *
 * 1回のタグ追加で「ダイアログ内のリスト」「ページの列」「focused_kyou」「開いているダイアログ」が
 * 独立にリフレッシュを始めるため、合流させないと同一 Kyou に対して
 * 1リフレッシュあたり約5往復 × 4系統 = 20往復のリクエストが飛ぶ。
 *
 * デバウンス（時間待ち）ではなく合流なので、最速のリクエストの結果を全員が受け取る。
 * レイテンシは増えない。決着したら即座にマップから消す
 * （キャッシュではないのでTTLは持たない。持つと「更新直後に別経路で更新→古い結果が返る」が起きる）。
 *
 * ただし合流してよいのは「同じ更新から派生した引き直し」だけ。無条件に合流していたころは、
 * ダイアログを開いたときの引き直し（`open_rykv_dialog` が毎回投げる）がまだ飛行中のうちに
 * 保存すると、そこへぶら下がって更新前の Kyou を配り、表示を古い内容で上書きしていた。
 * 同じ更新から派生したことは呼び出し元しか知らないので、`requested_at` を渡してもらって
 * 「その時刻より後に始まった引き直しだけ相乗りしてよい」と判定する。
 */
interface InFlightReload {
    promise: Promise<Kyou | null>
    /**
     * 引き直しを始めた時刻。実際の通信を始めるより前に取るので、
     * `started_at >= requested_at` なら、その要求より前に決着した書き込みを必ず見ている。
     */
    started_at: number
}

/**
 * 同じ更新から派生した引き直しに共通で渡す時刻。`performance.now()` を1回だけ取って使い回す。
 * これを渡さないと呼び出し時刻が使われ、飛行中の引き直しには相乗りしない（安全側に倒れる）。
 */
export function new_reload_batch(): number {
    return performance.now()
}

const in_flight_reloads = new Map<string, InFlightReload>()

// ── Reloading registry ──

/**
 * いまどの Kyou を引き直している最中か（id ごとの実行中件数）。
 *
 * KyouView は id が同じなら再マウントされず props が差し替わるだけなので、
 * 「引き直し中」をコンポーネントローカルに持つと引き直し完了で倒せない。
 * 表示側（use-kyou-view.ts）は id で購読する。
 */
const reloading_counts = reactive(new Map<string, number>())

/** その Kyou を引き直している最中か。Kyou 単位のスピナーの表示条件に使う */
export function is_kyou_reloading(id: string): boolean {
    return (reloading_counts.get(id) ?? 0) > 0
}

function begin_reloading(id: string): void {
    reloading_counts.set(id, (reloading_counts.get(id) ?? 0) + 1)
}

function end_reloading(id: string): void {
    const rest = (reloading_counts.get(id) ?? 0) - 1
    if (rest <= 0) {
        reloading_counts.delete(id)
        return
    }
    reloading_counts.set(id, rest)
}

/**
 * `FindKyouQuery` 全体をキーにすると `query_id` の UUID で毎回別キーになるので、
 * 引き直し結果に効くフィールドだけを見る。
 */
function build_reload_key(id: string, query?: FindKyouQuery): string {
    if (!query) {
        return `${id}|`
    }
    return `${id}|${query.for_mi ? '1' : '0'}|${query.mi_sort_type}`
}

// ── Refresh ──

/** 引き直し1回ぶん。失敗は投げる */
async function fetch_refreshed_kyou(kyou: Kyou, query?: FindKyouQuery): Promise<Kyou> {
    const refreshed = kyou.clone()
    // 引き直しは呼び出し元のダイアログや行より長生きさせる。
    // 呼び出し元の AbortController を引き継ぐと、KyouView の onUnmounted abort で
    // 「保存してダイアログを閉じた直後の引き直し」が道連れに死ぬ。
    // 今は clone() が abort_controller を写さないので実質そうなっているが、
    // clone() に1行足されただけで静かに壊れるため、ここで明示的に切り離しておく
    refreshed.abort_controller = new AbortController()
    try {
        await delete_gkill_kyou_cache(kyou.id)
    } catch (_e) {
        // Cache API が利用できない環境ではスキップ
    }
    await refreshed.reload(true, query)
    // clone() が is_typed_data_loaded を引き継ぐので、倒さないと load_typed_datas が
    // 早期returnして種別データが古いまま残る
    refreshed.is_typed_data_loaded = false
    // force_attached=true でないと is_attached_*_loaded が立っている Kyou の
    // 添付データ（タグ/テキスト/通知）を引き直せない
    await refreshed.load_all(query, true)
    return refreshed
}

/** 失敗したら1回だけ引き直す。それでも駄目なら null（呼び出し元はリストを触らない） */
const refresh_attempt_limit = 2

async function do_refresh_kyou(kyou: Kyou, query?: FindKyouQuery): Promise<Kyou | null> {
    let last_error: unknown = null
    for (let attempt = 0; attempt < refresh_attempt_limit; attempt++) {
        try {
            return await fetch_refreshed_kyou(kyou, query)
        } catch (err: unknown) {
            // 一時的な失敗で更新が落ちたまま古い表示が残り続けるのを避ける
            last_error = err
        }
    }
    // 半分だけ読めた Kyou で表示を差し替えるより、古いまま残すほうが安全。
    // ただし黙って諦めると画面上は「再読み込みされない」としか見えないので、
    // どの Kyou で落ちたかまで必ず残す
    console.error(`failed to refresh kyou: id=${kyou.id}`, last_error)
    return null
}

/**
 * Kyou 1件を最新化した新しいインスタンスを返す。引数の `kyou` は変更しない。
 * 失敗したときは null を返す（呼び出し元はリストを触らないこと）。
 *
 * `requested_at` には、同じ更新から派生した引き直し全部に `new_reload_batch()` の
 * 同じ値を渡す。省略すると呼び出し時刻になり、飛行中の引き直しには相乗りしない。
 */
export async function refresh_kyou(kyou: Kyou, query?: FindKyouQuery, requested_at: number = performance.now()): Promise<Kyou | null> {
    const key = build_reload_key(kyou.id, query)
    begin_reloading(kyou.id)
    try {
        const in_flight = in_flight_reloads.get(key)
        // 相乗りしてよいのは requested_at より後に始まった引き直しだけ。
        // requested_at より前に始まったものは、その更新をまだ見ていない可能性がある。
        // 無条件に合流していたころは、ダイアログを開いたときの引き直し
        // （open_rykv_dialog が毎回投げる）がまだ飛行中のうちに保存すると、
        // 更新前の Kyou を掴んで列・focused・ダイアログを一斉に古い内容へ戻していた
        if (in_flight && in_flight.started_at >= requested_at) {
            const shared = await in_flight.promise
            // 同一インスタンスを複数のリストに置くと後段の load_typed_datas 等で副作用が出る
            return shared ? shared.clone() : null
        }

        const started_at = performance.now()
        const entry: InFlightReload = { promise: do_refresh_kyou(kyou, query), started_at }
        in_flight_reloads.set(key, entry)
        try {
            const refreshed = await entry.promise
            return refreshed ? refreshed.clone() : null
        } finally {
            // 待っている間に後発の引き直しへ差し替わっていたら、そちらを消してはいけない
            if (in_flight_reloads.get(key) === entry) {
                in_flight_reloads.delete(key)
            }
        }
    } finally {
        end_reloading(kyou.id)
    }
}

/**
 * リスト内の同じ id の Kyou を最新化して差し替える。
 * id が一致する要素が無ければ何もしない（リクエストも飛ばさない）。
 */
export async function refresh_kyou_in_list(list: Array<Kyou>, kyou: Kyou, options?: RefreshKyouInListOptions): Promise<void> {
    // 走査は生の配列に対して行う。listはdeepなref配下のリアクティブProxyなので、
    // 素で list[i] を読むと1要素ごとに track と toReactive が走り、
    // 要素ぶんのProxyを確保する(30万件の列では効く)。読み取りだけなので意味論は同じ。
    // ★書き戻し(splice)は必ずリアクティブな list に対して行うこと。
    const raw_list = toRaw(list)
    let first_index = -1
    for (let i = 0; i < raw_list.length; i++) {
        if (raw_list[i].id === kyou.id) {
            first_index = i
            break
        }
    }
    if (first_index === -1) {
        return
    }

    const query = resolve_query(options?.query, raw_list[first_index])
    const refreshed = await refresh_kyou(kyou, query, options?.requested_at)
    if (!refreshed) {
        return
    }

    // 書き戻す位置は**awaitのあとに取り直す**。
    // 待っている間に局所挿入や削除でリストが動きうるので、
    // 待つ前のインデックスで splice すると別の行を潰す。idで引き直せばずれない。
    const raw_list_after_await = toRaw(list)
    // 同一インスタンスを複数行に置くと後段のload_typed_datas等で副作用が出るのでクローンする
    let used_refreshed = false
    const next_kyou_for = (): Kyou => {
        if (used_refreshed) {
            return refreshed.clone()
        }
        used_refreshed = true
        return refreshed
    }

    if (options?.replace) {
        const next_list = [...list]
        for (let i = 0; i < next_list.length; i++) {
            if (next_list[i].id === kyou.id) {
                next_list[i] = next_kyou_for()
            }
        }
        options.replace(next_list)
        return
    }

    for (let i = 0; i < raw_list_after_await.length; i++) {
        if (raw_list_after_await[i].id === kyou.id) {
            list.splice(i, 1, next_kyou_for())
        }
    }
}
