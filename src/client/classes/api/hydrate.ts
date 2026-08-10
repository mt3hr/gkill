'use strict'

import { yield_to_main } from './yield-to-main'

// 既定の日付キー判定。呼び出しごとに配列を作らない(大量件数経路では30万回呼ばれる)
const default_date_suffixes = ['time']

// キーがいずれかのsuffixで終わるか。
// クロージャを作らないモジュールレベル関数にしてある。
// 以前は `date_suffixes.some(suffix => key.endsWith(suffix))` で、
// キーごとにアロー関数を生成していた(30万件×15キー ≒ 450万個)
function ends_with_any(key: string, suffixes: Array<string>): boolean {
    for (let i = 0; i < suffixes.length; i++) {
        if (key.endsWith(suffixes[i])) {
            return true
        }
    }
    return false
}

/**
 * サーバから受け取った素のJSONオブジェクトを、クラスインスタンスへ詰め替える。
 *
 * `JSON.parse` の結果をそのままキャストしてもメソッドが生えないため、
 * 生成済みのインスタンス(`target`)へキーをコピーして使う。
 *
 * 挙動の約束:
 * - `source` の**全キー**をコピーする(target の型に無いキーも写す)。
 *   サーバが増やしたフィールドを黙って捨てないための既存挙動を踏襲している。
 * - `date_suffixes` に一致するキーは、値が truthy のときだけ `Date` に変換する。
 *   既定は `["time"]`。変換したくない場合は空配列を渡す。
 */
export function hydrate<T extends object>(
    target: T,
    source: unknown,
    options?: { date_suffixes?: Array<string> },
): T {
    if (source === null || typeof source !== 'object') {
        return target
    }
    const date_suffixes = options?.date_suffixes ?? default_date_suffixes
    const src = source as Record<string, unknown>
    const dst = target as Record<string, unknown>
    for (const key in src) {
        const value = src[key]
        dst[key] = value
        if (value && ends_with_any(key, date_suffixes)) {
            dst[key] = new Date(value as string | number | Date)
        }
    }
    return target
}

/**
 * 配列の各要素を `factory()` で作ったインスタンスへ詰め替えて返す。
 * `source` が配列でない場合は空配列を返す。
 */
export function hydrate_all<T extends object>(
    source: unknown,
    factory: () => T,
    options?: { date_suffixes?: Array<string> },
): Array<T> {
    if (!Array.isArray(source)) {
        return []
    }
    return source.map(element => hydrate(factory(), element, options))
}

/**
 * 大きい配列の実体化をチャンクに区切り、間で `yield_to_main()` して
 * メインスレッドを長時間ブロックしないようにする。
 * `list` の各要素(素のJSON)を `factory()` のインスタンスへin-placeで置き換える。
 *
 * - 1チャンクで終わる小さい配列にはyieldを挟まない。
 *   1件ごとのAPI(`load_attached_timeis`等)の応答は数件で、そこに毎回1tick
 *   足すと並列ロード全体が遅くなるため
 * - 各チャンクの先頭で `signal` を確認し、中断済みならfetch中断と同型の例外
 *   (`signal.reason`、name === "AbortError")をthrowして即座に抜ける。
 *   呼び出し元のcatchと main.ts のunhandledrejection網はこのnameで握る
 */
export async function hydrate_all_chunked<T extends object>(
    list: Array<unknown>,
    factory: () => T,
    options?: { chunk_size?: number, signal?: AbortSignal, date_suffixes?: Array<string> },
): Promise<void> {
    const chunk_size = options?.chunk_size ?? 5000
    const hydrate_options = options?.date_suffixes ? { date_suffixes: options.date_suffixes } : undefined
    for (let i = 0; i < list.length; i += chunk_size) {
        if (i !== 0) {
            await yield_to_main()
        }
        options?.signal?.throwIfAborted()
        const end = Math.min(i + chunk_size, list.length)
        for (let j = i; j < end; j++) {
            list[j] = hydrate(factory(), list[j], hydrate_options)
        }
    }
}
