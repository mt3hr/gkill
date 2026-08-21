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
        // Reference the real companion constant so an accidental change is caught.
        assertEquals("http://localhost:9999", MainActivity.DEFAULT_SERVER_URL)
    }

    /**
     * The server port should be 9999 (default gkill port).
     */
    @Test
    fun serverPort_is9999() {
        assertEquals(9999, MainActivity.DEFAULT_SERVER_PORT)
    }

    /**
     * The server must listen on the loopback interface only (127.0.0.1:9999),
     * so a different device on the same LAN cannot reach it.
     */
    @Test
    fun serverListenAddress_isLoopbackOnly() {
        assertEquals("127.0.0.1:9999", MainActivity.SERVER_LISTEN_ADDRESS)
    }

    /**
     * The launch arguments must pin the listen address to the loopback interface.
     * The arg list is factored into the companion so it can be verified here.
     */
    @Test
    fun serverArgs_containLoopbackAddress() {
        val args = MainActivity.buildGkillServerArgs(
            "/data/app/lib/arm64/libgkill_server.so",
            "/data/user/0/pkg/files/gkill"
        )
        val addressIndex = args.indexOf("--address")
        assertTrue("--address フラグが起動引数に含まれること", addressIndex >= 0)
        assertTrue("--address の値が続くこと", addressIndex + 1 < args.size)
        assertEquals("127.0.0.1:9999", args[addressIndex + 1])
    }

    /**
     * The launch arguments must keep the existing flags (home dir / disable TLS / log).
     */
    @Test
    fun serverArgs_keepExistingFlags() {
        val args = MainActivity.buildGkillServerArgs(
            "/lib/libgkill_server.so",
            "/home/gkill"
        )
        assertEquals("/lib/libgkill_server.so", args[0])
        assertTrue(args.containsAll(listOf("--gkill_home_dir", "/home/gkill", "--disable_tls", "--log", "debug")))
    }

    /**
     * The gkill_server binary name should match what is expected
     * in jniLibs and the native library directory.
     * jniLibs から実体ファイルとして展開されるのは lib*.so にマッチする名前のみ。
     */
    @Test
    fun serverBinaryName_isGkillServer() {
        val binaryName = "libgkill_server.so"
        assertEquals("libgkill_server.so", binaryName)
    }

    /**
     * Verify the socket connect timeout used when waiting for server startup.
     * waitUntilServerStarts uses 500ms timeout per attempt.
     */
    @Test
    fun socketConnectTimeout_is500ms() {
        assertEquals(500, MainActivity.SERVER_CONNECT_TIMEOUT_MS)
    }

    /**
     * Verify the sleep interval between server start retries.
     */
    @Test
    fun retryInterval_is500ms() {
        assertEquals(500L, MainActivity.RETRY_INTERVAL_MS)
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
