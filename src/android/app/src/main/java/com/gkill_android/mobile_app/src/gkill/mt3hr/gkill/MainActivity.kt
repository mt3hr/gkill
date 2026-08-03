package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.Manifest
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
    private var detectedServerUrl = "http://localhost:9999"

    companion object {
        private const val STORAGE_PERMISSION_REQUEST = 1001

        /**
         * 以前のデータ置き場。共有ストレージなので、全ファイルアクセス権を持つ他アプリ、
         * USB/MTP接続、ファイラーアプリのいずれからも中身が読めてしまっていた。
         */
        private const val LEGACY_GKILL_HOME = "/sdcard/gkill"
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

                val pb = ProcessBuilder(
                    gkillBinary.absolutePath,
                    "--gkill_home_dir", gkillHomeDir.absolutePath,
                    "--disable_tls",
                    "--log", "debug"
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

        checkPermissionAndStart()
    }

    private fun checkPermissionAndStart() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            // Android 11+ は MANAGE_EXTERNAL_STORAGE が必要
            if (Environment.isExternalStorageManager()) {
                startServerAndOpen()
            } else {
                @Suppress("DEPRECATION")
                startActivityForResult(
                    Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION).apply {
                        data = Uri.parse("package:$packageName")
                    },
                    STORAGE_PERMISSION_REQUEST
                )
            }
        } else {
            // Android 10 以下は WRITE_EXTERNAL_STORAGE で足りる
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.WRITE_EXTERNAL_STORAGE) == PackageManager.PERMISSION_GRANTED) {
                startServerAndOpen()
            } else {
                ActivityCompat.requestPermissions(
                    this,
                    arrayOf(Manifest.permission.WRITE_EXTERNAL_STORAGE),
                    STORAGE_PERMISSION_REQUEST
                )
            }
        }
    }

    @Deprecated("Deprecated in Java")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        @Suppress("DEPRECATION")
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode == STORAGE_PERMISSION_REQUEST) {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R && Environment.isExternalStorageManager()) {
                startServerAndOpen()
            } else {
                Toast.makeText(this, "ストレージアクセス権限が必要です", Toast.LENGTH_LONG).show()
            }
        }
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == STORAGE_PERMISSION_REQUEST) {
            if (grantResults.isNotEmpty() && grantResults[0] == PackageManager.PERMISSION_GRANTED) {
                startServerAndOpen()
            } else {
                Toast.makeText(this, "ストレージアクセス権限が必要です", Toast.LENGTH_LONG).show()
            }
        }
    }

    private fun startServerAndOpen() {
        serverUrlLatch = CountDownLatch(1)
        detectedServerUrl = "http://localhost:9999"
        killExistingGkillServer()
        startGkillServer()
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

    private fun waitUntilServerStarts(onReady: (String) -> Unit) {
        Thread {
            // stdoutからURLを受け取るまで最大10秒待つ
            serverUrlLatch.await(10, TimeUnit.SECONDS)

            // URLからポートを取得 (例: "http://localhost:9999" → 9999)
            val port = try {
                URI(detectedServerUrl).port.let { if (it == -1) 9999 else it }
            } catch (_: Exception) {
                9999
            }

            var connected = false
            for (i in 1..60) { // 最大30秒待つ（500ms × 60）
                try {
                    val socket = Socket()
                    socket.connect(InetSocketAddress("localhost", port), 500)
                    socket.close()
                    connected = true
                    break
                } catch (_: Exception) {
                    Thread.sleep(500)
                }
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
