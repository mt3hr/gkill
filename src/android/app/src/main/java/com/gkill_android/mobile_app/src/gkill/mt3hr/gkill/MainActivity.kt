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
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import java.io.File
import java.net.InetSocketAddress
import java.net.Socket
import java.net.URI
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit

class MainActivity : AppCompatActivity() {
    private var gkillServerProcess: Process? = null
    private var serverUrlLatch = CountDownLatch(1)
    private var detectedServerUrl = "http://localhost:9999"

    companion object {
        private const val STORAGE_PERMISSION_REQUEST = 1001
        private const val GKILL_HOME = "/sdcard/gkill"
    }

    private fun copyServerBinary(): File {
        val outputFile = File(filesDir, "gkill_server")

        // バージョン更新時にバイナリを上書きするため、毎回コピーする
        assets.open("gkill_server").use { inStream ->
            outputFile.outputStream().use { outStream ->
                inStream.copyTo(outStream)
            }
        }
        outputFile.setReadable(true, true)
        outputFile.setExecutable(true, true)

        return outputFile
    }

    private fun startGkillServer() {
        Thread {
            try {
                val gkillBinary = copyServerBinary()

                Log.i("gkill", "バイナリパス: ${gkillBinary.absolutePath}")
                Log.i("gkill", "バイナリサイズ: ${gkillBinary.length()} bytes")
                Log.i("gkill", "実行可能: ${gkillBinary.canExecute()}")
                Log.i("gkill", "読み取り可能: ${gkillBinary.canRead()}")

                val homeDir = filesDir.parentFile?.absolutePath ?: filesDir.absolutePath
                Log.i("gkill", "HOME: $homeDir")
                Log.i("gkill", "GKILL_HOME: $GKILL_HOME")

                val nativeDir = applicationInfo.nativeLibraryDir
                Log.i("gkill", "nativeLibraryDir: $nativeDir")

                File(GKILL_HOME).mkdirs()

                val pb = ProcessBuilder(
                    gkillBinary.absolutePath,
                    "--gkill_home_dir", GKILL_HOME,
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

        val webView = findViewById<WebView>(R.id.webview)
        webView.visibility = View.GONE
        webView.settings.javaScriptEnabled = true
        webView.settings.domStorageEnabled = true
        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(view: WebView?, url: String?): Boolean {
                view?.loadUrl(url ?: "")
                return true  // 外部に飛ばさずWebView内で処理
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
            findViewById<WebView>(R.id.webview).apply {
                visibility = View.VISIBLE
                loadUrl(url)
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        gkillServerProcess?.destroy()
        gkillServerProcess = null
    }

    private fun killExistingGkillServer() {
        try {
            gkillServerProcess?.destroy()
            gkillServerProcess = null
        } catch (e: Exception) {
            Log.w("gkill", "保存プロセスkill失敗", e)
        }
        try {
            val ps = Runtime.getRuntime().exec("ps -A")
            ps.inputStream.bufferedReader().useLines { lines ->
                lines.filter { it.contains("gkill_server") }.forEach { line ->
                    val parts = line.trim().split(Regex("\\s+"))
                    if (parts.size >= 2) {
                        val pid = parts[1]
                        Runtime.getRuntime().exec(arrayOf("kill", "-9", pid)).waitFor()
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
