import { createI18n } from "vue-i18n"
import ja from '../locales/ja.json'

/** gkill が対応するロケール */
export const SUPPORTED_LOCALES = ['ja', 'en', 'zh', 'ko', 'es', 'fr', 'de'] as const
export type SupportedLocale = typeof SUPPORTED_LOCALES[number]

export function isSupportedLocale(locale: string): locale is SupportedLocale {
    return (SUPPORTED_LOCALES as readonly string[]).includes(locale)
}

// ja のみ静的に持ち、他言語は使うときに動的importする。
// 7言語すべて同梱すると352KB分がmain chunkにインライン展開されてしまうため。
// fallbackLocale が ja なので、未ロードのキーは日本語で表示される。
// 型は全ロケール分を宣言しておく (実体は set_locale が後から流し込む)
const messages: Partial<Record<SupportedLocale, typeof ja>> = { ja }

export const i18n = createI18n({
    legacy: true,
    globalInjection: true,
    locale: 'ja',
    fallbackLocale: 'ja',
    messages,
})

// Vite が JS チャンク (/assets/<locale>-<hash>.js) に変換する。
// サーバ側の静的アセット判定 (utils.go) が /assets/ prefix で拾えるよう、
// 素の .json を fetch する形にはしないこと。
// import(`../locales/${locale}.json`) のようなテンプレート形式にすると
// 静的importしている ja の分までチャンクが二重に生成されるため、明示的に列挙する
const locale_loaders: Record<Exclude<SupportedLocale, 'ja'>, () => Promise<{ default: typeof ja }>> = {
    en: () => import('../locales/en.json'),
    zh: () => import('../locales/zh.json'),
    ko: () => import('../locales/ko.json'),
    es: () => import('../locales/es.json'),
    fr: () => import('../locales/fr.json'),
    de: () => import('../locales/de.json'),
}

const loaded_locales = new Set<SupportedLocale>(['ja'])

/**
 * 指定ロケールのメッセージを読み込んで適用する。
 * 読み込み済みのロケールでは何もしない。
 */
export async function set_locale(locale: SupportedLocale): Promise<void> {
    if (!loaded_locales.has(locale) && locale !== 'ja') {
        const messages = await locale_loaders[locale]()
        i18n.global.setLocaleMessage(locale, messages.default)
        loaded_locales.add(locale)
    }
    i18n.global.locale = locale
}
