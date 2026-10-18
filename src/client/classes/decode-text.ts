'use strict'

// バイト列から文字コードを推定してデコードする。
// BOM検出 → UTF-8厳密 → Shift_JIS → EUC-JP → 置換文字付きUTF-8フォールバック。
//
// truncated: ファイルの先頭だけを読んだバイト列のとき true にする。
// 読むのを止めた位置は文字の境目とは限らず、末尾に切れた多バイト文字が残る。
// そのまま厳密デコードすると UTF-8 の判定が末尾の切れ端だけで失敗し、Shift_JIS / EUC-JP として
// 読まれて全体が文字化けする（どれも外れれば置換文字が付く）。stream: true で渡すと
// TextDecoder は末尾の未完の並びを次の入力待ちとして保留するので、切れ端だけが落ちる。
export function detect_and_decode_text(bytes: Uint8Array, options: { truncated?: boolean } = {}): string {
    const decode_options = { stream: options.truncated === true }
    // BOM検出: UTF-8 / UTF-16LE / UTF-16BE
    if (bytes.length >= 3 && bytes[0] === 0xEF && bytes[1] === 0xBB && bytes[2] === 0xBF) {
        return new TextDecoder('utf-8').decode(bytes.slice(3), decode_options)
    }
    if (bytes.length >= 2 && bytes[0] === 0xFF && bytes[1] === 0xFE) {
        return new TextDecoder('utf-16le').decode(bytes.slice(2), decode_options)
    }
    if (bytes.length >= 2 && bytes[0] === 0xFE && bytes[1] === 0xFF) {
        return new TextDecoder('utf-16be').decode(bytes.slice(2), decode_options)
    }
    // UTF-8 厳密デコード
    try {
        return new TextDecoder('utf-8', { fatal: true }).decode(bytes, decode_options)
    } catch { /* UTF-8 ではない */ }
    // Shift_JIS (CP932)
    try {
        return new TextDecoder('shift_jis', { fatal: true }).decode(bytes, decode_options)
    } catch { /* Shift_JIS ではない */ }
    // EUC-JP
    try {
        return new TextDecoder('euc-jp', { fatal: true }).decode(bytes, decode_options)
    } catch { /* EUC-JP ではない */ }
    // フォールバック: 置換文字付き UTF-8
    return new TextDecoder('utf-8').decode(bytes, decode_options)
}
