/**
 * 地図へ渡す表示言語と地域。
 *
 * 渡さないと地図の読み込み URL に `region=undefined&language=undefined` が乗る。
 * ローダは渡されたオブジェクトのキーを `for...in` で回して `URLSearchParams.set` するので、
 * 値が `undefined` のキーも文字列 `"undefined"` になる。しかも `GoogleMap` の props には
 * 既定値が無く、ローダへ渡すオブジェクトには両方のキーが必ず置かれるので、
 * **こちらから値を渡す以外に消す方法が無い**。
 * 言語が無効だと地図のラベルがアプリの表示言語に従わない。
 *
 * 地域は「分からないときは空文字」。gkill は地域の設定を持っていないので、
 * 推測で埋めると利用者が頼んでいない偏りが地図にかかる。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    i18n: { global: { locale: 'ja' } },
}))

import { i18n } from '@/i18n'
import { get_google_map_language, get_google_map_region, get_region_from_language_tag } from '@/classes/google-map-locale'

describe('地図の地域（言語タグの地域サブタグ）', () => {
    test.each([
        ['ja-JP', 'JP'],
        ['en-US', 'US'],
        ['zh-Hans-CN', 'CN'],
        ['sr-Latn-RS', 'RS'],
        ['ja-jp', 'JP'],
    ])('%s から %s を取る', (tag, expected) => {
        expect(get_region_from_language_tag(tag)).toBe(expected)
    })

    test.each([
        ['ja'],
        ['en'],
        // UN M49 の数値の地域。2文字の国コードではないので採らない
        ['es-419'],
        [''],
    ])('%s からは取れないので空文字', (tag) => {
        expect(get_region_from_language_tag(tag)).toBe('')
    })

    test('言語タグが無いときも空文字（undefined を返さない）', () => {
        expect(get_region_from_language_tag(null)).toBe('')
        expect(get_region_from_language_tag(undefined)).toBe('')
    })

    test('ブラウザの言語タグから取る', () => {
        const original = navigator.language
        Object.defineProperty(navigator, 'language', { value: 'de-DE', configurable: true })
        expect(get_google_map_region()).toBe('DE')
        Object.defineProperty(navigator, 'language', { value: original, configurable: true })
    })
})

describe('地図の表示言語', () => {
    test('アプリの表示言語をそのまま使う', () => {
        i18n.global.locale = 'fr'
        expect(get_google_map_language()).toBe('fr')
        i18n.global.locale = 'ja'
    })

    test('ロケールが Ref で来ても読める', () => {
        // legacy では素の文字列、composer では Ref になりうる。
        // format-date-time.ts の format_number と同じ読み方をしている
        const original = i18n.global.locale
        ;(i18n.global as unknown as { locale: unknown }).locale = { value: 'ko' }
        expect(get_google_map_language()).toBe('ko')
        i18n.global.locale = original
    })

    test('読めないときは ja へ落ちる（undefined を返さない）', () => {
        const original = i18n.global.locale
        ;(i18n.global as unknown as { locale: unknown }).locale = ''
        expect(get_google_map_language()).toBe('ja')
        i18n.global.locale = original
    })
})
