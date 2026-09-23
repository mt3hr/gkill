package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.Manifest
import android.content.ActivityNotFoundException
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.util.Log
import android.view.View
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import java.io.File
import java.io.IOException
import java.net.InetSocketAddress
import java.net.Socket
import java.net.URI
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

class MainActivity : AppCompatActivity() {
    // WebViewはonCreateで一度だけ取得して設定を済ませる。
    // 使う場所ごとにfindViewByIdすると、設定漏れのインスタンスが生まれる。
    private lateinit var webView: WebView

    private var gkillServerProcess: Process? = null
    private var serverUrlLatch = CountDownLatch(1)
    private var detectedServerUrl = DEFAULT_SERVER_URL

    companion object {
        private const val STORAGE_PERMISSION_REQUEST = 1001

        /**
         * gkill のデータ置き場（gkill_server の --gkill_home_dir）。
         *
         * ここには全Kyouのデータベースに加えて、パスワードハッシュとリセットトークンを持つ
         * アカウントDB、ログ、TLSの秘密鍵が入る。共有ストレージなので、全ファイルアクセス権を
         * 持つ他アプリ、USB/MTP接続、ファイラーアプリのいずれからも中身が読める。
         * マニフェストの allowBackup=false もこの置き場には効かない。
         *
         * 2026-08-03 にアプリ専用領域（filesDir/gkill）へ移したが、2026-09-24 にここへ戻した。
         * 共有ストレージへ書くので、権限が許可されるまでサーバを起動しない。
         */
        const val GKILL_HOME = "/sdcard/gkill"

        /** 2026-08-03〜2026-09-24 の版がデータを置いていたアプリ専用領域（filesDir 配下）の名前。 */
        const val APP_PRIVATE_HOME_NAME = "gkill"

        /** アプリ専用領域から [GKILL_HOME] へ複製するときの一時ディレクトリの接尾辞。 */
        const val MIGRATION_STAGING_SUFFIX = ".migrating"

        /** 既定のサーバ待受ポート。 */
        const val DEFAULT_SERVER_PORT = 9999

        /** WebView が最初に読み込む既定URL。stdout からURLを検出するまでのフォールバックでもある。 */
        const val DEFAULT_SERVER_URL = "http://localhost:9999"

        /**
         * gkill_server をループバックに限定して待ち受けさせるアドレス。
         *
         * これは実行時オーバーライド（--address）であって設定DBは書き換えない。
         * 全インターフェース待受をやめ、同一LANの別端末から :9999 へ到達できないようにする。
         * stdout からのURL検出（http://localhost:9999）はポートが同じなので無傷。
         */
        const val SERVER_LISTEN_ADDRESS = "127.0.0.1:9999"

        /** 既存サーバの応答を確かめる先行プローブの接続タイムアウト(ms)。 */
        const val PROBE_TIMEOUT_MS = 300

        /** 起動待ちループでの1回ぶんのソケット接続タイムアウト(ms)。 */
        const val SERVER_CONNECT_TIMEOUT_MS = 500

        /** 起動待ちループのリトライ間隔(ms)。 */
        const val RETRY_INTERVAL_MS = 500L

        /**
         * gkill_server の起動引数を組み立てる。
         *
         * companion に切り出しているのはユニットテストから引数（--address 127.0.0.1:9999 を
         * 含むこと）を検証できるようにするため。
         */
        fun buildGkillServerArgs(binaryPath: String, gkillHomePath: String): List<String> =
            listOf(
                binaryPath,
                "--gkill_home_dir", gkillHomePath,
                "--address", SERVER_LISTEN_ADDRESS,
                "--disable_tls",
                "--log", "debug"
            )

        /** [copyAppPrivateHomeIfNeeded] の結果。 */
        enum class HomeMigrationResult {
            /** アプリ専用領域に中身が無い。何もしていない。 */
            NO_SOURCE,

            /** [GKILL_HOME] に中身がある。そちらを正として何もしていない（アプリ専用領域は残したまま）。 */
            TARGET_IN_USE,

            /** アプリ専用領域の中身を [GKILL_HOME] へ複製した（複製元は残したまま）。 */
            COPIED
        }

        /**
         * アプリ専用領域にデータを置いていた版から更新した端末のために、
         * [target]（[GKILL_HOME]）が無いか空のときだけ [source] の中身を複製する。
         *
         * [target] に中身があればそちらを正として何もしない。その版へ移行したときの
         * 古い複製が [target] に残っている端末では、専用領域にしか無い記録が見えなくなるが、
         * [target] を正とする方針なので上書きも退避もしない。複製元はどの場合も消さない。
         *
         * 複製は一時ディレクトリへ行い、終わってから [target] へ改名する。直接複製すると、
         * 途中で失敗した中途半端な [target] が次回の起動で「中身あり」と見なされ、
         * 以後二度と複製されずに正として使われてしまう。一時ディレクトリが前回の失敗で
         * 残っていても、複製元は変わっていないので上書きで続きから埋めればよい。
         *
         * companion に置いているのは android.* に触れずユニットテストから検証するため。
         * 失敗は例外で返す。呼び出し側は、失敗したままサーバを起動してはいけない
         * （gkill_server が空の [target] を作り、次回から TARGET_IN_USE になる）。
         */
        fun copyAppPrivateHomeIfNeeded(source: File, target: File): HomeMigrationResult {
            if (!source.hasEntries()) return HomeMigrationResult.NO_SOURCE
            if (target.hasEntries()) return HomeMigrationResult.TARGET_IN_USE
            if (target.exists() && !target.isDirectory) {
                // 同名のファイルがある。利用者のものかもしれないので消さずに止める
                throw IOException("データ置き場と同名のファイルがある")
            }
            val staging = File(target.path + MIGRATION_STAGING_SUFFIX)
            source.copyRecursively(staging, overwrite = true)
            // ここに来る target は空のディレクトリか存在しないかのどちらか。
            // delete() は空でないディレクトリを消さないので、中身を巻き込むことはない
            if (target.exists() && !target.delete()) {
                throw IOException("空のデータ置き場を片付けられなかった")
            }
            if (!staging.renameTo(target)) {
                throw IOException("複製したデータをデータ置き場へ改名できなかった")
            }
            return HomeMigrationResult.COPIED
        }

        private fun File.hasEntries(): Boolean = isDirectory && (list()?.isNotEmpty() == true)
    }

    /**
     * gkill_server の起動を始めたか。onResume と権限要求の結果の両方から
     * [startServerWhenStorageAccessible] が呼ばれるので、二重起動を防ぐ。
     */
    private var serverStartRequested = false

    /** この Activity で権限の要求画面を自動で一度出したか。 */
    private var storageAccessRequested = false

    /**
     * アプリ専用領域に以前の版のデータがあれば [GKILL_HOME] へ複製する。
     *
     * 失敗したら false を返す。このときサーバを起動してはいけない（[copyAppPrivateHomeIfNeeded]）。
     * ログにパスと例外の文言を載せないのは F-008 の方針（環境情報を logcat へ出さない）に合わせるため。
     */
    private fun migrateAppPrivateHome(target: File): Boolean {
        return try {
            when (copyAppPrivateHomeIfNeeded(File(filesDir, APP_PRIVATE_HOME_NAME), target)) {
                HomeMigrationResult.NO_SOURCE -> Unit
                HomeMigrationResult.TARGET_IN_USE -> Log.w(
                    "gkill",
                    "アプリ専用領域に以前の版のデータが残っているが、データ置き場に中身があるのでそちらを使う"
                )
                HomeMigrationResult.COPIED -> Log.i(
                    "gkill",
                    "アプリ専用領域のデータをデータ置き場へ複製した（複製元は残してある）"
                )
            }
            true
        } catch (e: Exception) {
            Log.e("gkill", "アプリ専用領域からデータ置き場への複製に失敗した (${e.javaClass.simpleName})")
            false
        }
    }

    /**
     * gkill_server は jniLibs に libgkill_server.so として同梱し、
     * nativeLibraryDir から直接実行する。
     * targetSdk 29以降、アプリのデータディレクトリ配下のファイルは
     * W^X 制約により execve() できないため、実行可能な nativeLibraryDir を使う。
     */
    private fun serverBinary(): File =
        File(applicationInfo.nativeLibraryDir, "libgkill_server.so")

    private fun startGkillServer() {
        Thread {
            try {
                val gkillBinary = serverBinary()

                // 起動診断は値の要らないものだけ残す。絶対パス(バイナリ・HOME・GKILL_HOME・
                // nativeLibraryDir)は logcat へ出さない — Log.d でも release 実行時に出力され、
                // 端末ログ・クラッシュ収集・adb 越しに環境情報が漏れる(指摘 F-008)。
                Log.d(
                    "gkill",
                    "バイナリ診断: size=${gkillBinary.length()} bytes, " +
                        "exec=${gkillBinary.canExecute()}, read=${gkillBinary.canRead()}"
                )

                val homeDir = filesDir.parentFile?.absolutePath ?: filesDir.absolutePath
                val gkillHomeDir = File(GKILL_HOME)

                if (!migrateAppPrivateHome(gkillHomeDir)) {
                    runOnUiThread {
                        Toast.makeText(
                            this,
                            "以前の版のデータを $GKILL_HOME へ複製できなかったため、起動を止めました",
                            Toast.LENGTH_LONG
                        ).show()
                    }
                    return@Thread
                }
                gkillHomeDir.mkdirs()

                // 起動引数は companion の buildGkillServerArgs に切り出してある（テストから検証するため）。
                // --address 127.0.0.1:9999 はループバック限定の実行時オーバーライドで、
                // 設定DBは書き換えず、stdout からのURL検出(http://localhost:9999)も無傷。
                val pb = ProcessBuilder(
                    buildGkillServerArgs(gkillBinary.absolutePath, gkillHomeDir.absolutePath)
                )
                pb.environment()["HOME"] = homeDir
                pb.redirectErrorStream(true)
                val process = pb.start()
                gkillServerProcess = process

                // stdoutを別スレッドで読み続ける（バッファフルによるハング防止）
                // サーバーURLを "Access your record space at : " 行から検出する。
                // 全行を logcat へ中継しない — サーバ出力には環境情報が混ざりうるうえ、
                // Log.d でも release 実行時に出力される(指摘 F-008)。
                // 異常終了の診断用に直近の行だけメモリに保持し、exitCode != 0 のときに出す。
                val recentServerLines = ArrayDeque<String>()
                Thread {
                    try {
                        process.inputStream.bufferedReader().forEachLine { line ->
                            synchronized(recentServerLines) {
                                recentServerLines.addLast(line)
                                if (recentServerLines.size > 20) recentServerLines.removeFirst()
                            }
                            val prefix = "Access your record space at : "
                            if (line.startsWith(prefix)) {
                                detectedServerUrl = line.removePrefix(prefix).trim()
                                Log.i("gkill", "サーバーURL検出: $detectedServerUrl")
                                serverUrlLatch.countDown()
                            }
                        }
                    } catch (e: Exception) {
                        // 無言で握ると、ここが死んだときにサーバURLの検出も止まるのに
                        // 何も残らない（画面は真っ白のまま待ち続ける）。
                        Log.e("gkill", "gkill_server の標準出力の読み取りが止まった", e)
                    }
                }.start()

                val exitCode = process.waitFor()
                // 正常終了(0)まで ERROR で出していた。すぐ下の Toast は exitCode で
                // 出し分けているのに、ログだけ揃っていなかった。
                if (exitCode != 0) {
                    Log.e("gkill", "プロセス終了コード: $exitCode")
                    val tail = synchronized(recentServerLines) { recentServerLines.joinToString("\n") }
                    if (tail.isNotEmpty()) {
                        Log.e("gkill_server_stdout", tail)
                    }
                } else {
                    Log.i("gkill", "gkill_server が正常終了した")
                }
                runOnUiThread {
                    if (exitCode != 0) {
                        Toast.makeText(this, "gkill_server 異常終了 (code=$exitCode)", Toast.LENGTH_LONG).show()
                    }
                }
            } catch (e: Exception) {
                runOnUiThread {
                    Toast.makeText(this, "gkill_server 起動失敗：${e.message}", Toast.LENGTH_LONG).show()
                    Log.e("gkill", "起動失敗", e)
                }
            }
        }.start()
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        // targetSdk 35以降は edge-to-edge が強制されるため、
        // システムバーぶんの余白を自前で確保して従来の見た目を維持する
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(android.R.id.content)) { v, insets ->
            val bars = insets.getInsets(WindowInsetsCompat.Type.systemBars())
            v.setPadding(bars.left, bars.top, bars.right, bars.bottom)
            insets
        }

        webView = findViewById(R.id.webview)
        webView.visibility = View.GONE
        // gkillのクライアントはVue SPAなのでJavaScriptは必須
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        // 読み込むのは自前サーバ(127.0.0.1)だけなので、
        // content:// と file:// 経由の他アプリ・ローカルファイルへのアクセスは塞ぐ
        webView.settings.allowContentAccess = false
        webView.settings.allowFileAccess = false
        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(view: WebView, request: WebResourceRequest): Boolean {
                val url = request.url
                if (isLocalServerUrl(url)) {
                    view.loadUrl(url.toString())
                    return true  // 外部に飛ばさずWebView内で処理
                }
                // 自前サーバ以外へのリンクはWebView内で開かず、端末のブラウザに任せる
                return try {
                    startActivity(Intent(Intent.ACTION_VIEW, url))
                    true
                } catch (e: Exception) {
                    // URL と例外本体は載せない(ActivityNotFoundException の文言に URL が入る。
                    // 利用者が開いたブックマーク先が端末ログへ残る。指摘 F-008)
                    Log.w("gkill", "外部URLを開けませんでした (${e.javaClass.simpleName})")
                    true
                }
            }
        }

        // 戻るボタン: WebView に履歴があれば1つ戻り、無ければ既定動作（アプリ終了）へ委譲する。
        // configChanges で回転を自前処理するようにしたので、Activity が作り直されず
        // WebView の履歴が保持され、ここでの canGoBack が意味を持つ。
        onBackPressedDispatcher.addCallback(this, object : OnBackPressedCallback(true) {
            override fun handleOnBackPressed() {
                if (webView.canGoBack()) {
                    webView.goBack()
                } else {
                    // 自分を無効化してから既定のバック処理を呼び直す（そのまま終了へ流す）
                    isEnabled = false
                    onBackPressedDispatcher.onBackPressed()
                }
            }
        })

        findViewById<View>(R.id.grant_storage_access_button).setOnClickListener {
            requestSharedStorageAccess()
        }

        // 起動は onResume の startServerWhenStorageAccessible が行う
        // （設定画面で権限を許可して戻ってきたときの再判定も同じ入口にするため）
    }

    override fun onResume() {
        super.onResume()
        startServerWhenStorageAccessible()
    }

    /**
     * [GKILL_HOME]（共有ストレージ）へ書ける権限があればサーバを起動する。無ければ起動しない。
     *
     * 権限なしで起動すると gkill_server がデータ置き場を作れずに落ちる。
     * 権限の要求画面は最初の1回だけ自動で出し、以後は画面のボタンから出し直す。
     * onResume のたびに出すと、設定画面から許可せずに戻るたびに設定画面へ送り返され、
     * アプリから抜けられなくなる。
     */
    private fun startServerWhenStorageAccessible() {
        if (serverStartRequested) return
        if (hasSharedStorageAccess()) {
            serverStartRequested = true
            showStorageAccessMissing(false)
            startServerAndOpen()
            return
        }
        showStorageAccessMissing(true)
        if (!storageAccessRequested) {
            storageAccessRequested = true
            requestSharedStorageAccess()
        }
    }

    /**
     * 共有ストレージへ書けるか。
     * Android 11+ は MANAGE_EXTERNAL_STORAGE（全ファイルアクセス）、10 以下は WRITE_EXTERNAL_STORAGE。
     * Go サーバは SAF の content:// を扱えず実パスでしか読み書きできないので、細かい権限では足りない。
     */
    private fun hasSharedStorageAccess(): Boolean =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            Environment.isExternalStorageManager()
        } else {
            ContextCompat.checkSelfPermission(
                this,
                Manifest.permission.WRITE_EXTERNAL_STORAGE
            ) == PackageManager.PERMISSION_GRANTED
        }

    /**
     * 共有ストレージへの権限を求める。許可されたかどうかの判定は、Android 11+ は設定画面から
     * 戻ったときの onResume、10 以下は onRequestPermissionsResult で行う。
     */
    private fun requestSharedStorageAccess() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.R) {
            ActivityCompat.requestPermissions(
                this,
                arrayOf(Manifest.permission.WRITE_EXTERNAL_STORAGE),
                STORAGE_PERMISSION_REQUEST
            )
            return
        }
        try {
            startActivity(
                Intent(
                    Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION,
                    Uri.parse("package:$packageName")
                )
            )
        } catch (_: ActivityNotFoundException) {
            // アプリ単位の画面を持たない端末がある。アプリ一覧の画面へ落とす
            try {
                startActivity(Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION))
            } catch (e: ActivityNotFoundException) {
                Log.w("gkill", "全ファイルアクセスの設定画面を開けなかった (${e.javaClass.simpleName})")
                Toast.makeText(
                    this,
                    "端末の設定で gkill に「すべてのファイルへのアクセス」を許可してください",
                    Toast.LENGTH_LONG
                ).show()
            }
        }
    }

    /** 起動待ちの画面を、権限待ち（説明文と許可ボタン）と起動中（スピナー）で切り替える。 */
    private fun showStorageAccessMissing(missing: Boolean) {
        val (whenMissing, whenStarting) = if (missing) View.VISIBLE to View.GONE else View.GONE to View.VISIBLE
        findViewById<View>(R.id.loading_progress).visibility = whenStarting
        findViewById<View>(R.id.loading_message).visibility = whenStarting
        findViewById<View>(R.id.storage_access_message).visibility = whenMissing
        findViewById<View>(R.id.grant_storage_access_button).visibility = whenMissing
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == STORAGE_PERMISSION_REQUEST) {
            startServerWhenStorageAccessible()
        }
    }

    private fun startServerAndOpen() {
        serverUrlLatch = CountDownLatch(1)
        detectedServerUrl = DEFAULT_SERVER_URL
        // ポート先行プローブ: 既に応答する gkill_server があれば kill/start を丸ごと飛ばす。
        // 画面回転などで Activity が作り直されても、生きているサーバを殺して立て直さないため。
        // ソケット接続はメインスレッドで行えないので別スレッドに逃がす。
        Thread {
            if (isPortOpen(DEFAULT_SERVER_PORT, PROBE_TIMEOUT_MS)) {
                Log.i("gkill", "既存の gkill_server が応答したので起動処理を省略する")
                // waitUntilServerStarts が 10 秒待たずに進めるよう、URL検出ラッチを即座に開ける
                serverUrlLatch.countDown()
            } else {
                // 応答が無いときだけ従来経路。死にかけプロセスの回収も兼ねる。
                killExistingGkillServer()
                startGkillServer()
            }
        }.start()
        waitUntilServerStarts { url ->
            findViewById<View>(R.id.loading_layout).visibility = View.GONE
            // onCreateで設定済みだが、読み込み直前にも明示しておく。
            // 読み込むのは自前サーバだけなので content:// と file:// は塞ぐ。
            // applyでまとめるとレシーバがラムダ経由になり静的解析が追えないため、
            // onCreate側と同じくwebViewを明示的に書く
            webView.settings.allowContentAccess = false
            webView.settings.allowFileAccess = false
            webView.visibility = View.VISIBLE
            webView.loadUrl(url)
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        gkillServerProcess?.destroy()
        gkillServerProcess = null
    }

    /**
     * WebView内で開いてよいURLかどうか。
     * このアプリが表示するのは同梱のgkill_serverだけなので、ループバックに限定する。
     */
    private fun isLocalServerUrl(url: Uri): Boolean {
        val scheme = url.scheme?.lowercase()
        if (scheme != "http" && scheme != "https") {
            return false
        }
        return when (url.host?.lowercase()) {
            "localhost", "127.0.0.1", "::1", "[::1]" -> true
            else -> false
        }
    }

    private fun killExistingGkillServer() {
        try {
            gkillServerProcess?.destroy()
            gkillServerProcess = null
        } catch (e: Exception) {
            Log.w("gkill", "保存プロセスkill失敗", e)
        }
        try {
            // PATHを差し替えられても別バイナリが動かないよう絶対パスで叩く
            val ps = Runtime.getRuntime().exec(arrayOf("/system/bin/ps", "-A"))
            ps.inputStream.bufferedReader().useLines { lines ->
                lines.filter { it.contains("gkill_server") }.forEach { line ->
                    val parts = line.trim().split(Regex("\\s+"))
                    if (parts.size >= 2) {
                        val pid = parts[1]
                        Runtime.getRuntime().exec(arrayOf("/system/bin/kill", "-9", pid)).waitFor()
                        Log.d("gkill", "Killed gkill_server pid=$pid")
                    }
                }
            }
        } catch (e: Exception) {
            Log.w("gkill", "ps-based kill失敗", e)
        }
    }

    /**
     * ループバックの指定ポートに接続できるか（＝サーバが応答するか）を1回だけ確かめる。
     */
    private fun isPortOpen(port: Int, timeoutMs: Int): Boolean {
        return try {
            Socket().use { socket ->
                socket.connect(InetSocketAddress("localhost", port), timeoutMs)
            }
            true
        } catch (_: Exception) {
            false
        }
    }

    private fun waitUntilServerStarts(onReady: (String) -> Unit) {
        Thread {
            // stdoutからURLを受け取るまで最大10秒待つ
            serverUrlLatch.await(10, TimeUnit.SECONDS)

            // URLからポートを取得 (例: "http://localhost:9999" → 9999)
            val port = try {
                URI(detectedServerUrl).port.let { if (it == -1) DEFAULT_SERVER_PORT else it }
            } catch (_: Exception) {
                DEFAULT_SERVER_PORT
            }

            var connected = false
            for (i in 1..60) { // 最大30秒待つ（500ms × 60）
                if (isPortOpen(port, SERVER_CONNECT_TIMEOUT_MS)) {
                    connected = true
                    break
                }
                Thread.sleep(RETRY_INTERVAL_MS)
            }

            if (connected) {
                runOnUiThread { onReady(detectedServerUrl) }
            } else {
                runOnUiThread {
                    Toast.makeText(this, "gkill_server 起動に失敗", Toast.LENGTH_LONG).show()
                }
            }
        }.start()
    }
}
