'use strict'

import { i18n } from '@/i18n'

/**
 * Google Maps の読み込みに渡す表示言語と地域。
 *
 * **2つある地図（`gps-log-map.vue` と `map-query.vue`）は必ず同じ値を渡すこと。**
 * 地図のローダは `setOptions` の**最初の1回だけ**を採り、2回目以降は警告して無視する。
 * 呼び出し側ごとに違う値を書くと、どちらが先にマウントされたかで URL が変わる。
 *
 * **渡さないと `undefined` が URL に乗る。** ローダは渡されたオブジェクトのキーを
 * `for...in` で回して `URLSearchParams.set` するので、値が `undefined` のキーも
 * 文字列 `"undefined"` になって `?...&region=undefined&language=undefined` が出来上がる。
 * `GoogleMap` の props には既定値が無く、`setOptions` へ渡すオブジェクトには
 * `region` / `language` のキーが必ず置かれるので、**こちらから値を渡す以外に消す方法が無い**。
 * 言語が無効だと地図のラベルがアプリの表示言語に従わない（見た目に出る）。
 */

/** 言語タグから地域サブタグを取る。`ja-JP` → `JP`、`zh-Hans-CN` → `CN` */
const REGION_SUBTAG_PATTERN = /^[A-Za-z]{2,3}(?:-[A-Za-z]{4})?-([A-Za-z]{2})(?:-|$)/

/**
 * 地図の表示言語。アプリの表示言語をそのまま使う。
 *
 * `i18n.global.locale` は legacy では素の文字列、composer では `Ref` になりうるので、
 * `format-date-time.ts` の `format_number` と同じ読み方をする。
 */
export function get_google_map_language(): string {
    return (i18n.global.locale as unknown as { value: string }).value || i18n.global.locale || 'ja'
}

/**
 * 地図の地域。**ブラウザの言語タグの地域サブタグだけを使う**（`ja-JP` → `JP`）。
 *
 * gkill は地域の設定を持っていないので、持っていない値を作り出さない。
 * 地域が分からないときは**空文字**を返す —— 地域は地図の振る舞いに偏りをかける指定なので、
 * 推測で埋めると利用者が頼んでいない偏りが入る。空は「指定なし」として扱われる。
 *
 * `es-419` のような数値の地域（UN M49）は2文字の国コードではないので採らない。
 */
export function get_google_map_region(): string {
    if (typeof navigator === 'undefined') {
        return ''
    }
    return get_region_from_language_tag(navigator.language)
}

/** 言語タグから地域を取り出す。取れないときは空文字。テストから直接呼ぶ */
export function get_region_from_language_tag(language_tag: string | null | undefined): string {
    if (!language_tag) {
        return ''
    }
    const matched = REGION_SUBTAG_PATTERN.exec(language_tag)
    if (!matched) {
        return ''
    }
    return matched[1].toUpperCase()
}
