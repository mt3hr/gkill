package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

/**
 * Unit tests for GkillApiClient using MockWebServer.
 * Tests login, submitKFTLText, and getKftlTemplateStructJson methods.
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
}
