package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * Unit tests for WearRequestHandler using MockWebServer.
 *
 * The handler is the Android-free core extracted from the old service handlers, so
 * every path (success / failure / ERROR: prefix / DUPLICATE) is testable on the JVM.
 * The session provider is stubbed and the duplicate ledger uses in-memory storage.
 */
class WearRequestHandlerTest {

    private lateinit var mockServer: MockWebServer
    private lateinit var apiClient: GkillApiClient

    private class MemoryStorage : WearSubmitLedger.Storage {
        private var value = ""
        override fun read(): String = value
        override fun write(value: String) {
            this.value = value
        }
    }

    private fun response(text: String): MockResponse =
        MockResponse().setBody(text).setResponseCode(200)

    private fun handler(
        session: String? = "test-session",
        ledger: WearSubmitLedger? = null,
        idempotencyKey: String? = null,
    ): WearRequestHandler = WearRequestHandler(
        apiClient = apiClient,
        sessionProvider = { session },
        submitLedger = ledger,
        idempotencyKey = idempotencyKey,
    )

    @Before
    fun setUp() {
        mockServer = MockWebServer()
        mockServer.start()
        apiClient = GkillApiClient(mockServer.url("/").toString().trimEnd('/'))
    }

    @After
    fun tearDown() {
        mockServer.shutdown()
    }

    // ─── dispatch ────────────────────────────────────────────────────────────

    @Test
    fun handle_unknownPath_returnsNull() {
        assertNull(handler().handle("/gkill/nope", ByteArray(0)))
    }

    // ─── get_templates ───────────────────────────────────────────────────────

    @Test
    fun handleGetTemplates_success_returnsTemplatesJson() {
        mockServer.enqueue(response("""{"application_config":{"kftl_template_struct":{"name":"root"}},"errors":null}"""))

        val resp = handler().handleGetTemplates()

        assertEquals(PATH_TEMPLATES, resp.path)
        assertTrue(String(resp.data, Charsets.UTF_8).contains("root"))
    }

    @Test
    fun handleGetTemplates_loginFailed_returnsErrorLoginFailed() {
        val resp = handler(session = null).handleGetTemplates()

        assertEquals(PATH_TEMPLATES, resp.path)
        assertEquals("ERROR:login_failed", String(resp.data, Charsets.UTF_8))
    }

    @Test
    fun handleGetTemplates_configError_returnsErrorGetConfigFailed() {
        mockServer.enqueue(response("""{"application_config":null,"errors":[{"error_code":"NO_SESSION","error_message":"expired"}]}"""))

        val resp = handler().handleGetTemplates()

        assertEquals("ERROR:get_config_failed", String(resp.data, Charsets.UTF_8))
    }

    // ─── submit ──────────────────────────────────────────────────────────────

    @Test
    fun handleSubmit_success_returnsOkAndRecordsLedger() {
        mockServer.enqueue(response("""{"errors":null}"""))
        val ledger = WearSubmitLedger(MemoryStorage())
        val h = handler(ledger = ledger)

        val resp = h.handleSubmit("/m memo！".toByteArray(Charsets.UTF_8), force = false)

        assertEquals(PATH_SUBMIT_RESULT, resp.path)
        assertEquals("OK", String(resp.data, Charsets.UTF_8))
        // recorded only after a successful save
        assertTrue(ledger.isDuplicate("/m memo！"))
    }

    @Test
    fun handleSubmit_error_returnsErrorAndDoesNotRecord() {
        mockServer.enqueue(response("""{"errors":[{"error_code":"PARSE_ERROR","error_message":"bad"}]}"""))
        val ledger = WearSubmitLedger(MemoryStorage())
        val h = handler(ledger = ledger)

        val resp = h.handleSubmit("/m bad".toByteArray(Charsets.UTF_8), force = false)

        assertEquals("ERROR:bad", String(resp.data, Charsets.UTF_8))
        assertFalse(ledger.isDuplicate("/m bad"))
    }

    @Test
    fun handleSubmit_loginFailed_returnsErrorLoginFailed() {
        val resp = handler(session = null).handleSubmit("/m memo".toByteArray(Charsets.UTF_8), force = false)

        assertEquals(PATH_SUBMIT_RESULT, resp.path)
        assertEquals("ERROR:login_failed", String(resp.data, Charsets.UTF_8))
    }

    @Test
    fun handleSubmit_duplicate_returnsDuplicateWithoutCallingServer() {
        val ledger = WearSubmitLedger(MemoryStorage())
        ledger.recordSuccess("/m dup")
        val h = handler(ledger = ledger)

        val resp = h.handleSubmit("/m dup".toByteArray(Charsets.UTF_8), force = false)

        assertEquals(PATH_SUBMIT_RESULT, resp.path)
        assertEquals("DUPLICATE", String(resp.data, Charsets.UTF_8))
        // duplicate is rejected before any HTTP call
        assertEquals(0, mockServer.requestCount)
    }

    @Test
    fun handleSubmit_withIdempotencyKey_sendsKeyInBody() {
        mockServer.enqueue(response("""{"errors":null}"""))
        val h = handler(idempotencyKey = "wear-key-1")

        h.handleSubmit("/m memo".toByteArray(Charsets.UTF_8), force = false)

        val body = mockServer.takeRequest().body.readUtf8()
        assertTrue(body.contains(""""idempotency_key":"wear-key-1""""))
    }

    @Test
    fun handleSubmit_withoutIdempotencyKey_omitsKeyFromBody() {
        mockServer.enqueue(response("""{"errors":null}"""))
        val h = handler()

        h.handleSubmit("/m memo".toByteArray(Charsets.UTF_8), force = false)

        // encodeDefaults=false なので空キーは本文に出ない
        val body = mockServer.takeRequest().body.readUtf8()
        assertFalse(body.contains("idempotency_key"))
    }

    @Test
    fun handleSubmit_force_overridesDuplicate() {
        mockServer.enqueue(response("""{"errors":null}"""))
        val ledger = WearSubmitLedger(MemoryStorage())
        ledger.recordSuccess("/m dup")
        val h = handler(ledger = ledger)

        val resp = h.handleSubmit("/m dup".toByteArray(Charsets.UTF_8), force = true)

        assertEquals("OK", String(resp.data, Charsets.UTF_8))
        assertEquals(1, mockServer.requestCount)
    }

    // ─── get_plaing_timeis ───────────────────────────────────────────────────

    @Test
    fun handleGetPlaingTimeis_success_returnsJsonArray() {
        mockServer.enqueue(response("""{"kyous":[],"errors":null}"""))

        val resp = handler().handleGetPlaingTimeis()

        assertEquals(PATH_PLAING_TIMEIS, resp.path)
        assertEquals("[]", String(resp.data, Charsets.UTF_8))
    }

    @Test
    fun handleGetPlaingTimeis_error_returnsErrorPrefix() {
        mockServer.enqueue(response("""{"kyous":null,"errors":[{"error_code":"NO_SESSION","error_message":"expired"}]}"""))

        val resp = handler().handleGetPlaingTimeis()

        assertEquals("ERROR:get_plaing_timeis_failed", String(resp.data, Charsets.UTF_8))
    }

    @Test
    fun handleGetPlaingTimeis_loginFailed_returnsErrorLoginFailed() {
        val resp = handler(session = null).handleGetPlaingTimeis()

        assertEquals("ERROR:login_failed", String(resp.data, Charsets.UTF_8))
    }

    // ─── end_timeis ──────────────────────────────────────────────────────────

    @Test
    fun handleEndTimeis_success_returnsOk() {
        mockServer.enqueue(response("""{"timeis_histories":[{"id":"t1","title":"x"}],"errors":null}"""))
        mockServer.enqueue(response("""{"errors":null}"""))

        val resp = handler().handleEndTimeis("t1\nmy_rep".toByteArray(Charsets.UTF_8))

        assertEquals(PATH_END_TIMEIS_RESULT, resp.path)
        assertEquals("OK", String(resp.data, Charsets.UTF_8))
    }

    @Test
    fun handleEndTimeis_emptyId_returnsError() {
        val resp = handler().handleEndTimeis("".toByteArray(Charsets.UTF_8))

        assertEquals("ERROR:empty_timeis_id", String(resp.data, Charsets.UTF_8))
        assertEquals(0, mockServer.requestCount)
    }

    @Test
    fun handleEndTimeis_loginFailed_returnsErrorLoginFailed() {
        val resp = handler(session = null).handleEndTimeis("t1\nmy_rep".toByteArray(Charsets.UTF_8))

        assertEquals("ERROR:login_failed", String(resp.data, Charsets.UTF_8))
    }
}
