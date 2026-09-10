package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.os.Bundle
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
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
            setText(R.string.settings_title)
            textSize = 20f
            setPadding(0, 0, 0, p16)
            layoutParams = lp
        }

        val etServerUrl = EditText(this).apply {
            setHint(R.string.hint_server_url)
            setText(store.getServerUrl())
            layoutParams = lp
        }
        val etUserId = EditText(this).apply {
            setHint(R.string.hint_user_id)
            setText(store.getUserId())
            layoutParams = lp
        }
        val etPassword = EditText(this).apply {
            setHint(R.string.hint_password)
            inputType = android.text.InputType.TYPE_CLASS_TEXT or
                    android.text.InputType.TYPE_TEXT_VARIATION_PASSWORD
            layoutParams = lp
        }
        val cbAllowSelfSigned = android.widget.CheckBox(this).apply {
            setText(R.string.allow_self_signed_label)
            isChecked = store.getAllowSelfSignedCert()
            layoutParams = lp
        }
        val btnSave = Button(this).apply {
            setText(R.string.save_and_test)
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
        layout.addView(cbAllowSelfSigned)
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
                Toast.makeText(this, R.string.toast_fill_all_fields, Toast.LENGTH_SHORT).show()
                return@setOnClickListener
            }
            // 平文HTTPはループバック限定。外部ホストへは資格情報が平文で流れるため保存前に拒否する
            if (!GkillServerUrlPolicy.isAllowed(serverUrl)) {
                Toast.makeText(this, R.string.url_policy_rejection, Toast.LENGTH_LONG).show()
                return@setOnClickListener
            }

            val passwordSha256 = sha256(password)
            val allowSelfSigned = cbAllowSelfSigned.isChecked
            store.setServerUrl(serverUrl)
            store.setUserId(userId)
            store.setPasswordSha256(passwordSha256)
            store.setAllowSelfSignedCert(allowSelfSigned)
            store.clearSession()

            tvStatus.setText(R.string.status_testing)

            CoroutineScope(Dispatchers.IO).launch {
                attemptConnect(serverUrl, userId, passwordSha256, allowSelfSigned, tvStatus, allowPinPrompt = true)
            }
        }

    }

    /**
     * ログインを試み、結果を [tvStatus] に反映する。
     *
     * ピン留めモード（[allowSelfSigned]）で自己署名証明書が拒否されたときは、
     * 提示されたフィンガープリントを確認ダイアログで表示し、利用者が明示承認した場合だけ
     * ピンを保存して1回だけ再接続する（[allowPinPrompt]=false で再入し、確認ループを防ぐ）。
     */
    private suspend fun attemptConnect(
        serverUrl: String,
        userId: String,
        passwordSha256: String,
        allowSelfSigned: Boolean,
        tvStatus: TextView,
        allowPinPrompt: Boolean
    ) {
        val hostKey = GkillServerTrust.hostKeyOf(serverUrl)
        val pin = store.getPinnedCertSha256(hostKey).ifEmpty { null }
        // locale_name は UI と同じ言語（サーバーがログイン失敗の文言を訳して返す）
        val client = GkillApiClient(serverUrl, allowSelfSigned, pin, GkillLocale.serverLocaleName(this))
        val (sessionId, errorMsg) = client.loginWithError(userId, passwordSha256)

        if (sessionId != null) {
            store.setSessionId(sessionId)
            withContext(Dispatchers.Main) {
                tvStatus.text = getString(R.string.status_connected, sessionId.take(8))
            }
            return
        }

        val capturedFingerprint = client.lastServerCertSha256
        if (allowPinPrompt && allowSelfSigned && client.lastServerCertRejected && capturedFingerprint != null) {
            withContext(Dispatchers.Main) {
                showPinConfirmDialog(capturedFingerprint) { approved ->
                    if (approved) {
                        store.setPinnedCertSha256(hostKey, capturedFingerprint)
                        tvStatus.setText(R.string.status_pin_saved_reconnecting)
                        CoroutineScope(Dispatchers.IO).launch {
                            // ピン保存後の再接続。確認ループを避けるため再プロンプトは無効
                            attemptConnect(serverUrl, userId, passwordSha256, allowSelfSigned, tvStatus, allowPinPrompt = false)
                        }
                    } else {
                        tvStatus.setText(R.string.status_failed_cert_unapproved)
                    }
                }
            }
            return
        }

        withContext(Dispatchers.Main) {
            // errorMsg はサーバーの error_message（訳済み）か GkillApiClient の ASCII コード。後者はここで訳す
            tvStatus.text = getString(R.string.status_failed, GkillErrorText.localize(this@MainActivity, errorMsg))
        }
    }

    /**
     * 未知の自己署名証明書のフィンガープリントを表示し、ピン留めの可否を利用者に尋ねる。
     * [onResult] は承認/拒否を渡してメインスレッドで呼ばれる。
     */
    private fun showPinConfirmDialog(fingerprintHex: String, onResult: (Boolean) -> Unit) {
        AlertDialog.Builder(this)
            .setTitle(R.string.cert_dialog_title)
            .setMessage(getString(R.string.cert_dialog_message, GkillServerTrust.formatFingerprint(fingerprintHex)))
            .setCancelable(false)
            .setPositiveButton(R.string.cert_dialog_trust) { _, _ -> onResult(true) }
            .setNegativeButton(R.string.cert_dialog_reject) { _, _ -> onResult(false) }
            .show()
    }

    private fun sha256(input: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
        val hash = digest.digest(input.toByteArray(Charsets.UTF_8))
        return hash.joinToString("") { "%02x".format(it) }
    }
}
