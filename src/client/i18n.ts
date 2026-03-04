import { createI18n } from "vue-i18n"
import ja from './locales/ja.json'
import en from './locales/en.json'
import zh from './locales/zh.json'
import ko from './locales/ko.json'
import es from './locales/es.json'
import fr from './locales/fr.json'
import de from './locales/de.json'

export const i18n = createI18n({
    legacy: true,
    globalInjection: true,
    locale: 'ja',
    fallbackLocale: 'ja',
    messages: {
        ja,
        en,
        zh,
        ko,
        es,
        fr,
        de,
    }
})