/**
 * export-i18n（出力 HTML の中のロケール）。出力 HTML は通信をしないので、ja 以外のメッセージは焼き込みから読む。
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { i18n, apply_baked_locale, set_locale, is_supported_locale } from '@/export-i18n'

describe('export-i18n', () => {
    beforeEach(async () => {
        await set_locale('ja')
    })

    it('is_supported_locale は7言語だけを受け入れる', () => {
        expect(is_supported_locale('ja')).toBe(true)
        expect(is_supported_locale('en')).toBe(true)
        expect(is_supported_locale('de')).toBe(true)
        expect(is_supported_locale('xx')).toBe(false)
    })

    it('apply_baked_locale は ja・未対応ロケール・不正メッセージでは何もしない', () => {
        apply_baked_locale('xx', { KEY: 'val' })
        expect(i18n.global.locale).toBe('ja')

        apply_baked_locale('ja', { KEY: 'val' })
        expect(i18n.global.locale).toBe('ja')

        apply_baked_locale('en', null)
        expect(i18n.global.locale).toBe('ja')

        apply_baked_locale('en', { SAMPLE_EXPORT_KEY: 'Exported English' })
        expect(i18n.global.locale).toBe('en')
        expect(i18n.global.t('SAMPLE_EXPORT_KEY')).toBe('Exported English')
    })
})
