package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.content.Context
import io.mockk.every
import io.mockk.mockk
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

/**
 * エラーコード → 文言の照合（GkillErrorText）のテスト。
 *
 * Context.getString はリソースを読めないので、リソースIDを "res:<id>" の形で返す MockK に差し替える。
 * 「どのIDに解決されたか」と「未知の文字列を素通しするか」を見るのが目的で、訳文そのものは見ない。
 */
class GkillErrorTextTest {

    private lateinit var context: Context

    @Before
    fun setUp() {
        context = mockk()
        every { context.getString(any<Int>()) } answers { "res:${firstArg<Int>()}" }
    }

    private fun res(id: Int): String = "res:$id"

    // -----------------------------------------------------------------------
    // stringResOf
    // -----------------------------------------------------------------------

    // ハンドラ / API クライアントが生成しうるコード全件に訳がある。
    // コードを足して strings.xml と stringResOf を忘れると、ここで落ちる。
    @Test
    fun stringResOf_everyWireErrorCode_hasResource() {
        for (code in WIRE_ERROR_CODES) {
            assertNotNull("no string resource for code '$code'", GkillErrorText.stringResOf(code))
        }
    }

    @Test
    fun stringResOf_codesMapToDistinctResources() {
        val ids = WIRE_ERROR_CODES.map { GkillErrorText.stringResOf(it) }
        assertEquals("two codes share one string resource", ids.size, ids.toSet().size)
    }

    @Test
    fun stringResOf_knownCodes_mapToExpectedResources() {
        assertEquals(R.string.error_login_failed, GkillErrorText.stringResOf(WIRE_ERR_LOGIN_FAILED))
        assertEquals(R.string.error_empty_session_id, GkillErrorText.stringResOf(WIRE_ERR_EMPTY_SESSION_ID))
        assertEquals(R.string.error_timeis_not_found, GkillErrorText.stringResOf(WIRE_ERR_TIMEIS_NOT_FOUND))
    }

    @Test
    fun stringResOf_unknownStrings_returnNull() {
        // サーバーの error_message・HTTP フォールバック・例外メッセージ・空文字は訳さない
        assertNull(GkillErrorText.stringResOf("Invalid credentials"))
        assertNull(GkillErrorText.stringResOf("HTTP 500"))
        assertNull(GkillErrorText.stringResOf("timeout"))
        assertNull(GkillErrorText.stringResOf(""))
        // 大文字小文字は区別する（コードは小文字固定）
        assertNull(GkillErrorText.stringResOf("LOGIN_FAILED"))
    }

    // -----------------------------------------------------------------------
    // localize
    // -----------------------------------------------------------------------

    @Test
    fun localize_knownCode_returnsResourceText() {
        assertEquals(res(R.string.error_login_failed), GkillErrorText.localize(context, WIRE_ERR_LOGIN_FAILED))
        assertEquals(res(R.string.error_unknown), GkillErrorText.localize(context, WIRE_ERR_UNKNOWN))
    }

    @Test
    fun localize_unknownString_passesThrough() {
        assertEquals("Invalid credentials", GkillErrorText.localize(context, "Invalid credentials"))
        assertEquals("HTTP 403", GkillErrorText.localize(context, "HTTP 403"))
        assertEquals("", GkillErrorText.localize(context, ""))
    }

    // -----------------------------------------------------------------------
    // localizeWireResponse
    // -----------------------------------------------------------------------

    private fun bytes(s: String): ByteArray = s.toByteArray(Charsets.UTF_8)
    private fun text(b: ByteArray): String = String(b, Charsets.UTF_8)

    @Test
    fun localizeWireResponse_errorWithKnownCode_translatesBody() {
        val out = GkillErrorText.localizeWireResponse(context, bytes("ERROR:login_failed"))
        assertEquals("ERROR:" + res(R.string.error_login_failed), text(out))
    }

    @Test
    fun localizeWireResponse_errorWithUnknownMessage_isUnchanged() {
        val input = bytes("ERROR:Invalid KFTL syntax")
        val out = GkillErrorText.localizeWireResponse(context, input)
        assertArrayEquals(input, out)
    }

    @Test
    fun localizeWireResponse_okAndDuplicate_areUnchanged() {
        assertArrayEquals(bytes("OK"), GkillErrorText.localizeWireResponse(context, bytes("OK")))
        assertArrayEquals(bytes("DUPLICATE"), GkillErrorText.localizeWireResponse(context, bytes("DUPLICATE")))
    }

    @Test
    fun localizeWireResponse_jsonPayload_isUnchanged() {
        // テンプレート一覧・実行中一覧は JSON をそのまま時計へ渡す
        val json = bytes("""[{"name":"root","children":[]}]""")
        assertArrayEquals(json, GkillErrorText.localizeWireResponse(context, json))
    }

    @Test
    fun localizeWireResponse_prefixOnly_isUnchanged() {
        val input = bytes("ERROR:")
        assertArrayEquals(input, GkillErrorText.localizeWireResponse(context, input))
    }

    @Test
    fun localizeWireResponse_codeInsideText_isNotTranslated() {
        // コードは本文全体との完全一致でしか訳さない（部分一致で誤訳しない）
        val input = bytes("ERROR:login_failed: details")
        assertArrayEquals(input, GkillErrorText.localizeWireResponse(context, input))
    }
}
