package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.Manifest
import android.content.ActivityNotFoundException
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.net.http.SslError
import android.os.Build
import android.os.Bundle
import android.os.Environment
import android.provider.Settings
import android.util.Log
import android.view.View
import android.view.ViewGroup
import android.webkit.SslErrorHandler
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.activity.OnBackPressedCallback
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import androidx.core.content.edit
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsCompat
import androidx.core.view.updateLayoutParams
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

        /**
         * gkill_server が起動するたびに標準出力へ出す、WebView で開くURLの行の接頭辞。
         *
         * URL はサーバが ServerConfig（ENABLE_THIS_DEVICE の行の ADDRESS / ENABLE_TLS）から
         * `<http|https>://localhost:<ポート>` の形で組み立てる（close.go の PrintStartedMessage。
         * デスクトップ版のウィンドウも同じ組み立て方）。ここで拾った URL が画面のアドレスの唯一の出所で、
         * Kotlin 側にポートやスキームを持たない。サーバ設定を保存するとサーバは内部で作り直され、
         * この行をもう一度出す。
         */
        const val SERVER_URL_LINE_PREFIX = "Access your record space at : "

        /** 前回拾ったサーバURLを保存する SharedPreferences の名前とキー。 */
        private const val PREFS_NAME = "gkill_server"
        private const val PREF_LAST_SERVER_URL = "last_server_url"

        /** 既存サーバの応答を確かめる先行プローブの接続タイムアウト(ms)。 */
        const val PROBE_TIMEOUT_MS = 300

        /** 起動待ちループでの1回ぶんのソケット接続タイムアウト(ms)。 */
        const val SERVER_CONNECT_TIMEOUT_MS = 500

        /** 起動待ちループのリトライ間隔(ms)。 */
        const val RETRY_INTERVAL_MS = 500L

        /** 起動待ちループの試行回数（500ms × 60 = 最大30秒）。 */
        const val SERVER_CONNECT_ATTEMPTS = 60

        /** プロセスを起動してから URL の行が出るまで待つ上限(ms)。超えたら知らせる。 */
        const val SERVER_URL_WAIT_MS = 60_000L

        /**
         * gkill_server の起動引数を組み立てる。
         *
         * --address と --disable_tls は渡さない。どちらも設定DBを書き換えない実行時上書きで、
         * 渡すと設定画面の待受アドレスと TLS が Android でだけ効かなくなる（2026-09-24 まで渡していた）。
         * companion に切り出しているのはユニットテストから引数を検証できるようにするため。
         */
        fun buildGkillServerArgs(binaryPath: String, gkillHomePath: String): List<String> =
            listOf(
                binaryPath,
                "--gkill_home_dir", gkillHomePath,
                "--log", "debug"
            )

        /**
         * 標準出力の1行がサーバURLの行なら、その URL を返す。
         * http / https でループバックのホストを指す URL だけを受け付ける。サーバは常に localhost で
         * 知らせるので、それ以外は壊れた行として捨てる（例: ADDRESS をポートだけの "9999" にすると
         * サーバは "http://localhost9999" を出す。これを開くと名前解決に失敗するだけで原因が見えない）。
         */
        fun parseServerUrlLine(line: String): String? {
            if (!line.startsWith(SERVER_URL_LINE_PREFIX)) return null
            val url = line.removePrefix(SERVER_URL_LINE_PREFIX).trim()
            val uri = try {
                URI(url)
            } catch (_: Exception) {
                return null
            }
            val scheme = uri.scheme?.lowercase()
            if (scheme != "http" && scheme != "https") return null
            if (!isLoopbackHost(uri.host)) return null
            return url
        }

        /** URL のポート。明示が無ければスキームの既定（http=80 / https=443）。解析できなければ null。 */
        fun serverPortOf(url: String): Int? {
            val uri = try {
                URI(url)
            } catch (_: Exception) {
                return null
            }
            if (uri.port != -1) return uri.port
            return when (uri.scheme?.lowercase()) {
                "http" -> 80
                "https" -> 443
                else -> null
            }
        }

        /**
         * 2つの URL が同じオリジン（スキーム・ホスト・ポート）か。パスは見ない。
         * WebView が開いているページを、同じサーバのURLが再通知されただけで開き直さないために使う。
         */
        fun isSameServerOrigin(current: String?, target: String): Boolean {
            if (current == null) return false
            val a = try {
                URI(current)
            } catch (_: Exception) {
                return false
            }
            val b = try {
                URI(target)
            } catch (_: Exception) {
                return false
            }
            return a.scheme.equals(b.scheme, ignoreCase = true) &&
                a.host.equals(b.host, ignoreCase = true) &&
                serverPortOf(current) == serverPortOf(target)
        }

        /** 同梱サーバとみなすループバックのホスト名か。 */
        fun isLoopbackHost(host: String?): Boolean =
            when (host?.lowercase()) {
                "localhost", "127.0.0.1", "::1", "[::1]" -> true
                else -> false
            }

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
     * サーバが URL の行を出したときに呼ばれる。起動時のほか、画面の設定からサーバ設定を保存して
     * サーバが内部で作り直されたときにも出るので、待受アドレスや TLS を変えればここで新しい URL が届く。
     *
     * 呼ばれるのは標準出力の読み取りスレッドで、そこは出力を吸い続けなければならない
     * （止めるとバッファが詰まってサーバが止まる）ので、待受の確認は別スレッドで行う。
     */
    private fun onServerUrlAnnounced(url: String, process: Process) {
        Log.i("gkill", "サーバーURL検出: $url")
        saveLastServerUrl(url)
        Thread { openWhenReachable(url, process) }.start()
    }

    /**
     * [url] のポートが応答するまで待ってから WebView で開く。
     *
     * URL の行は待ち受けを始める前に出るので、行を拾っただけでは開けるとは限らない。
     * [process] が終了したら待つのをやめる（待受に失敗して落ちたのに、同じポートの別物を開かないため）。
     * 既に動いているサーバを使い回すときは起動したプロセスが無いので null。
     */
    private fun openWhenReachable(url: String, process: Process?) {
        val port = serverPortOf(url)
        if (port == null) {
            Log.w("gkill", "サーバーURLからポートを取り出せない: $url")
            return
        }
        for (i in 1..SERVER_CONNECT_ATTEMPTS) {
            if (process != null && !process.isAlive) return
            if (isPortOpen(port, SERVER_CONNECT_TIMEOUT_MS)) {
                runOnUiThread { showServerPage(url) }
                return
            }
            Thread.sleep(RETRY_INTERVAL_MS)
        }
        runOnUiThread {
            Toast.makeText(this, "gkill_server 起動に失敗", Toast.LENGTH_LONG).show()
        }
    }

    /**
     * WebView でサーバの画面を出す。今開いているページと同じオリジンなら開き直さない
     * （サーバ設定を保存しただけで同じ URL がもう一度届くため）。
     * オリジンが変わったとき（ポートや http/https を変えたとき）は新しい URL へ移る。
     * オリジンごとに保存領域が分かれるので、その場合はログインし直しになる。
     */
    private fun showServerPage(url: String) {
        if (isFinishing || isDestroyed) return
        findViewById<View>(R.id.loading_layout).visibility = View.GONE
        // onCreateで設定済みだが、読み込み直前にも明示しておく。
        // 読み込むのは自前サーバだけなので content:// と file:// は塞ぐ。
        // applyでまとめるとレシーバがラムダ経由になり静的解析が追えないため、
        // onCreate側と同じくwebViewを明示的に書く
        webView.settings.allowContentAccess = false
        webView.settings.allowFileAccess = false
        webView.visibility = View.VISIBLE
        if (!isSameServerOrigin(webView.url, url)) {
            webView.loadUrl(url)
        }
    }

    /**
     * プロセスを起動してもしばらく URL の行が出ないときに知らせる。
     * 黙っていると画面が読み込み中のまま止まって見える。プロセスが先に終了したときは
     * 終了コードの Toast が出るので、ここでは何もしない。
     */
    private fun warnIfServerUrlNotAnnounced(process: Process, announced: CountDownLatch) {
        Thread {
            if (!announced.await(SERVER_URL_WAIT_MS, TimeUnit.MILLISECONDS) && process.isAlive) {
                Log.w("gkill", "gkill_server を起動してから URL の行が出ないまま ${SERVER_URL_WAIT_MS / 1000} 秒たった")
                runOnUiThread {
                    Toast.makeText(this, "gkill_server の起動を確認できません", Toast.LENGTH_LONG).show()
                }
            }
        }.start()
    }

    /** 前回サーバが出した URL。次の起動で、既に動いているサーバを確かめる先に使う。 */
    private fun loadLastServerUrl(): String? =
        getSharedPreferences(PREFS_NAME, MODE_PRIVATE).getString(PREF_LAST_SERVER_URL, null)

    private fun saveLastServerUrl(url: String) {
        getSharedPreferences(PREFS_NAME, MODE_PRIVATE).edit { putString(PREF_LAST_SERVER_URL, url) }
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
                // 待受アドレスと TLS は ServerConfig に従わせるので、上書きのフラグは渡さない。
                val pb = ProcessBuilder(
                    buildGkillServerArgs(gkillBinary.absolutePath, gkillHomeDir.absolutePath)
                )
                pb.environment()["HOME"] = homeDir
                pb.redirectErrorStream(true)
                val process = pb.start()
                gkillServerProcess = process
                val urlAnnounced = CountDownLatch(1)
                warnIfServerUrlNotAnnounced(process, urlAnnounced)

                // stdoutを別スレッドで読み続ける（バッファフルによるハング防止）
                // サーバーURLを SERVER_URL_LINE_PREFIX の行から検出する。
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
                            parseServerUrlLine(line)?.let { url ->
                                urlAnnounced.countDown()
                                onServerUrlAnnounced(url, process)
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
        // システムバーぶんの余白を自前で確保して従来の見た目を維持する。
        // その環境ではテーマの android:statusBarColor が効かず、ステータスバーには裏の画面が透けるので、
        // ステータスバーの高さぶんの帯（gkill のテーマ色）を敷く。
        // edge-to-edge でない API 34 以下では、ステータスバーの色はテーマの android:statusBarColor が付ける。
        val statusBarBackground = findViewById<View>(R.id.status_bar_background)
        val contentContainer = findViewById<View>(R.id.content_container)
        ViewCompat.setOnApplyWindowInsetsListener(findViewById(R.id.root_layout)) { _, insets ->
            val bars = insets.getInsets(WindowInsetsCompat.Type.systemBars())
            statusBarBackground.updateLayoutParams { height = bars.top }
            contentContainer.updateLayoutParams<ViewGroup.MarginLayoutParams> {
                setMargins(bars.left, bars.top, bars.right, bars.bottom)
            }
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

            override fun onReceivedSslError(view: WebView, handler: SslErrorHandler, error: SslError) {
                // ServerConfig で TLS を有効にすると、同梱サーバは自己署名の証明書で https を返す。
                // ループバックの同梱サーバに限って通す。ループバックへの http はサーバを認証せずに
                // 受け入れているので、ここで証明書を検証しなくても守りは弱くならない。
                // それ以外（外部のサイト）は既定どおり止める。
                if (isLoopbackHost(Uri.parse(error.url).host)) {
                    handler.proceed()
                } else {
                    handler.cancel()
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
        // 先行プローブ: 前回サーバが出した URL のポートが応答すれば kill/start を丸ごと飛ばす。
        // Activity が作り直されても、生きているサーバを殺して立て直さないため。
        // ポートは ServerConfig で変わるので、固定値ではなく前回拾った URL から取る。
        // ソケット接続はメインスレッドで行えないので別スレッドに逃がす。
        Thread {
            val lastUrl = loadLastServerUrl()
            val lastPort = lastUrl?.let { serverPortOf(it) }
            if (lastUrl != null && lastPort != null && isPortOpen(lastPort, PROBE_TIMEOUT_MS)) {
                Log.i("gkill", "既存の gkill_server が応答したので起動処理を省略する")
                openWhenReachable(lastUrl, null)
            } else {
                // 応答が無いときだけ従来経路。死にかけプロセスの回収も兼ねる。
                // URL はサーバが起動行で知らせてくる（onServerUrlAnnounced）。
                killExistingGkillServer()
                startGkillServer()
            }
        }.start()
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
        return isLoopbackHost(url.host)
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
}
