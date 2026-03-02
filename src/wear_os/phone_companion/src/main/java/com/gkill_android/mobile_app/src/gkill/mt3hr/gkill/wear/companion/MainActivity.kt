package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.os.Bundle
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.security.MessageDigest

/**
 * Settings screen for the Phone Companion app.
 * Users enter the gkill server URL, user ID, and password here.
 * Also provides a button to push the watch APK to the paired Pixel Watch 2.
 */
class MainActivity : AppCompatActivity() {

    private lateinit var store: GkillCredentialStore

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        store = GkillCredentialStore(this)

        val dp = resources.displayMetrics.density
        val p16 = (16 * dp).toInt()
        val lp = android.widget.LinearLayout.LayoutParams(
            android.widget.LinearLayout.LayoutParams.MATCH_PARENT,
            android.widget.LinearLayout.LayoutParams.WRAP_CONTENT
        )

        val layout = android.widget.LinearLayout(this).apply {
            orientation = android.widget.LinearLayout.VERTICAL
            setPadding(p16, p16, p16, p16)
        }

        val tvTitle = TextView(this).apply {
            text = "gkill Wear 設定"
            textSize = 20f
            setPadding(0, 0, 0, p16)
            layoutParams = lp
        }

        val etServerUrl = EditText(this).apply {
            hint = "サーバーURL (例: http://localhost:9999)"
            setText(store.getServerUrl())
            layoutParams = lp
        }
        val etUserId = EditText(this).apply {
            hint = "ユーザーID"
            setText(store.getUserId())
            layoutParams = lp
        }
        val etPassword = EditText(this).apply {
            hint = "パスワード"
            inputType = android.text.InputType.TYPE_CLASS_TEXT or
                    android.text.InputType.TYPE_TEXT_VARIATION_PASSWORD
            layoutParams = lp
        }
        val btnSave = Button(this).apply {
            text = "保存 & 接続テスト"
            layoutParams = lp
        }
        val tvStatus = TextView(this).apply {
            text = ""
            layoutParams = lp
        }

        layout.addView(tvTitle)
        layout.addView(etServerUrl)
        layout.addView(etUserId)
        layout.addView(etPassword)
        layout.addView(btnSave)
        layout.addView(tvStatus)

        val scrollView = android.widget.ScrollView(this).apply {
            addView(layout)
        }
        setContentView(scrollView)

        btnSave.setOnClickListener {
            val serverUrl = etServerUrl.text.toString().trimEnd('/')
            val userId = etUserId.text.toString()
            val password = etPassword.text.toString()

            if (serverUrl.isEmpty() || userId.isEmpty() || password.isEmpty()) {
                Toast.makeText(this, "全項目を入力してください", Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }

            val passwordSha256 = sha256(password)
            store.setServerUrl(serverUrl)
            store.setUserId(userId)
            store.setPasswordSha256(passwordSha256)
            store.clearSession()

            tvStatus.text = "接続テスト中..."

            CoroutineScope(Dispatchers.IO).launch {
                val client = GkillApiClient(serverUrl)
                val (sessionId, errorMsg) = client.loginWithError(userId, passwordSha256)
                withContext(Dispatchers.Main) {
                    if (sessionId != null) {
                        store.setSessionId(sessionId)
                        tvStatus.text = "接続成功！ セッションID: ${sessionId.take(8)}..."
                    } else {
                        tvStatus.text = "接続失敗: $errorMsg"
                    }
                }
            }
        }

    }

    private fun sha256(input: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
        val hash = digest.digest(input.toByteArray(Charsets.UTF_8))
        return hash.joinToString("") { "%02x".format(it) }
    }
}
