'use strict'

// バイト列から文字コードを推定してデコードする。
// BOM検出 → UTF-8厳密 → Shift_JIS → EUC-JP → 置換文字付きUTF-8フォールバック。
export function detect_and_decode_text(bytes: Uint8Array): string {
    // BOM検出: UTF-8 / UTF-16LE / UTF-16BE
    if (bytes.length >= 3 && bytes[0] === 0xEF && bytes[1] === 0xBB && bytes[2] === 0xBF) {
        return new TextDecoder('utf-8').decode(bytes.slice(3))
    }
    if (bytes.length >= 2 && bytes[0] === 0xFF && bytes[1] === 0xFE) {
        return new TextDecoder('utf-16le').decode(bytes.slice(2))
    }
    if (bytes.length >= 2 && bytes[0] === 0xFE && bytes[1] === 0xFF) {
        return new TextDecoder('utf-16be').decode(bytes.slice(2))
    }
    // UTF-8 厳密デコード
    try {
        return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
    } catch { /* UTF-8 ではない */ }
    // Shift_JIS (CP932)
    try {
        return new TextDecoder('shift_jis', { fatal: true }).decode(bytes)
    } catch { /* Shift_JIS ではない */ }
    // EUC-JP
    try {
        return new TextDecoder('euc-jp', { fatal: true }).decode(bytes)
    } catch { /* EUC-JP ではない */ }
    // フォールバック: 置換文字付き UTF-8
    return new TextDecoder('utf-8').decode(bytes)
}
