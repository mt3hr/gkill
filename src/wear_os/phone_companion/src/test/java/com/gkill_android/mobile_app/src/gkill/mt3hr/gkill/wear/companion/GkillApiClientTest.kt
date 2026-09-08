package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import kotlinx.serialization.json.jsonObject
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import okhttp3.tls.HandshakeCertificates
import okhttp3.tls.HeldCertificate
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

/**
 * Unit tests for GkillApiClient using MockWebServer.
 * Tests login, submitKFTLText, getKftlTemplateStructJson, getPlayingTimeis,
 * and endTimeis methods.
 */
class GkillApiClientTest {

    private lateinit var mockServer: MockWebServer
    private lateinit var client: GkillApiClient

    @Before
    fun setUp() {
        mockServer = MockWebServer()
        mockServer.start()
        val baseUrl = mockServer.url("/").toString().trimEnd('/')
        client = GkillApiClient(baseUrl)
    }

    @After
    fun tearDown() {
        mockServer.shutdown()
    }

    // ─── login ─────────────────────────────────────────────────────────────

    @Test
    fun login_success_returnsSessionId() {
        val responseJson = """{"session_id":"abc-session-123","errors":null}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val sessionId = client.login("admin", "sha256hash")

        assertEquals("abc-session-123", sessionId)

        val request = mockServer.takeRequest()
        assertEquals("/api/login", request.path)
        assertEquals("POST", request.method)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"user_id\":\"admin\""))
        assertTrue(body.contains("\"password_sha256\":\"sha256hash\""))
    }

    @Test
    fun login_withErrors_returnsNull() {
        val responseJson = """{"session_id":"","errors":[{"error_code":"AUTH_FAILED","error_message":"Invalid credentials"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val sessionId = client.login("admin", "wronghash")

        assertNull(sessionId)
    }

    @Test
    fun loginWithError_success_returnsSessionIdAndEmptyError() {
        val responseJson = """{"session_id":"session-xyz","errors":null}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val (sessionId, errorMsg) = client.loginWithError("admin", "sha256hash")

        assertEquals("session-xyz", sessionId)
        assertEquals("", errorMsg)
    }

    @Test
    fun loginWithError_withErrors_returnsNullAndErrorMessage() {
        val responseJson = """{"session_id":"","errors":[{"error_code":"AUTH_FAILED","error_message":"Invalid credentials"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val (sessionId, errorMsg) = client.loginWithError("admin", "wronghash")

        assertNull(sessionId)
        assertEquals("Invalid credentials", errorMsg)
    }

    @Test
    fun login_emptySessionId_returnsNull() {
        val responseJson = """{"session_id":"","errors":null}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val sessionId = client.login("admin", "sha256hash")

        assertNull(sessionId)
    }

    @Test
    fun loginWithError_emptySessionId_returnsNullAndMessage() {
        val responseJson = """{"session_id":"","errors":null}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val (sessionId, errorMsg) = client.loginWithError("admin", "sha256hash")

        assertNull(sessionId)
        assertEquals("セッションIDが空です", errorMsg)
    }

    @Test
    fun login_httpError_returnsNull() {
        mockServer.enqueue(MockResponse().setResponseCode(500))

        val sessionId = client.login("admin", "sha256hash")

        assertNull(sessionId)
    }

    @Test
    fun loginWithError_httpError_returnsHttpCode() {
        mockServer.enqueue(MockResponse().setResponseCode(500))

        val (sessionId, errorMsg) = client.loginWithError("admin", "sha256hash")

        assertNull(sessionId)
        assertEquals("HTTP 500", errorMsg)
    }

    // gkill returns 4xx/5xx on failure, but the reason (error_message) lives only
    // in the response body. The client must read the body first and surface that
    // message; "HTTP 401" is only the fallback for an empty body (a bare status
    // is useless on a watch screen).
    @Test
    fun loginWithError_non2xxWithErrorsBody_returnsBodyErrorMessage() {
        val responseJson = """{"session_id":"","errors":[{"error_code":"AUTH_FAILED","error_message":"Invalid credentials"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(401))

        val (sessionId, errorMsg) = client.loginWithError("admin", "wronghash")

        assertNull(sessionId)
        assertEquals("Invalid credentials", errorMsg)
    }

    // ─── submitKFTLText ────────────────────────────────────────────────────

    @Test
    fun submitKFTLText_success_returnsNull() {
        val responseJson = """{"errors":null}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val error = client.submitKFTLText("session-123", "/m test memo")

        assertNull(error)

        val request = mockServer.takeRequest()
        assertEquals("/api/submit_kftl_text", request.path)
        assertEquals("POST", request.method)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"session_id\":\"session-123\""))
        assertTrue(body.contains("\"kftl_text\":\"/m test memo\""))
        // locale_name may be omitted by kotlinx.serialization when it's the default value
    }

    @Test
    fun submitKFTLText_withErrors_returnsErrorMessage() {
        val responseJson = """{"errors":[{"error_code":"PARSE_ERROR","error_message":"Invalid KFTL syntax"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val error = client.submitKFTLText("session-123", "invalid text")

        assertEquals("Invalid KFTL syntax", error)
    }

    @Test
    fun submitKFTLText_httpError_returnsHttpCode() {
        mockServer.enqueue(MockResponse().setResponseCode(403))

        val error = client.submitKFTLText("session-123", "/m memo")

        assertEquals("HTTP 403", error)
    }

    // Non-2xx with an errors body: the body's error_message wins over "HTTP 400"
    // (the status-only fallback applies only when the body is empty).
    @Test
    fun submitKFTLText_non2xxWithErrorsBody_returnsBodyErrorMessage() {
        val responseJson = """{"errors":[{"error_code":"PARSE_ERROR","error_message":"Invalid KFTL syntax"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(400))

        val error = client.submitKFTLText("session-123", "invalid text")

        assertEquals("Invalid KFTL syntax", error)
    }

    @Test
    fun submitKFTLText_emptyResponse_returnsErrorMessage() {
        mockServer.enqueue(MockResponse().setBody("").setResponseCode(200))

        val error = client.submitKFTLText("session-123", "/m memo")

        // Empty body will cause a parse error, returning the exception message
        assertNotNull(error)
    }

    // ─── getKftlTemplateStructJson ─────────────────────────────────────────

    @Test
    fun getKftlTemplateStructJson_success_returnsJsonString() {
        val templateStruct = """{"name":"root","children":[]}"""
        val responseJson = """{"application_config":{"kftl_template_struct":$templateStruct},"errors":null}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val result = client.getKftlTemplateStructJson("session-123")

        assertNotNull(result)
        assertTrue(result!!.contains("root"))

        val request = mockServer.takeRequest()
        assertEquals("/api/get_application_config", request.path)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"session_id\":\"session-123\""))
        // locale_name may be omitted by kotlinx.serialization when it's the default value
    }

    @Test
    fun getKftlTemplateStructJson_withErrors_returnsNull() {
        val responseJson = """{"application_config":null,"errors":[{"error_code":"NO_SESSION","error_message":"session expired"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val result = client.getKftlTemplateStructJson("expired-session")

        assertNull(result)
    }

    @Test
    fun getKftlTemplateStructJson_httpError_returnsNull() {
        mockServer.enqueue(MockResponse().setResponseCode(500))

        val result = client.getKftlTemplateStructJson("session-123")

        assertNull(result)
    }

    // Non-2xx with an errors body: the body is still read, and its errors mean
    // failure (null). Pins that the errors decision comes from the body, not from
    // a status cut before reading it.
    @Test
    fun getKftlTemplateStructJson_non2xxWithErrorsBody_returnsNull() {
        val responseJson = """{"application_config":null,"errors":[{"error_code":"NO_SESSION","error_message":"session expired"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(401))

        val result = client.getKftlTemplateStructJson("expired-session")

        assertNull(result)
    }

    // ─── getPlayingTimeis ───────────────────────────────────────────────────

    // FindQuery is null-based: unused filters must be omitted (or null), never
    // sent as empty arrays ([] means "enabled with zero selections" = matches
    // nothing). The playing query must therefore contain only playing_time.
    // A legacy client that sent use_*=false with empty arrays would silently
    // get zero results from a null-based server, so this pins the wire shape.
    @Test
    fun getPlayingTimeis_sendsNullBasedPlayingQuery() {
        val responseJson = """{"kyous":[],"errors":null}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val result = client.getPlayingTimeis("session-123")

        assertEquals("[]", result)

        val request = mockServer.takeRequest()
        assertEquals("/api/get_kyous", request.path)
        assertEquals("POST", request.method)
        val body = request.body.readUtf8()
        assertTrue(body.contains("\"session_id\":\"session-123\""))

        val queryJson = kotlinx.serialization.json.Json.parseToJsonElement(body)
            .jsonObject["query"]!!.jsonObject

        // playing_time is the only filter and must be a non-null string
        val playingTime = queryJson["playing_time"]
        assertNotNull(playingTime)
        assertTrue(playingTime is kotlinx.serialization.json.JsonPrimitive)
        assertTrue((playingTime as kotlinx.serialization.json.JsonPrimitive).isString)

        // no legacy use_* keys
        for (key in queryJson.keys) {
            assertFalse("legacy flag key must not be sent: $key", key.startsWith("use_"))
        }

        // unused filters must be omitted, not sent as empty arrays
        for (key in listOf("tags", "reps", "rep_types", "ids", "words", "not_words", "timeis_words", "timeis_tags", "mi_board_name")) {
            assertFalse("unused filter key must be omitted: $key", queryJson.containsKey(key))
        }
    }

    @Test
    fun getPlayingTimeis_withErrors_returnsNull() {
        val responseJson = """{"kyous":null,"errors":[{"error_code":"NO_SESSION","error_message":"session expired"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(200))

        val result = client.getPlayingTimeis("expired-session")

        assertNull(result)
    }

    // Non-2xx get_kyous with an errors body: the failure decision comes from the
    // body's errors array (read after the status), and the call returns null.
    @Test
    fun getPlayingTimeis_non2xxWithErrorsBody_returnsNull() {
        val responseJson = """{"kyous":null,"errors":[{"error_code":"NO_SESSION","error_message":"session expired"}]}"""
        mockServer.enqueue(MockResponse().setBody(responseJson).setResponseCode(401))

        val result = client.getPlayingTimeis("expired-session")

        assertNull(result)
    }

    // Non-2xx get_kyous must not cut processing on status alone: when the body is
    // valid JSON without errors, the kyous are processed and get_timeis is still
    // called. Pins the no-status-cut semantics (the client previously returned
    // null on non-2xx before reading the body).
    @Test
    fun getPlayingTimeis_non2xxGetKyousWithValidBody_continuesProcessing() {
        val kyousJson = """{"kyous":[{"id":"timeis-1","rep_name":"TimeIs"}],"errors":null}"""
        mockServer.enqueue(MockResponse().setBody(kyousJson).setResponseCode(500))
        val timeisJson = """{"timeis_histories":[{"title":"work","start_time":"2026-01-01T10:00:00+09:00","data_type":"timeis_start","is_deleted":false}],"errors":null}"""
        mockServer.enqueue(MockResponse().setBody(timeisJson).setResponseCode(200))

        val result = client.getPlayingTimeis("session-123")

        assertNotNull(result)
        assertTrue(result!!.contains("timeis-1"))
        assertTrue(result.contains("work"))

        // Both requests were actually made: processing continued past the 500.
        assertEquals("/api/get_kyous", mockServer.takeRequest().path)
        assertEquals("/api/get_timeis", mockServer.takeRequest().path)
    }

    // ─── endTimeis ─────────────────────────────────────────────────────────

    // Non-2xx update_timeis with an errors body: the body's error_message is
    // returned instead of "HTTP 409" (the status-only fallback applies only when
    // the body is empty).
    @Test
    fun endTimeis_non2xxUpdateWithErrorsBody_returnsBodyErrorMessage() {
        val getTimeisJson = """{"timeis_histories":[{"id":"timeis-1","title":"work","start_time":"2026-01-01T10:00:00+09:00","data_type":"timeis_start","is_deleted":false}],"errors":null}"""
        mockServer.enqueue(MockResponse().setBody(getTimeisJson).setResponseCode(200))
        val updateJson = """{"errors":[{"error_code":"CONFLICT","error_message":"update conflict"}]}"""
        mockServer.enqueue(MockResponse().setBody(updateJson).setResponseCode(409))

        val error = client.endTimeis("session-123", "timeis-1", "TimeIs")

        assertEquals("update conflict", error)

        assertEquals("/api/get_timeis", mockServer.takeRequest().path)
        assertEquals("/api/update_timeis", mockServer.takeRequest().path)
    }

    // ─── TLS pinning (H-05) ──────────────────────────────────────────────────
    // A throwaway self-signed HeldCertificate stands in for a localhost gkill
    // server. No real server certificate, SAN, or hostname is used.

    /** Starts a fresh HTTPS MockWebServer that serves [held]. */
    private fun startHttpsServer(held: HeldCertificate): MockWebServer {
        val server = MockWebServer()
        val serverCerts = HandshakeCertificates.Builder().heldCertificate(held).build()
        server.useHttps(serverCerts.sslSocketFactory(), false)
        server.start()
        return server
    }

    private val loginOk = """{"session_id":"pinned-session","errors":null}"""

    // (a) pinned fingerprint matches the leaf → handshake succeeds → login works.
    @Test
    fun tls_pinMatch_loginSucceeds() {
        val held = HeldCertificate.Builder().addSubjectAlternativeName("localhost").build()
        val server = startHttpsServer(held)
        try {
            server.enqueue(MockResponse().setBody(loginOk).setResponseCode(200))
            val fingerprint = GkillServerTrust.certSha256Hex(held.certificate)
            val baseUrl = server.url("/").toString().trimEnd('/')
            val c = GkillApiClient(baseUrl, allowSelfSignedCert = true, pinnedCertSha256 = fingerprint)

            assertEquals("pinned-session", c.login("admin", "hash"))
        } finally {
            server.shutdown()
        }
    }

    // (b) pinned fingerprint does not match → handshake fails, but the presented
    // fingerprint is captured and marked rejected (for the learning UI).
    @Test
    fun tls_pinMismatch_loginFailsAndCapturesFingerprint() {
        val held = HeldCertificate.Builder().addSubjectAlternativeName("localhost").build()
        val server = startHttpsServer(held)
        try {
            server.enqueue(MockResponse().setBody(loginOk).setResponseCode(200))
            val baseUrl = server.url("/").toString().trimEnd('/')
            val c = GkillApiClient(baseUrl, allowSelfSignedCert = true, pinnedCertSha256 = "00".repeat(32))

            assertNull(c.login("admin", "hash"))
            assertEquals(GkillServerTrust.certSha256Hex(held.certificate), c.lastServerCertSha256)
            assertTrue(c.lastServerCertRejected)
        } finally {
            server.shutdown()
        }
    }

    // (c) background-equivalent: pinned self-signed mode but no pin stored yet →
    // the self-signed cert is rejected (the service/worker never learns pins).
    @Test
    fun tls_noPin_loginFailsAndCapturesFingerprint() {
        val held = HeldCertificate.Builder().addSubjectAlternativeName("localhost").build()
        val server = startHttpsServer(held)
        try {
            server.enqueue(MockResponse().setBody(loginOk).setResponseCode(200))
            val baseUrl = server.url("/").toString().trimEnd('/')
            val c = GkillApiClient(baseUrl, allowSelfSignedCert = true, pinnedCertSha256 = null)

            assertNull(c.login("admin", "hash"))
            assertTrue(c.lastServerCertRejected)
            assertEquals(GkillServerTrust.certSha256Hex(held.certificate), c.lastServerCertSha256)
        } finally {
            server.shutdown()
        }
    }

    // (d) default mode (allowSelfSignedCert=false) trusts the platform store only.
    // A self-signed server is rejected and nothing is captured/offered for pinning.
    @Test
    fun tls_defaultMode_selfSignedRejectedNoCapture() {
        val held = HeldCertificate.Builder().addSubjectAlternativeName("localhost").build()
        val server = startHttpsServer(held)
        try {
            server.enqueue(MockResponse().setBody(loginOk).setResponseCode(200))
            val baseUrl = server.url("/").toString().trimEnd('/')
            val c = GkillApiClient(baseUrl, allowSelfSignedCert = false)

            assertNull(c.login("admin", "hash"))
            assertNull(c.lastServerCertSha256)
            assertFalse(c.lastServerCertRejected)
        } finally {
            server.shutdown()
        }
    }

    // (e) SAN-less self-signed cert: the default hostname verifier fails, but the
    // pin byte-match fallback in the verifier allows it.
    @Test
    fun tls_sanlessCert_pinFallbackAllowsHostname() {
        val held = HeldCertificate.Builder().commonName("gkill-no-san").build()
        val server = startHttpsServer(held)
        try {
            server.enqueue(MockResponse().setBody(loginOk).setResponseCode(200))
            val fingerprint = GkillServerTrust.certSha256Hex(held.certificate)
            val baseUrl = server.url("/").toString().trimEnd('/')
            val c = GkillApiClient(baseUrl, allowSelfSignedCert = true, pinnedCertSha256 = fingerprint)

            assertEquals("pinned-session", c.login("admin", "hash"))
        } finally {
            server.shutdown()
        }
    }
}
