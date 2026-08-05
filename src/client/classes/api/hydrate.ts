'use strict'

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
    const date_suffixes = options?.date_suffixes ?? ['time']
    const src = source as Record<string, unknown>
    const dst = target as Record<string, unknown>
    for (const key in src) {
        dst[key] = src[key]
        if (dst[key] && date_suffixes.some(suffix => key.endsWith(suffix))) {
            dst[key] = new Date(dst[key] as string | number | Date)
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
