import { is_url } from '@/classes/looks-like-url'

export interface TextSegment {
    is_url: boolean
    text: string
}

// URL候補: https?:// から始まる連続文字。
// 日本語文は URL の直後に空白を挟まず本文が続くことが多いため、
// 空白に加えて和文の文字（ひらがな・カタカナ・漢字・全角記号）も区切りとして扱う。
// ブラウザからコピーした URL はパーセントエンコード済みなので、生の和文パスを持つ URL は対象外とする。
const url_candidate_pattern = /https?:\/\/[^\s"'<>、-ヿ一-鿿！-ﾟ]+/g

// URL 末尾から取り除く約物。「)」だけは括弧の対応が取れている場合に残す（Wikipedia 等の URL 対策）。
const trailing_punctuation = new Set(['.', ',', ';', ':', '!', '?', ')', ']', '}'])

function count_char(text: string, target: string): number {
    let count = 0
    for (const ch of text) {
        if (ch === target) count++
    }
    return count
}

function trim_trailing_punctuation(candidate: string): string {
    let end = candidate.length
    while (end > 0) {
        const ch = candidate[end - 1]
        if (!trailing_punctuation.has(ch)) break
        if (ch === ')') {
            const body = candidate.slice(0, end)
            if (count_char(body, ')') <= count_char(body, '(')) break
        }
        end--
    }
    return candidate.slice(0, end)
}

// テキストを「URL」と「それ以外」のセグメント列に分割する。
// リンク化は http/https のみ（最終判定は is_url に委ねる）。URL が無ければ全体を1セグメントで返す。
export function split_text_by_urls(text: string): TextSegment[] {
    if (!text) return []
    const segments: TextSegment[] = []
    let cursor = 0
    for (const match of text.matchAll(url_candidate_pattern)) {
        const url = trim_trailing_punctuation(match[0])
        if (!is_url(url)) continue
        if (match.index > cursor) {
            segments.push({ is_url: false, text: text.slice(cursor, match.index) })
        }
        segments.push({ is_url: true, text: url })
        cursor = match.index + url.length
    }
    if (cursor < text.length || segments.length === 0) {
        segments.push({ is_url: false, text: text.slice(cursor) })
    }
    return segments
}
