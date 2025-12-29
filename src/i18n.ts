import { createI18n } from "vue-i18n"
import ja from './locales/ja.json'
import en from './locales/en.json'
import zh from './locales/zh.json'
import ko from './locales/ko.json'
import es from './locales/es.json'
import fr from './locales/fr.json'
import de from './locales/de.json'

const locale_name_cookie_key = "gkill_locale_name"
function get_locale_name_from_cookie(): string {
    const cookies = document.cookie.split(';')
    const locale_name_string = cookies.find(
        (cookie) => cookie.split('=')[0].trim() === locale_name_cookie_key.trim()
    )?.replace(locale_name_cookie_key + "=", "").trim()
    const locale_name = locale_name_string ? locale_name_string : "ja"
    return locale_name
}

let locale: 'ja' | 'en' | 'zh' | 'ko' | 'es' | 'fr' | 'de' = 'ja'
const locale_in_cookie = get_locale_name_from_cookie()
switch (locale_in_cookie) {
    case 'ja':
    case 'en':
    case 'zh':
    case 'ko':
    case 'es':
    case 'fr':
    case 'de':
        locale = locale_in_cookie
        break
    default:
        switch (window.navigator.language) {
            case 'ja':
            case 'en':
            case 'zh':
            case 'ko':
            case 'es':
            case 'fr':
            case 'de':
                locale = window.navigator.language
                break
            default:
                locale = 'ja'
        }
}

export const i18n = createI18n({
    legacy: true,
    globalInjection: true,
    locale: locale,
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