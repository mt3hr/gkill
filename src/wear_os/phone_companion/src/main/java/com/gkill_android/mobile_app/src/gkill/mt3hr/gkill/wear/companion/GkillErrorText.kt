package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.content.Context
import androidx.annotation.StringRes

/**
 * companion 自身が生成するエラーコード（[WIRE_ERROR_CODES]）を利用者に見せる文言へ変える唯一の場所。
 *
 * [WearRequestHandler] と [GkillApiClient] は Android に依存しない純粋クラスなので日本語を持たず、
 * `login_failed` のような ASCII コードを返す。時計へ送る直前（[WearRequestWorker]）と
 * 設定画面の表示（[MainActivity]）でここを通して訳す。
 *
 * 時計側では訳さない。時計に届く文言はスマホのロケールになるが、サーバーの `error_message`
 * （`locale_name` はスマホが決める）も同じ性質なので一貫する。時計側で訳すと照合表が
 * 2モジュールに二重化し、サーバー文言だけスマホロケールのままで割れる。
 *
 * 未知の文字列（サーバーの `error_message`、`HTTP 500`、例外メッセージ）はそのまま返す。
 */
object GkillErrorText {

    /** 既知のコードならその文言のリソースID、未知なら null。 */
    @StringRes
    fun stringResOf(code: String): Int? = when (code) {
        WIRE_ERR_LOGIN_FAILED -> R.string.error_login_failed
        WIRE_ERR_GET_CONFIG_FAILED -> R.string.error_get_config_failed
        WIRE_ERR_GET_PLAYING_TIMEIS_FAILED -> R.string.error_get_playing_timeis_failed
        WIRE_ERR_EMPTY_TIMEIS_ID -> R.string.error_empty_timeis_id
        WIRE_ERR_EMPTY_RESPONSE -> R.string.error_empty_response
        WIRE_ERR_EMPTY_SESSION_ID -> R.string.error_empty_session_id
        WIRE_ERR_UNKNOWN -> R.string.error_unknown
        WIRE_ERR_GET_TIMEIS_FAILED -> R.string.error_get_timeis_failed
        WIRE_ERR_TIMEIS_NOT_FOUND -> R.string.error_timeis_not_found
        WIRE_ERR_UPDATE_TIMEIS_FAILED -> R.string.error_update_timeis_failed
        else -> null
    }

    /** [raw] が既知コードなら訳した文言、そうでなければ [raw] のまま。 */
    fun localize(context: Context, raw: String): String {
        val resId = stringResOf(raw) ?: return raw
        return context.getString(resId)
    }

    /**
     * 時計へ送る応答本文を訳す。`ERROR:` で始まるときだけ接頭辞の後ろを [localize] し、
     * `OK` / `DUPLICATE` / JSON はバイト列をそのまま返す。
     */
    fun localizeWireResponse(context: Context, data: ByteArray): ByteArray {
        val text = String(data, Charsets.UTF_8)
        if (!text.startsWith(WIRE_ERROR_PREFIX)) return data
        val code = text.removePrefix(WIRE_ERROR_PREFIX)
        val localized = localize(context, code)
        if (localized == code) return data
        return (WIRE_ERROR_PREFIX + localized).toByteArray(Charsets.UTF_8)
    }
}
