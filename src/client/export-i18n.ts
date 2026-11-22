// 「HTML で出力」のガワ用の i18n。vite.export.config.ts の alias が `@/i18n` をここへ向ける。
//
// 本体の i18n.ts は ja を静的・他6言語を動的 import するが、出力のガワは
// inlineDynamicImports で1チャンクへ畳むので、そのままだと7言語ぶんのメッセージが
// 記録0件のガワにも常駐する（数百KB級）。
//
// 出力 HTML の閲覧者が言語を切り替える要件は無いので、ここは ja だけを内蔵し、
// 他言語のときはサーバが <script type="application/json" id="gkill-export-i18n"> へ
// そのロケールのメッセージを差し込む（export-main.ts が起動時に流し込む）。
// 公開している形は本体の i18n.ts と同じにしておくこと（448箇所が `@/i18n` から
// `i18n` と `set_locale` を import している）。

import { createI18n } from "vue-i18n"
import ja from '../locales/ja.json'

/** gkill が対応するロケール（本体の i18n.ts と同じ） */
export const SUPPORTED_LOCALES = ['ja', 'en', 'zh', 'ko', 'es', 'fr', 'de'] as const
export type SupportedLocale = typeof SUPPORTED_LOCALES[number]

export function is_supported_locale(locale: string): locale is SupportedLocale {
    return (SUPPORTED_LOCALES as readonly string[]).includes(locale)
}

const messages: Partial<Record<SupportedLocale, typeof ja>> = { ja }

export const i18n = createI18n({
    legacy: true,
    globalInjection: true,
    locale: 'ja',
    fallbackLocale: 'ja',
    messages,
})

/**
 * ロケールを適用する。
 *
 * 出力 HTML は焼き込まれた言語ぶんのメッセージしか持たないので、**読み込みはしない**。
 * 未登録のロケールを渡しても fallbackLocale（ja）で表示されるだけで、通信は起きない。
 */
export async function set_locale(locale: SupportedLocale): Promise<void> {
    i18n.global.locale = locale
}

/**
 * サーバが焼き込んだロケールのメッセージを流し込む。
 *
 * export-main.ts が mount の前に1回だけ呼ぶ。差し込みが無い（ja で出力した）ときは
 * 何もしない。messages の形は locales/*.json と同じ平坦なキー値。
 */
export function apply_baked_locale(locale: string, baked_messages: unknown): void {
    if (!is_supported_locale(locale)) {
        return
    }
    if (locale === 'ja' || !baked_messages || typeof baked_messages !== 'object') {
        return
    }
    i18n.global.setLocaleMessage(locale, baked_messages as typeof ja)
    i18n.global.locale = locale
}
