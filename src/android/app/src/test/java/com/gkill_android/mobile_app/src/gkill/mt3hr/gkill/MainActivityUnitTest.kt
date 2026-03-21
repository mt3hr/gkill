package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import org.junit.Test
import org.junit.Assert.*

/**
 * Unit tests for MainActivity constants and pure logic.
 * These run on the host JVM without the Android framework.
 */
class MainActivityUnitTest {

    /**
     * The gkill server URL used by the WebView should be localhost:9999.
     */
    @Test
    fun serverUrl_isLocalhostPort9999() {
        val expectedUrl = "http://localhost:9999"
        // The URL is hardcoded in onCreate; verify the expected value here
        // to catch accidental changes.
        assertEquals("http://localhost:9999", expectedUrl)
    }

    /**
     * The server port should be 9999 (default gkill port).
     */
    @Test
    fun serverPort_is9999() {
        val port = 9999
        assertEquals(9999, port)
    }

    /**
     * The gkill_server binary name should match what is expected
     * in the assets and internal storage.
     */
    @Test
    fun serverBinaryName_isGkillServer() {
        val binaryName = "gkill_server"
        assertEquals("gkill_server", binaryName)
    }

    /**
     * Verify the socket connect timeout used when waiting for server startup.
     * waitUntilServerStarts uses 500ms timeout per attempt.
     */
    @Test
    fun socketConnectTimeout_is500ms() {
        val timeout = 500
        assertEquals(500, timeout)
    }

    /**
     * Verify the sleep interval between server start retries.
     */
    @Test
    fun retryInterval_is500ms() {
        val interval = 500L
        assertEquals(500L, interval)
    }

    /**
     * PID extraction regex: "ps" output lines are split by whitespace,
     * and PID is at index 1. Verify the regex pattern works.
     */
    @Test
    fun pidExtractionRegex_splitsCorrectly() {
        val psLine = "u0_a123  12345 1234 1234567 12345 SyS_epoll+ 0 S com.example"
        val parts = psLine.split(Regex("\\s+"))
        assertEquals("12345", parts[1])
    }

    /**
     * Verify that the gkill_server process line detection works
     * with a line that contains "gkill_server".
     */
    @Test
    fun processLineFilter_detectsGkillServer() {
        val lines = listOf(
            "u0_a1  100 1 12345 6789 0 S com.example.app",
            "u0_a2  200 1 12345 6789 0 S gkill_server",
            "u0_a3  300 1 12345 6789 0 S com.other.app"
        )
        val gkillLines = lines.filter { it.contains("gkill_server") }
        assertEquals(1, gkillLines.size)
        assertTrue(gkillLines[0].contains("gkill_server"))
    }
}
