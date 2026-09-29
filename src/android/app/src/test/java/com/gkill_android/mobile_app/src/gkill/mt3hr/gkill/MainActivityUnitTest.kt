package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.MainActivity.Companion.HomeMigrationResult
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.MainActivity.Companion.StorageGateDecision
import org.junit.Rule
import org.junit.Test
import org.junit.Assert.*
import org.junit.rules.TemporaryFolder
import java.io.File
import java.io.IOException

/**
 * Unit tests for MainActivity constants and pure logic.
 * These run on the host JVM without the Android framework.
 *
 * Activity 側の配線（判定関数を呼んで結果どおりに動くこと）は JVM から実行できないので、
 * MainActivity.kt とリソースの本文を文字列で検査する。判定関数だけを検証すると、呼び出し側を
 * 書き換えても（例: onReceivedSslError を無条件の proceed() にする）ビルドもテストも通ってしまう。
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
     * 証明書のエラーを通すのは、ループバックの同梱サーバ（https・ポート付き・パスやクエリ付き）だけ。
     * WebView が渡す URL はページ本体のほか、サブリソースのパスやクエリを持つ。
     */
    @Test
    fun sslError_proceedsForLoopbackServer() {
        listOf(
            "https://localhost:8443",
            "https://localhost:8443/",
            "https://LOCALHOST:8443/rykv?x=1#top",
            "https://127.0.0.1:8443/api/get_kyous",
            "https://[::1]:8443/",
            "https://localhost/",
            // クエリに URI が受け付けない文字があっても、ホストで判定できること
            "https://localhost:8443/rykv?q={\"a\":1}"
        ).forEach {
            assertTrue(it, MainActivity.shouldProceedOnSslError(it))
        }
    }

    /**
     * 外部のサイト・LAN のサーバ・ホストを取り出せない URL は止める。
     * 止め忘れると、外部サイトの偽の証明書をこのアプリだけが黙って受け入れる。
     *
     * 判定が正しくても、WebViewClient の onReceivedSslError が判定を呼ばずに proceed() すれば
     * 全部通ってしまう（ビルドは通る）。本文が「判定が true なら proceed、それ以外は cancel」だけで
     * あることも確かめる。
     */
    @Test
    fun sslError_cancelsForNonLoopbackOrUnparsableUrl() {
        assertEquals(
            "onReceivedSslError は判定を呼び、false なら cancel すること",
            "if (shouldProceedOnSslError(error.url)) { handler.proceed() } else { handler.cancel() }",
            blockAfter(mainActivityCode(), "override fun onReceivedSslError(")
        )

        listOf(
            "https://example.com/",
            "https://192.168.0.10:8443/",
            "https://127.0.0.2:8443/",
            "https://localhost.example.com/",
            "https://user@example.com:8443/",
            "https://localhost@example.com:8443/",
            "https://[fe80::1]:8443/",
            "localhost:8443",
            "not a url",
            ""
        ).forEach {
            assertFalse(it, MainActivity.shouldProceedOnSslError(it))
        }
        assertFalse("null", MainActivity.shouldProceedOnSslError(null))
    }

    /**
     * 権限の判断: 起動済みなら何もしない・権限があれば起動・無ければ要求は最初の1回だけで、
     * 2回目以降は権限待ちの画面にするだけ（onResume のたびに要求画面を出すと設定画面から抜けられない）。
     *
     * 「1回だけ」は判断表だけでは守れない。startServerWhenStorageAccessible が要求したことを
     * storageAccessRequested に残さないと、次の onResume でも REQUEST_ACCESS になって設定画面へ
     * 送り返される（ビルドは通る）。判断に渡す値と、要求画面を出す分岐の本文も確かめる。
     */
    @Test
    fun storageGate_startsOnlyWithAccessAndRequestsOnce() {
        val gate = blockAfter(mainActivityCode(), "private fun startServerWhenStorageAccessible()")
        assertTrue(
            "判断には Activity の2つのフラグと実際の権限の有無を渡すこと: $gate",
            gate.contains(
                "decideStorageGate( serverStartRequested = serverStartRequested, " +
                    "hasStorageAccess = hasSharedStorageAccess(), accessRequested = storageAccessRequested )"
            )
        )
        val request = blockAfter(gate, "StorageGateDecision.REQUEST_ACCESS ->")
        assertTrue("要求画面を出す分岐で要求したことを残すこと: $request", request.contains("storageAccessRequested = true"))
        assertTrue("要求画面を出す分岐で要求すること: $request", request.contains("requestSharedStorageAccess()"))
        assertEquals(
            "要求画面を出すのは REQUEST_ACCESS の分岐だけ: $gate",
            1,
            Regex("""\brequestSharedStorageAccess\(\)""").findAll(gate).count()
        )

        assertEquals(
            StorageGateDecision.START_SERVER,
            MainActivity.decideStorageGate(serverStartRequested = false, hasStorageAccess = true, accessRequested = false)
        )
        assertEquals(
            "権限を許可して戻ってきたときも起動する",
            StorageGateDecision.START_SERVER,
            MainActivity.decideStorageGate(serverStartRequested = false, hasStorageAccess = true, accessRequested = true)
        )
        assertEquals(
            StorageGateDecision.REQUEST_ACCESS,
            MainActivity.decideStorageGate(serverStartRequested = false, hasStorageAccess = false, accessRequested = false)
        )
        assertEquals(
            "自動の要求は1回だけ",
            StorageGateDecision.WAIT_FOR_ACCESS,
            MainActivity.decideStorageGate(serverStartRequested = false, hasStorageAccess = false, accessRequested = true)
        )
    }

    /**
     * 起動を始めた後は、権限の有無や要求の有無にかかわらず二度と起動しない（onResume と権限結果の両方から呼ばれる）。
     * 起動する分岐が serverStartRequested を立てないと、判断表が正しくても毎回起動し直す（ビルドは通る）。
     */
    @Test
    fun storageGate_neverStartsTwice() {
        val gate = blockAfter(mainActivityCode(), "private fun startServerWhenStorageAccessible()")
        val start = blockAfter(gate, "StorageGateDecision.START_SERVER ->")
        assertTrue("起動する分岐で起動したことを残すこと: $start", start.contains("serverStartRequested = true"))
        assertTrue("起動する分岐で起動すること: $start", start.contains("startServerAndOpen()"))
        assertEquals(
            "起動するのは START_SERVER の分岐だけ: $gate",
            1,
            Regex("""\bstartServerAndOpen\(\)""").findAll(gate).count()
        )
        assertTrue("起動済みなら何もしないこと: $gate", gate.contains("StorageGateDecision.ALREADY_STARTED -> Unit"))


        for (hasAccess in listOf(true, false)) {
            for (requested in listOf(true, false)) {
                assertEquals(
                    "hasAccess=$hasAccess requested=$requested",
                    StorageGateDecision.ALREADY_STARTED,
                    MainActivity.decideStorageGate(
                        serverStartRequested = true,
                        hasStorageAccess = hasAccess,
                        accessRequested = requested
                    )
                )
            }
        }
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
     * `ps -A` の行（USER PID PPID VSZ RSS WCHAN ADDR S NAME）から PID は2列目を取る。
     * 先頭の空白やタブ区切りでも同じ。列が足りない行や空行からは取らない（kill に渡す値が無い）。
     * killExistingGkillServer が kill に渡すのがこの関数の結果であることも確かめる（手書きの split に
     * 戻すと、この検証が本番の動きを見なくなる）。
     */
    @Test
    fun psLinePid_isSecondColumn() {
        val kill = blockAfter(mainActivityCode(), "private fun killExistingGkillServer()")
        assertTrue("PID は parsePsLinePid で取ること: $kill", kill.contains("parsePsLinePid(line)?.let { pid ->"))
        assertTrue("kill に渡すのはその PID: $kill", kill.contains("""arrayOf("/system/bin/kill", "-9", pid)"""))
        assertFalse("行を手書きで切らないこと: $kill", kill.contains(".split("))

        assertEquals("12345", MainActivity.parsePsLinePid("u0_a123  12345 1234 1234567 12345 SyS_epoll+ 0 S libgkill_server"))
        assertEquals("200", MainActivity.parsePsLinePid("  u0_a2\t200 1 12345 6789 0 S libgkill_server"))
        assertNull(MainActivity.parsePsLinePid("u0_a2"))
        assertNull(MainActivity.parsePsLinePid(""))
        assertNull(MainActivity.parsePsLinePid("   "))
    }

    /**
     * 殺すのは gkill_server の行だけ。このアプリ自身（com.mt3hr.gkill）や他のアプリの行を拾わない。
     *
     * 同梱の実体は libgkill_server.so で、起動時の argv[0] はその絶対パス。実機の ps の出力はリポジトリで
     * 確かめていないので、形を1つに決めずに拾えることを見る: argv[0] 由来の列なら libgkill_server.so
     * （ps の実装によってはパス付き）、comm 由来の列なら 15 文字に切られた libgkill_server。
     * どの形でも gkill_server を含む。
     *
     * killExistingGkillServer がこの関数で行を選んでいることも確かめる。判定が正しくても、呼び出し側を
     * 別の条件（例: "com." を含む行）に替えると、このアプリ自身を kill -9 する（ビルドは通る）。
     */
    @Test
    fun processLine_detectsOnlyGkillServer() {
        val kill = blockAfter(mainActivityCode(), "private fun killExistingGkillServer()")
        assertTrue(
            "殺す行は isGkillServerProcessLine で選ぶこと: $kill",
            kill.contains("lines.filter { isGkillServerProcessLine(it) }")
        )
        assertFalse("行を手書きの contains で選ばないこと: $kill", kill.contains(".contains("))

        listOf(
            "u0_a2  200 1 12345 6789 0 S libgkill_server",
            "u0_a2  200 1 12345 6789 0 S libgkill_server.so",
            "u0_a2  200 1 12345 6789 0 S gkill_server",
            "u0_a2  200 1 12345 6789 0 S /data/app/lib/arm64/libgkill_server.so"
        ).forEach {
            assertTrue(it, MainActivity.isGkillServerProcessLine(it))
        }
        listOf(
            "u0_a1  100 1 12345 6789 0 S com.example.app",
            "u0_a2  150 1 12345 6789 0 S com.mt3hr.gkill",
            "USER           PID  PPID     VSZ    RSS WCHAN            ADDR S NAME",
            ""
        ).forEach {
            assertFalse(it, MainActivity.isGkillServerProcessLine(it))
        }
    }

    /**
     * モジュール（src/android/app）からの相対パスでファイルを読む。
     * Gradle のユニットテストは作業ディレクトリがモジュールのディレクトリになる。
     */
    private fun moduleFile(path: String): File {
        val file = File(path)
        assertTrue("見つからない: ${file.absolutePath}", file.isFile)
        return file
    }

    private val mainActivityPath = "src/main/java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/MainActivity.kt"

    /**
     * コメントを除いた MainActivity.kt の本文。配線の検査がコメントの文言に引っかからないようにする。
     * 行コメントは行頭か空白の後の `//` だけを落とす（文字列中の `://` を巻き込まないため）。
     */
    private fun mainActivityCode(): String =
        moduleFile(mainActivityPath).readText()
            .replace(Regex("""/\*.*?\*/""", RegexOption.DOT_MATCHES_ALL), "")
            .replace(Regex("""(?m)(^|\s)//.*$"""), "")

    /**
     * [text] で最初に現れる [header] の直後の `{ ... }` の中身を、空白を1つに詰めて返す。
     * 終わりは波括弧の対応で決めるので、入れ子のブロックや `${...}` を含んでも途中で切れない。
     */
    private fun blockAfter(text: String, header: String): String {
        val at = text.indexOf(header)
        assertTrue("見つからない: $header", at >= 0)
        val open = text.indexOf('{', at + header.length)
        assertTrue("$header の後に { が無い", open >= 0)
        var depth = 0
        for (i in open until text.length) {
            when (text[i]) {
                '{' -> depth++
                '}' -> {
                    depth--
                    if (depth == 0) return text.substring(open + 1, i).replace(Regex("""\s+"""), " ").trim()
                }
            }
        }
        throw AssertionError("$header の本文の終わりが見つからない")
    }

    /**
     * res の layout で始まるディレクトリ（layout / layout-sw600dp など）にある activity_main.xml をすべて集める。
     * 置き場を手で列挙すると、後から足した画面幅や向きのレイアウトが検査から黙って漏れる。
     */
    private fun activityMainLayouts(): List<File> {
        val layouts = File("src/main/res")
            .listFiles { dir -> dir.isDirectory && dir.name.startsWith("layout") }
            .orEmpty()
            .map { File(it, "activity_main.xml") }
            .filter { it.isFile }
            .sortedBy { it.path }
        // スマホ用とタブレット用の2つは必ずある。集め損ねて素通りしないように
        assertTrue("activity_main.xml が2つ以上見つからない: $layouts", layouts.size >= 2)
        return layouts
    }

    /**
     * ステータスバーの色 gkill_indigo は Web のテーマ色 primary（ライト・ダークとも）と同じ。
     * 片方だけ変えると、アプリバーとステータスバーの色が黙ってずれる。
     */
    @Test
    fun statusBarColor_matchesWebPrimary() {
        val colors = moduleFile("src/main/res/values/colors.xml").readText()
        val indigo = Regex("""<color name="gkill_indigo">#([0-9A-Fa-f]{6})</color>""")
            .find(colors)?.groupValues?.get(1)?.lowercase()
        assertNotNull("colors.xml に gkill_indigo が無い", indigo)

        val vuetify = moduleFile("../../client/plugins/vuetify.ts").readText()
        val primaries = Regex("""\bprimary: '#([0-9A-Fa-f]{6})'""").findAll(vuetify)
            .map { it.groupValues[1].lowercase() }.toList()
        // ライトとダークの2テーマぶん拾えていること（0件で素通りしないように）
        assertEquals(2, primaries.size)
        primaries.forEach { assertEquals(indigo, it) }
    }

    /**
     * ステータスバーを gkill_indigo で塗る設定が、昼夜のテーマと全てのレイアウト（スマホ用・タブレット用など）にそろっている。
     * API 34 以下はテーマの android:statusBarColor、API 35 以上はレイアウトの帯 status_bar_background が塗る。
     * 夜間用のテーマは丸ごと差し替わり、タブレット用のレイアウトは別ファイルなので、片方だけ直すと黙って外れる。
     */
    @Test
    fun statusBar_isPaintedWithGkillIndigoInAllThemesAndLayouts() {
        for (path in listOf("src/main/res/values/themes.xml", "src/main/res/values-night/themes.xml")) {
            val theme = moduleFile(path).readText()
            assertTrue(path, theme.contains("""<item name="android:statusBarColor">@color/gkill_indigo</item>"""))
            assertTrue(path, theme.contains("""<item name="android:windowLightStatusBar">false</item>"""))
        }
        for (file in activityMainLayouts()) {
            val path = file.path
            val layout = file.readText()
            val band = Regex("""<View\s[^>]*android:id="@\+id/status_bar_background"[^>]*/>""").find(layout)?.value
            assertNotNull("$path に status_bar_background が無い", band)
            assertTrue(path, band!!.contains("""android:background="@color/gkill_indigo""""))
            assertTrue(path, layout.contains("""android:id="@+id/root_layout""""))
            assertTrue(path, layout.contains("""android:id="@+id/content_container""""))
        }
    }

    /**
     * 共有ストレージへ実パスで書くための宣言が AndroidManifest.xml から落ちていない。
     * MANAGE_EXTERNAL_STORAGE（Android 11 以降の全ファイルアクセス）と requestLegacyExternalStorage
     * （Android 10 でだけ効く）は、どちらが欠けても対応する版の端末で /sdcard/gkill へ書けず、
     * 権限を許可しても gkill_server がデータ置き場を作れずに落ちる（ビルドは通る）。
     * Android 8〜10（API 26〜29。minSdk は 26）は WRITE_EXTERNAL_STORAGE の許可で起動する
     * （MainActivity.hasSharedStorageAccess）。宣言が無いか maxSdkVersion が 29 より小さいと、その版では
     * requestPermissions が画面を出さずに拒否で返り、サーバが永久に起動しない。
     * コメントアウトした宣言を拾わないように、XML のコメントを除いてから見る。
     */
    @Test
    fun manifest_declaresSharedStorageAccessForAllSupportedVersions() {
        val manifest = moduleFile("src/main/AndroidManifest.xml").readText()
            .replace(Regex("""<!--.*?-->""", RegexOption.DOT_MATCHES_ALL), "")

        val writeLegacy = Regex("""<uses-permission\b[^>]*>""").findAll(manifest).map { it.value }
            .firstOrNull { it.contains("""android:name="android.permission.WRITE_EXTERNAL_STORAGE"""") }
        assertNotNull("WRITE_EXTERNAL_STORAGE の uses-permission が無い", writeLegacy)
        val writeMaxSdk = Regex("""android:maxSdkVersion="(\d+)"""").find(writeLegacy!!)?.groupValues?.get(1)?.toInt()
        assertTrue(
            "WRITE_EXTERNAL_STORAGE の maxSdkVersion は無しか 29 以上（Android 10 まで使う）: $writeMaxSdk",
            writeMaxSdk == null || writeMaxSdk >= 29
        )

        val manageAll = Regex("""<uses-permission\s+android:name="android\.permission\.MANAGE_EXTERNAL_STORAGE"([^>]*)/>""")
            .find(manifest)
        assertNotNull("MANAGE_EXTERNAL_STORAGE の uses-permission が無い", manageAll)
        assertFalse(
            "MANAGE_EXTERNAL_STORAGE に maxSdkVersion を付けると、それより新しい端末で黙って外れる",
            manageAll!!.groupValues[1].contains("maxSdkVersion")
        )

        val application = Regex("""<application\s[^>]*>""").find(manifest)?.value
        assertNotNull("application 要素が無い", application)
        assertTrue(
            "application に requestLegacyExternalStorage=\"true\" が無い",
            application!!.contains("""android:requestLegacyExternalStorage="true"""")
        )
    }

    /**
     * MainActivity.kt が R.id で引く全ての ID が、全てのレイアウト（スマホ用・タブレット用など）にある。
     * レイアウトは別ファイルなので片方だけ足し忘れてもビルドは通り、その画面幅の端末でだけ
     * findViewById が null を返して起動時に落ちる。ID の一覧は MainActivity.kt の本文から
     * 正規表現で集める（手で列挙すると、後から足した ID が検査から漏れる）。
     */
    @Test
    fun layouts_defineEveryIdMainActivityLooksUp() {
        val referenced = Regex("""\bR\.id\.(\w+)""").findAll(mainActivityCode()).map { it.groupValues[1] }.toSet()
        // 正規表現が空振りして素通りしないように、確実に引いている ID を1つ確かめる
        assertTrue("MainActivity.kt から R.id.webview を拾えていない", referenced.contains("webview"))

        for (file in activityMainLayouts()) {
            val layout = file.readText()
            val defined = Regex("""android:id="@\+id/(\w+)"""").findAll(layout).map { it.groupValues[1] }.toSet()
            val missing = referenced - defined
            assertTrue("${file.path} に無い ID: $missing", missing.isEmpty())
        }
    }
}
