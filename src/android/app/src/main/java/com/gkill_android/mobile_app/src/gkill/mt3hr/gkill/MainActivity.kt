package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.os.Bundle
import android.util.Log
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import java.io.File
import java.io.IOException
import java.net.InetSocketAddress
import java.net.Socket

class MainActivity : AppCompatActivity() {
    private fun copyServerBinary(): File {
        val input = assets.open("gkill_server")
        val outputFile = File(filesDir, "gkill_server")

        if (!outputFile.exists()) {
            input.use { inStream ->
                outputFile.outputStream().use { outStream ->
                    inStream.copyTo(outStream)
                }
            }
            outputFile.setReadable(true, true)
            outputFile.setExecutable(true, true)
        }

        return outputFile
    }

    private fun startGkillServer() {
        val gkillBinary = copyServerBinary()

        Thread {
            try {
                Thread.sleep(1000) // 応急措置
                val homeDir = filesDir.parentFile?.absolutePath ?: filesDir.absolutePath
                val pb = ProcessBuilder(gkillBinary.absolutePath, "--log=true")

                pb.environment()["HOME"] = homeDir
                pb.redirectErrorStream(true)
                val process = pb.start()

                val exitCode = process.waitFor()
                process.inputStream.bufferedReader().forEachLine {
                    Log.d("gkill_server_stdout", it)
                }
                process.errorStream.bufferedReader().forEachLine {
                    Log.e("gkill_server_stderr", it)
                }
                Log.e("gkill", "プロセス終了コード: $exitCode")
            } catch (e: IOException) {
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
            webView.loadUrl("http://localhost:9999")
            webView.webViewClient = object : WebViewClient() {
                override fun shouldOverrideUrlLoading(view: WebView?, url: String?): Boolean {
                    view?.loadUrl(url ?: "")
                    return true  // 外部に飛ばさずWebView内で処理
                }
            }
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
            for (i in 1..20) { // 最大10秒待つ（500ms * 20）
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