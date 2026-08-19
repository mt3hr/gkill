'use strict'

/**
 * Web Push の VAPID 公開鍵（URL-safe base64）を `pushManager.subscribe` に渡せる形へ変換する。
 *
 * 6つのページコンポーザブル（kftl / kyou / mi / mkfl / plaing / rykv）が
 * 同じ関数をそれぞれ持っていて、戻り値の型注釈の有無だけが食い違っていた。
 * 型を書いていない3本は推論に任せていたぶん `as any` が要らず、
 * 書いていた3本は `applicationServerKey` に渡すところで `as any` を足していた
 * ——つまり「同じ処理なのに片方だけ any で押し切る」状態だったのでここへ寄せる。
 *
 * 戻り値を `Uint8Array<ArrayBuffer>` と明示するのが要点。
 * TypeScript 5.7 以降 `Uint8Array` はバッファ型で総称化されており、
 * 既定の `Uint8Array<ArrayBufferLike>` は `BufferSource` を満たさないため、
 * ここを曖昧にすると呼び出し側でキャストが必要になる。
 */
export function url_base64_to_uint8_array(base64_string: string): Uint8Array<ArrayBuffer> {
    const padding = '='.repeat((4 - (base64_string.length % 4)) % 4)
    /* eslint no-useless-escape: 0 */
    const base64 = (base64_string + padding).replace(/\-/g, '+').replace(/_/g, '/')
    const raw_data = window.atob(base64)
    const bytes = new Uint8Array(new ArrayBuffer(raw_data.length))
    for (let i = 0; i < raw_data.length; i++) {
        bytes[i] = raw_data.charCodeAt(i)
    }
    return bytes
}
