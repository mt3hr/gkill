package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.os.Bundle
import android.util.Log
import android.webkit.WebResourceError
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import java.io.File
import java.io.IOException
import java.net.HttpURLConnection
import java.net.InetSocketAddress
import java.net.Socket
import java.net.URL

class MainActivity : AppCompatActivity() {
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
        val gkillBinary = copyServerBinary()

        Log.i("gkill", "バイナリパス: ${gkillBinary.absolutePath}")
        Log.i("gkill", "バイナリサイズ: ${gkillBinary.length()} bytes")
        Log.i("gkill", "実行可能: ${gkillBinary.canExecute()}")
        Log.i("gkill", "読み取り可能: ${gkillBinary.canRead()}")

        Thread {
            try {
                val homeDir = filesDir.parentFile?.absolutePath ?: filesDir.absolutePath
                Log.i("gkill", "HOME: $homeDir")

                // nativeLibraryDirからの実行を試みる（W^X制限回避）
                val nativeDir = applicationInfo.nativeLibraryDir
                Log.i("gkill", "nativeLibraryDir: $nativeDir")

                val pb = ProcessBuilder(gkillBinary.absolutePath)
                pb.environment()["HOME"] = homeDir
                pb.redirectErrorStream(true)
                val process = pb.start()

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
        // killExistingGkillServer()
        startGkillServer() // アプリ再起動時も呼ばれるように
        waitUntilServerStarts {
            setContentView(R.layout.activity_main)
            val webView = findViewById<WebView>(R.id.webview)
            webView.settings.javaScriptEnabled = true
            webView.webViewClient = object : WebViewClient() {
                override fun shouldOverrideUrlLoading(view: WebView?, url: String?): Boolean {
                    view?.loadUrl(url ?: "")
                    return true  // 外部に飛ばさずWebView内で処理
                }
            }

            val gkillURL = "http://localhost:9999"
            webView.loadUrl(gkillURL)
        }
    }


    private fun killExistingGkillServer() {
        try {
            val process = Runtime.getRuntime().exec("ps")
            process.inputStream.bufferedReader().useLines { lines ->
                lines.filter { it.contains("gkill_server") }.forEach { line ->
                    val pid = line.split(Regex("\\s+"))[1] // PIDは2番目の列にある
                    Runtime.getRuntime().exec("kill $pid")
                    Log.d("gkill", "Killed existing gkill_server with pid $pid")
                }
            }
        } catch (e: Exception) {
            Log.w("gkill", "既存プロセスkill失敗", e)
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