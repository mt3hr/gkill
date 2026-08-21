package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.Environment
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
         * 以前のデータ置き場。共有ストレージなので、全ファイルアクセス権を持つ他アプリ、
         * USB/MTP接続、ファイラーアプリのいずれからも中身が読めてしまっていた。
         */
        private const val LEGACY_GKILL_HOME = "/sdcard/gkill"

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
    }

    /**
     * gkillのデータ置き場。アプリ専用領域に置く。
     *
     * ここには全Kyouのデータベースに加えて、パスワードハッシュとリセットトークンを持つ
     * アカウントDB、ログ、TLSの秘密鍵が入る。以前は /sdcard/gkill だったため、
     * 端末内の他アプリから丸ごと読める状態だった。
     * マニフェストの allowBackup=false は、データがアプリ専用領域にあって初めて意味を持つ。
     *
     * なお外部ストレージ権限は引き続き必要。写真などを指すファイルリポジトリは、
     * 利用者が選んだ共有ストレージ上のディレクトリを参照するため。
     */
    private val gkillHome: File
        get() = File(filesDir, "gkill")

    /**
     * 以前 /sdcard/gkill に置いていたデータをアプリ専用領域へ複製する。
     *
     * 複製元は消さない。移行が途中で失敗しても元データが残るようにするためで、
     * 動作を確認できたら利用者自身に削除してもらう想定。
     * 消すまでは古い方が共有ストレージに残り続けるので、ログで警告する。
     */
    private fun migrateLegacyHomeIfNeeded(target: File) {
        val legacy = File(LEGACY_GKILL_HOME)
        if (!legacy.isDirectory) return
        // 移行済み(中身がある)なら触らない
        if (target.isDirectory && (target.list()?.isNotEmpty() == true)) return

        Log.i("gkill", "旧データ置き場から移行する: ${legacy.absolutePath} -> ${target.absolutePath}")
        try {
            legacy.copyRecursively(target, overwrite = false)
            Log.w(
                "gkill",
                "移行が完了した。${legacy.absolutePath} は共有ストレージに残っているので、" +
                    "動作を確認したら削除すること"
            )
        } catch (e: Exception) {
            Log.e("gkill", "旧データ置き場からの移行に失敗した: ${e.message}", e)
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

                Log.i("gkill", "バイナリパス: ${gkillBinary.absolutePath}")
                Log.i("gkill", "バイナリサイズ: ${gkillBinary.length()} bytes")
                Log.i("gkill", "実行可能: ${gkillBinary.canExecute()}")
                Log.i("gkill", "読み取り可能: ${gkillBinary.canRead()}")

                val homeDir = filesDir.parentFile?.absolutePath ?: filesDir.absolutePath
                val gkillHomeDir = gkillHome
                Log.i("gkill", "HOME: $homeDir")
                Log.i("gkill", "GKILL_HOME: ${gkillHomeDir.absolutePath}")

                val nativeDir = applicationInfo.nativeLibraryDir
                Log.i("gkill", "nativeLibraryDir: $nativeDir")

                migrateLegacyHomeIfNeeded(gkillHomeDir)
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
                // サーバーURLを "Access your record space at : " 行から検出する
                Thread {
                    try {
                        process.inputStream.bufferedReader().forEachLine { line ->
                            Log.d("gkill_server_stdout", line)
                            val prefix = "Access your record space at : "
                            if (line.startsWith(prefix)) {
                                detectedServerUrl = line.removePrefix(prefix).trim()
                                Log.i("gkill", "サーバーURL検出: $detectedServerUrl")
                                serverUrlLatch.countDown()
                            }
                        }
                    } catch (_: Exception) {}
                }.start()

                val exitCode = process.waitFor()
                Log.e("gkill", "プロセス終了コード: $exitCode")
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
                    Log.w("gkill", "外部URLを開けませんでした: $url", e)
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

        checkPermissionAndStart()
    }

    /**
     * サーバを起動する。権限の有無で起動をブロックしない。
     *
     * filesDir 配下のデータだけで全機能が動くので、共有ストレージ上のファイルリポジトリを
     * 開くための全ファイルアクセス権限が無くても起動する。権限の案内は非ブロッキングに出す。
     */
    private fun checkPermissionAndStart() {
        startServerAndOpen()
        guideAllFilesAccessIfNeeded()
    }

    /**
     * 全ファイルアクセス権限が無ければ非ブロッキングに案内する。
     *
     * Android 11+ は MANAGE_EXTERNAL_STORAGE（設定画面での付与）を Toast で案内するだけで、
     * 設定画面を自動で開いて起動を妨げることはしない。
     * Android 10 以下は従来どおり WRITE_EXTERNAL_STORAGE のランタイム権限を要求するが、
     * サーバは既に起動しているので拒否されても filesDir 配下の全機能は動く。
     */
    private fun guideAllFilesAccessIfNeeded() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            if (!Environment.isExternalStorageManager()) {
                Toast.makeText(
                    this,
                    "共有ストレージ上のファイルを扱うには、設定から全ファイルアクセスを許可してください",
                    Toast.LENGTH_LONG
                ).show()
            }
        } else if (ContextCompat.checkSelfPermission(
                this,
                Manifest.permission.WRITE_EXTERNAL_STORAGE
            ) != PackageManager.PERMISSION_GRANTED
        ) {
            ActivityCompat.requestPermissions(
                this,
                arrayOf(Manifest.permission.WRITE_EXTERNAL_STORAGE),
                STORAGE_PERMISSION_REQUEST
            )
        }
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        // サーバは権限に関わらず起動済み。ここでは案内だけ行い、再起動はしない。
        if (requestCode == STORAGE_PERMISSION_REQUEST &&
            (grantResults.isEmpty() || grantResults[0] != PackageManager.PERMISSION_GRANTED)
        ) {
            Toast.makeText(
                this,
                "共有ストレージ上のファイルを扱うにはストレージアクセスの許可が必要です",
                Toast.LENGTH_LONG
            ).show()
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
