'use strict'

import { i18n } from '@/i18n'

// Go側 src/server/gkill/api/kftl/kftl_factory.go のASCIIプレフィックス定数と対応
export const KFTL_ASCII_TAG_PREFIX = "#"
export const KFTL_ASCII_TEXT_SPLITTER_TITLE = "--"
export const KFTL_ASCII_RELATED_TIME_PREFIX = "?"
export const KFTL_ASCII_SPLIT_PREFIX = ","
export const KFTL_ASCII_SPLIT_APPEND_TIME_PREFIX = ",,"
export const KFTL_ASCII_KC_SPLITTER_TITLE = "/num"
export const KFTL_ASCII_MI_SPLITTER_TITLE = "/mi"
export const KFTL_ASCII_MI_REKYOU_SPLITTER_TITLE = "~~"
export const KFTL_ASCII_LANTANA_SPLITTER_TITLE = "/mood"
export const KFTL_ASCII_NLOG_SPLITTER_TITLE = "/expense"
export const KFTL_ASCII_TIMEIS_START_SPLITTER_TITLE = "/start"
export const KFTL_ASCII_TIMEIS_END_SPLITTER_TITLE = "/end"
export const KFTL_ASCII_TIMEIS_SPLITTER_TITLE = "/timeis"
export const KFTL_ASCII_TIMEIS_END_END_SPLITTER_TITLE = "/end?"
export const KFTL_ASCII_TIMEIS_END_TAG_END_SPLITTER_TITLE = "/endt"
export const KFTL_ASCII_TIMEIS_END_IF_TAG_END_SPLITTER_TITLE = "/endt?"
export const KFTL_ASCII_URLOG_SPLITTER_TITLE = "/url"
export const KFTL_ASCII_SAVE_CHARACTOR = "!"
export const KFTL_ASCII_TIMEIS_TIME_PREFIX = "?"

/**
 * 波ダッシュ(U+301C)を全角チルダ(U+FF5E)に揃える。
 *
 * 「～」はWindowsのIMEがU+FF5E、macOS/iOSのIMEがU+301Cを出す。見た目が同じで
 * 打った端末によって別の文字になるので、揃えずに比較すると
 * iOSのPWAからだけリポストタスクの記法が効かない。i18nに入れている値はU+FF5E。
 * Go側 src/server/gkill/api/kftl/kftl_factory.go の normalizeWaveDash と対応
 */
export function normalize_wave_dash(line_text: string): string {
    return line_text.replaceAll("〜", "～")
}

// 行全体がプレフィックスと一致するか(日本語はi18nキー、ASCIIは定数)
export function matches_exact(line_text: string, i18n_key: string, ascii_prefix: string): boolean {
    return line_text === i18n.global.t(i18n_key) || line_text === ascii_prefix
}

// 行がプレフィックスで始まるか(日本語はi18nキー、ASCIIは定数)
export function matches_prefix(line_text: string, i18n_key: string, ascii_prefix: string): boolean {
    return line_text.startsWith(i18n.global.t(i18n_key)) || line_text.startsWith(ascii_prefix)
}

// 行頭のプレフィックスのみ除去する(行中の出現は除去しない)
export function strip_prefix(line_text: string, i18n_key: string, ascii_prefix: string): string {
    const ja_prefix = i18n.global.t(i18n_key)
    if (line_text.startsWith(ja_prefix)) {
        return line_text.slice(ja_prefix.length)
    }
    if (line_text.startsWith(ascii_prefix)) {
        return line_text.slice(ascii_prefix.length)
    }
    return line_text
}

// タグ列を「、」または「,」で分割する
export function split_tags(text: string): Array<string> {
    return text.split(/[、,]/)
}

// 保存文字(全角「！」または半角「!」)の行か
export function is_save_charactor_line(line_text: string): boolean {
    return line_text === i18n.global.t("KFTL_SAVE_CHARACTOR") || line_text === KFTL_ASCII_SAVE_CHARACTOR
}
