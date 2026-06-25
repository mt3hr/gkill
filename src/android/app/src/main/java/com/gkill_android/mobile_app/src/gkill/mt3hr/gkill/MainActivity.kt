package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.os.Bundle
import android.util.Log
import android.view.View
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import java.io.File
import java.net.InetSocketAddress
import java.net.Socket

class MainActivity : AppCompatActivity() {
    private var gkillServerProcess: Process? = null

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

                val nativeDir = applicationInfo.nativeLibraryDir
                Log.i("gkill", "nativeLibraryDir: $nativeDir")

                val pb = ProcessBuilder(gkillBinary.absolutePath, "--disable_tls", "--log", "debug")
                pb.environment()["HOME"] = homeDir
                pb.redirectErrorStream(true)
                val process = pb.start()
                gkillServerProcess = process

                // stdoutを別スレッドで読み続ける（バッファフルによるハング防止）
                Thread {
                    try {
                        process.inputStream.bufferedReader().forEachLine {
                            Log.d("gkill_server_stdout", it)
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
        webView.setOnLongClickListener { true }
        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(view: WebView?, url: String?): Boolean {
                view?.loadUrl(url ?: "")
                return true  // 外部に飛ばさずWebView内で処理
            }
        }

        killExistingGkillServer()
        startGkillServer()
        waitUntilServerStarts {
            findViewById<View>(R.id.loading_layout).visibility = View.GONE
            webView.visibility = View.VISIBLE
            webView.loadUrl("http://localhost:9999")
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

    fun waitUntilServerStarts(onReady: () -> Unit) {
        Thread {
            var connected = false
            for (i in 1..60) { // 最大30秒待つ（500ms * 60）
                try {
                    val socket = Socket()
                    socket.connect(InetSocketAddress("localhost", 9999), 500)
                    socket.close()
                    connected = true
                    break
                } catch (_: Exception) {
                    Thread.sleep(500)
                }
            }

            if (connected) {
                runOnUiThread { onReady() }
            } else {
                runOnUiThread {
                    Toast.makeText(this, "gkill_server 起動に失敗", Toast.LENGTH_LONG).show()
                }
            }
        }.start()
    }
}
