// GPSログのページングカーソルのコーデック。
//
// **正本はこの1ファイルだけにすること。** 以前は encode/decode が read-handlers.mjs に、
// 受け取り側の検証が normalization.mjs にあり、後者が gkill_get_kyous 用の複合カーソル
// (`{RFC3339Nano}::{ID}`、Go製) の検証をコピペしたままだった。GPSカーソルは Node 製の
// base64url なので Date.parse が必ず NaN になり、**説明文どおり next_cursor を verbatim で
// 渡すと 100% `Invalid argument 'cursor'` になっていた**。
// 発行側と検証側が別実装だったことが原因なので、両方からここを import する。

import { GkillApiError } from "./errors.mjs";

/**
 * encodeGpsCursor は次ページの開始位置をカーソル文字列にする。
 *
 * t=最後に返した点の related_time、n=同一時刻の中で消費済みの点数。
 * サーバの並びは (related_time降順, latitude昇順, longitude昇順) で決定的なので、
 * 同一時刻ランの途中でも位置を特定できる（get_kyous v2 と同じ考え方）。
 *
 * @param {string} t 最後に返した点の related_time。
 * @param {number} n 同一時刻ラン内で消費済みの点数。
 * @returns {string} base64url(JSON {t, n})。
 */
export function encodeGpsCursor(t, n) {
  return Buffer.from(JSON.stringify({ t, n }), "utf8").toString("base64url");
}

/**
 * decodeGpsCursor はカーソル文字列を {t, n} に戻す。解釈できなければ投げる。
 *
 * @param {string} cursor encodeGpsCursor が返した文字列。
 * @returns {{t: string, n: number}} 復元した位置。
 */
export function decodeGpsCursor(cursor) {
  try {
    const decoded = JSON.parse(Buffer.from(cursor, "base64url").toString("utf8"));
    if (typeof decoded.t === "string" && Number.isInteger(decoded.n) && decoded.n >= 0) {
      return decoded;
    }
  } catch {
    // fallthrough
  }
  throw new GkillApiError(`Invalid GPS cursor: ${JSON.stringify(cursor)} (pass next_cursor verbatim)`);
}

/**
 * isValidGpsCursor は文字列が GPS カーソルとして解釈できるかを返す。
 *
 * 入口の検証（normalization）用。ここで弾いておかないと、後段のエラーが
 * 「カーソルが原因」だと分からない形に畳まれる。
 *
 * @param {string} cursor 検査する文字列。
 * @returns {boolean} 解釈できれば true。
 */
export function isValidGpsCursor(cursor) {
  try {
    decodeGpsCursor(cursor);
    return true;
  } catch {
    return false;
  }
}
