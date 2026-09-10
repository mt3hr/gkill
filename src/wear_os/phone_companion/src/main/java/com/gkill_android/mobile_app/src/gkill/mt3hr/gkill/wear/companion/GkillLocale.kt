package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.content.Context

/**
 * gkill サーバーへ送る `locale_name` の決め方。
 *
 * 値は `Locale.getDefault()` からではなく、リソース `R.string.server_locale_name`
 * （`values/`=ja、`values-en/`=en、…）から引く。API 24+ の端末は言語の優先リストを持ち、
 * リソース解決は「リスト中でアプリが持つ最初の言語」を選ぶ。利用者のリストが `[ar, en]` なら
 * UI は `values-en` になるが `Locale.getDefault().language` は `ar` を返すので、言語コードを
 * 自前で判定すると UI は英語・サーバーの error_message は日本語（フォールバック）に割れる。
 * リソースから引けば LocaleList・Android 13 のアプリ別言語設定・スクリプト一致まで
 * すべてリソース解決に委ねられ、UI と必ず同じ言語になる。
 *
 * サーバー側は `src/server/gkill/api/embed.go` の `GetLocalizer` が ja / en / zh / ko / es / fr / de
 * 以外を ja に落とすので、値の集合は `values-*` のディレクトリ名と一致させておく
 * （StringsParityTest が各 `values-xx` の値が `xx` であることを検査する）。
 */
object GkillLocale {

    /**
     * リソースが無い場面（JVM 単体テスト）での既定値。サーバー側フォールバックと同じ ja。
     * `Locale.getDefault()` 由来にすると単体テストの結果が実行機の OS 言語で変わるので定数にしてある。
     */
    const val DEFAULT_LOCALE_NAME = "ja"

    /** UI が解決したロケールに対応する言語コードを返す。 */
    fun serverLocaleName(context: Context): String =
        context.getString(R.string.server_locale_name)
}
