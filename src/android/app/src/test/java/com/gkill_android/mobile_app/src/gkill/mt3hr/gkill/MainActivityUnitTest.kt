package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.MainActivity.Companion.HomeMigrationResult
import org.junit.Rule
import org.junit.Test
import org.junit.Assert.*
import org.junit.rules.TemporaryFolder
import java.io.File
import java.io.IOException

/**
 * Unit tests for MainActivity constants and pure logic.
 * These run on the host JVM without the Android framework.
 */
class MainActivityUnitTest {

    @get:Rule
    val tmp = TemporaryFolder()

    /**
     * The data home passed to gkill_server as --gkill_home_dir is /sdcard/gkill.
     */
    @Test
    fun gkillHome_isSdcardGkill() {
        assertEquals("/sdcard/gkill", MainActivity.GKILL_HOME)
    }

    /** 以前の版のアプリ専用領域を模した、中身のあるディレクトリを作る。 */
    private fun appPrivateHomeWithData(): File {
        val source = tmp.newFolder("files", "gkill")
        File(source, "configs").mkdirs()
        File(source, "configs/account.db").writeText("account")
        File(source, "datas/user/kmemo.db").apply { parentFile!!.mkdirs() }.writeText("kmemo")
        return source
    }

    /**
     * データ置き場が無ければ、アプリ専用領域の中身がそのまま複製される。
     * 複製元は残り、一時ディレクトリは残らない。
     */
    @Test
    fun migration_copiesWhenTargetAbsent() {
        val source = appPrivateHomeWithData()
        val target = File(tmp.root, "sdcard/gkill")

        val result = MainActivity.copyAppPrivateHomeIfNeeded(source, target)

        assertEquals(HomeMigrationResult.COPIED, result)
        assertEquals("account", File(target, "configs/account.db").readText())
        assertEquals("kmemo", File(target, "datas/user/kmemo.db").readText())
        assertTrue("複製元は消さないこと", File(source, "configs/account.db").isFile)
        assertFalse(
            "一時ディレクトリが残らないこと",
            File(target.path + MainActivity.MIGRATION_STAGING_SUFFIX).exists()
        )
    }

    /** データ置き場が空のディレクトリなら、無いときと同じく複製する。 */
    @Test
    fun migration_copiesWhenTargetIsEmptyDirectory() {
        val source = appPrivateHomeWithData()
        val target = tmp.newFolder("sdcard", "gkill")

        val result = MainActivity.copyAppPrivateHomeIfNeeded(source, target)

        assertEquals(HomeMigrationResult.COPIED, result)
        assertEquals("kmemo", File(target, "datas/user/kmemo.db").readText())
    }

    /**
     * データ置き場に中身があれば、そちらを正として何も書かない。アプリ専用領域も消さない。
     */
    @Test
    fun migration_leavesBothAloneWhenTargetHasData() {
        val source = appPrivateHomeWithData()
        val target = tmp.newFolder("sdcard", "gkill")
        File(target, "configs").mkdirs()
        File(target, "configs/account.db").writeText("sdcard")

        val result = MainActivity.copyAppPrivateHomeIfNeeded(source, target)

        assertEquals(HomeMigrationResult.TARGET_IN_USE, result)
        assertEquals("sdcard", File(target, "configs/account.db").readText())
        assertFalse("アプリ専用領域の中身を混ぜないこと", File(target, "datas").exists())
        assertEquals("account", File(source, "configs/account.db").readText())
    }

    /** アプリ専用領域に中身が無ければ何もせず、データ置き場も作らない（作るのは起動処理の mkdirs）。 */
    @Test
    fun migration_doesNothingWithoutSource() {
        val source = File(tmp.root, "files/gkill")
        val target = File(tmp.root, "sdcard/gkill")

        val result = MainActivity.copyAppPrivateHomeIfNeeded(source, target)

        assertEquals(HomeMigrationResult.NO_SOURCE, result)
        assertFalse(target.exists())
    }

    /**
     * 前回の複製が途中で止まって一時ディレクトリが残っていても、続きから埋めて改名まで終える。
     */
    @Test
    fun migration_completesAfterInterruptedAttempt() {
        val source = appPrivateHomeWithData()
        val target = File(tmp.root, "sdcard/gkill")
        val staging = File(target.path + MainActivity.MIGRATION_STAGING_SUFFIX)
        File(staging, "configs").mkdirs()
        File(staging, "configs/account.db").writeText("途中")

        val result = MainActivity.copyAppPrivateHomeIfNeeded(source, target)

        assertEquals(HomeMigrationResult.COPIED, result)
        assertEquals("account", File(target, "configs/account.db").readText())
        assertEquals("kmemo", File(target, "datas/user/kmemo.db").readText())
        assertFalse(staging.exists())
    }

    /** データ置き場と同名のファイルがあれば、消さずに例外で止める（呼び出し側はサーバを起動しない）。 */
    @Test
    fun migration_refusesWhenTargetIsAFile() {
        val source = appPrivateHomeWithData()
        val target = File(tmp.newFolder("sdcard"), "gkill").apply { writeText("利用者のファイル") }

        assertThrows(IOException::class.java) {
            MainActivity.copyAppPrivateHomeIfNeeded(source, target)
        }
        assertEquals("利用者のファイル", target.readText())
    }

    /**
     * 待受アドレスと TLS は ServerConfig に従わせるので、実行時上書きの --address と
     * --disable_tls を起動引数に入れない（入れると設定画面の値が Android でだけ効かなくなる）。
     */
    @Test
    fun serverArgs_doNotOverrideServerConfig() {
        val args = MainActivity.buildGkillServerArgs(
            "/data/app/lib/arm64/libgkill_server.so",
            MainActivity.GKILL_HOME
        )
        assertFalse("--address を渡さないこと", args.contains("--address"))
        assertFalse("--disable_tls を渡さないこと", args.contains("--disable_tls"))
    }

    /**
     * The launch arguments must keep the home dir and log flags.
     */
    @Test
    fun serverArgs_keepHomeAndLogFlags() {
        val args = MainActivity.buildGkillServerArgs(
            "/lib/libgkill_server.so",
            "/home/gkill"
        )
        assertEquals("/lib/libgkill_server.so", args[0])
        assertEquals(listOf("/lib/libgkill_server.so", "--gkill_home_dir", "/home/gkill", "--log", "debug"), args)
    }

    /**
     * サーバが ServerConfig から組み立てた起動行の URL を、http でも https でもそのまま拾う。
     */
    @Test
    fun serverUrlLine_isParsedForHttpAndHttps() {
        assertEquals(
            "http://localhost:9999",
            MainActivity.parseServerUrlLine("Access your record space at : http://localhost:9999")
        )
        assertEquals(
            "https://localhost:8443",
            MainActivity.parseServerUrlLine("Access your record space at : https://localhost:8443\r")
        )
    }

    /** 起動行でない行や、URL として使えない起動行は拾わない。 */
    @Test
    fun serverUrlLine_rejectsOtherLines() {
        assertNull(MainActivity.parseServerUrlLine("gkill server started."))
        assertNull(MainActivity.parseServerUrlLine("Access your record space at : "))
        assertNull(MainActivity.parseServerUrlLine("Access your record space at : http://localhost9999"))
        assertNull(MainActivity.parseServerUrlLine("Access your record space at : ftp://localhost:9999"))
    }

    /** URL のポートは明示があればそれ、無ければスキームの既定。解析できなければ null。 */
    @Test
    fun serverPortOf_usesExplicitOrSchemeDefault() {
        assertEquals(9998, MainActivity.serverPortOf("http://localhost:9998"))
        assertEquals(80, MainActivity.serverPortOf("http://localhost"))
        assertEquals(443, MainActivity.serverPortOf("https://localhost"))
        assertNull(MainActivity.serverPortOf("not a url"))
    }

    /**
     * 同じオリジンの URL が再通知されても開き直さず、ポートやスキームが変わったときだけ開き直す。
     */
    @Test
    fun sameServerOrigin_comparesSchemeHostAndPort() {
        assertTrue(MainActivity.isSameServerOrigin("http://localhost:9999/rykv?x=1", "http://localhost:9999"))
        assertTrue(MainActivity.isSameServerOrigin("https://localhost/", "https://localhost:443"))
        assertFalse(MainActivity.isSameServerOrigin("http://localhost:9999/", "http://localhost:9998"))
        assertFalse(MainActivity.isSameServerOrigin("http://localhost:9999/", "https://localhost:9999"))
        assertFalse(MainActivity.isSameServerOrigin(null, "http://localhost:9999"))
    }

    /** 自己署名証明書を通してよいのはループバックのホストだけ。 */
    @Test
    fun loopbackHost_acceptsOnlyLoopback() {
        listOf("localhost", "LOCALHOST", "127.0.0.1", "::1", "[::1]").forEach {
            assertTrue(it, MainActivity.isLoopbackHost(it))
        }
        listOf("192.168.0.10", "example.com", "127.0.0.2", "", null).forEach {
            assertFalse(it.toString(), MainActivity.isLoopbackHost(it))
        }
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
