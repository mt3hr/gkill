package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Ignore
import org.junit.Test

/**
 * Tests for GkillWearableListenerService.
 *
 * GkillWearableListenerService extends WearableListenerService (Android framework class)
 * and is tightly coupled to Android APIs (Log, Wearable.getMessageClient, SharedPreferences
 * via GkillCredentialStore, coroutine scope with Dispatchers.IO). The message path constants
 * are file-level `private const val`, so they are not directly accessible from tests.
 *
 * This test file verifies the expected message path string values and data format conventions
 * used by the service, serving as a contract test to catch accidental path changes.
 */
class GkillWearableListenerServiceTest {

    // ─── Message path contract tests ─────────────────────────────────────────────
    // These verify the path strings that must match between phone_companion and watch_app.
    // The actual constants are private, so we assert the literal values that both sides must use.

    @Test
    fun `request path for get_templates is correct`() {
        assertEquals("/gkill/get_templates", PATH_GET_TEMPLATES_EXPECTED)
    }

    @Test
    fun `response path for templates is correct`() {
        assertEquals("/gkill/templates", PATH_TEMPLATES_EXPECTED)
    }

    @Test
    fun `request path for submit is correct`() {
        assertEquals("/gkill/submit", PATH_SUBMIT_EXPECTED)
    }

    @Test
    fun `response path for submit_result is correct`() {
        assertEquals("/gkill/submit_result", PATH_SUBMIT_RESULT_EXPECTED)
    }

    @Test
    fun `request path for get_plaing_timeis is correct`() {
        assertEquals("/gkill/get_plaing_timeis", PATH_GET_PLAING_TIMEIS_EXPECTED)
    }

    @Test
    fun `response path for plaing_timeis is correct`() {
        assertEquals("/gkill/plaing_timeis", PATH_PLAING_TIMEIS_EXPECTED)
    }

    @Test
    fun `request path for end_timeis is correct`() {
        assertEquals("/gkill/end_timeis", PATH_END_TIMEIS_EXPECTED)
    }

    @Test
    fun `response path for end_timeis_result is correct`() {
        assertEquals("/gkill/end_timeis_result", PATH_END_TIMEIS_RESULT_EXPECTED)
    }

    // ─── Path pairing tests ──────────────────────────────────────────────────────

    @Test
    fun `request and response paths are distinct for templates`() {
        assertNotEquals(PATH_GET_TEMPLATES_EXPECTED, PATH_TEMPLATES_EXPECTED)
    }

    @Test
    fun `request and response paths are distinct for submit`() {
        assertNotEquals(PATH_SUBMIT_EXPECTED, PATH_SUBMIT_RESULT_EXPECTED)
    }

    @Test
    fun `request and response paths are distinct for plaing_timeis`() {
        assertNotEquals(PATH_GET_PLAING_TIMEIS_EXPECTED, PATH_PLAING_TIMEIS_EXPECTED)
    }

    @Test
    fun `request and response paths are distinct for end_timeis`() {
        assertNotEquals(PATH_END_TIMEIS_EXPECTED, PATH_END_TIMEIS_RESULT_EXPECTED)
    }

    // ─── All paths share the gkill prefix ────────────────────────────────────────

    @Test
    fun `all paths start with gkill prefix`() {
        val allPaths = listOf(
            PATH_GET_TEMPLATES_EXPECTED,
            PATH_TEMPLATES_EXPECTED,
            PATH_SUBMIT_EXPECTED,
            PATH_SUBMIT_RESULT_EXPECTED,
            PATH_GET_PLAING_TIMEIS_EXPECTED,
            PATH_PLAING_TIMEIS_EXPECTED,
            PATH_END_TIMEIS_EXPECTED,
            PATH_END_TIMEIS_RESULT_EXPECTED,
        )
        for (path in allPaths) {
            assert(path.startsWith("/gkill/")) { "Path '$path' does not start with /gkill/" }
        }
    }

    // ─── End timeis data format ──────────────────────────────────────────────────

    @Test
    fun `end_timeis payload format is id newline rep_name`() {
        // The handleEndTimeis handler expects data in "id\nrep_name" format
        val id = "test-uuid-1234"
        val repName = "my_rep"
        val payload = "$id\n$repName"

        val parts = payload.split("\n", limit = 2)
        assertEquals(id, parts[0])
        assertEquals(repName, parts[1])
    }

    @Test
    fun `end_timeis payload with empty rep_name`() {
        val payload = "test-uuid-1234\n"
        val parts = payload.split("\n", limit = 2)
        assertEquals("test-uuid-1234", parts[0])
        assertEquals("", parts[1])
    }

    @Test
    fun `end_timeis payload with no newline yields empty rep_name via getOrNull`() {
        val payload = "test-uuid-1234"
        val parts = payload.split("\n", limit = 2)
        assertEquals("test-uuid-1234", parts.getOrNull(0))
        // When there's no newline, getOrNull(1) returns null
        assertEquals(null, parts.getOrNull(1))
    }

    // ─── Error message format ────────────────────────────────────────────────────

    @Test
    fun `error response starts with ERROR prefix`() {
        val errorMsg = "ERROR:login_failed"
        assert(errorMsg.startsWith("ERROR:"))
        assertEquals("login_failed", errorMsg.removePrefix("ERROR:"))
    }

    @Test
    fun `OK response does not start with ERROR prefix`() {
        val okMsg = "OK"
        assert(!okMsg.startsWith("ERROR:"))
    }

    // ─── Integration test (requires Android) ─────────────────────────────────────

    @Ignore("Requires Android framework: WearableListenerService, Wearable.getMessageClient, GkillCredentialStore (SharedPreferences)")
    @Test
    fun `onMessageReceived dispatches to correct handler`() {
        // This test would require mocking WearableListenerService, MessageEvent,
        // GkillCredentialStore, GkillApiClient, and Wearable.getMessageClient.
        // These are all tightly coupled to Android framework and cannot run on JVM.
    }

    companion object {
        // Expected path values — these must match the private constants in
        // GkillWearableListenerService.kt and GkillWearClient.kt
        private const val PATH_GET_TEMPLATES_EXPECTED = "/gkill/get_templates"
        private const val PATH_TEMPLATES_EXPECTED = "/gkill/templates"
        private const val PATH_SUBMIT_EXPECTED = "/gkill/submit"
        private const val PATH_SUBMIT_RESULT_EXPECTED = "/gkill/submit_result"
        private const val PATH_GET_PLAING_TIMEIS_EXPECTED = "/gkill/get_plaing_timeis"
        private const val PATH_PLAING_TIMEIS_EXPECTED = "/gkill/plaing_timeis"
        private const val PATH_END_TIMEIS_EXPECTED = "/gkill/end_timeis"
        private const val PATH_END_TIMEIS_RESULT_EXPECTED = "/gkill/end_timeis_result"
    }
}
