package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import android.os.Bundle
import android.util.Log
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import java.io.File
import java.io.IOException

class MainActivity : AppCompatActivity() {
    private fun copyServerBinary(): File {
        val input = assets.open("gkill_server")
        val outputFile = File(filesDir, "gkill_server")  // ← 修正ポイント

        input.use { inStream ->
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
        startGkillServer()
        Thread.sleep(10000) // 応急措置：1秒待ってからWebViewロード（改善可能）

        setContentView(R.layout.activity_main)
        val webView = findViewById<WebView>(R.id.webview)
        webView.settings.javaScriptEnabled = true
        webView.loadUrl("http://127.0.0.1:9999")
        webView.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(view: WebView?, url: String?): Boolean {
                view?.loadUrl(url ?: "")
                return true  // 外部に飛ばさずWebView内で処理
            }
        }
    }
}