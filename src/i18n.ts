import { createI18n } from "vue-i18n"
import ja from './locales/ja.json'
import original from './locales/original.json'

export const i18n = createI18n({
    legacy: true,
    globalInjection: true,
    locale: 'ja',
    fallbackLocale: 'original',
    messages: {
        original,
        ja,
    }
})