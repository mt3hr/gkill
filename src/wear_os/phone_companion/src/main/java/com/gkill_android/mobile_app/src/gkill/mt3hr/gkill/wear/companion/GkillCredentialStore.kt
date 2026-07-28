package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.content.Context

/**
 * Stores gkill server credentials and session in SharedPreferences.
 *
 * password_sha256 と session_id は [GkillSecretCipher] で暗号化して保存する。
 * 旧バージョンが平文で書いた値も読めるようにしてあり、次の書き込みで暗号化される。
 */
class GkillCredentialStore(
    context: Context,
    private val cipher: GkillSecretCipher = AndroidKeystoreSecretCipher(),
) {
    private val prefs = context.getSharedPreferences("gkill_wear_prefs", Context.MODE_PRIVATE)

    fun getServerUrl(): String =
        prefs.getString("server_url", "http://localhost:9999") ?: "http://localhost:9999"

    fun setServerUrl(url: String) {
        prefs.edit().putString("server_url", url).apply()
    }

    fun getUserId(): String = prefs.getString("user_id", "") ?: ""

    fun setUserId(id: String) {
        prefs.edit().putString("user_id", id).apply()
    }

    fun getPasswordSha256(): String = getSecret("password_sha256")

    fun setPasswordSha256(hash: String) {
        setSecret("password_sha256", hash)
    }

    fun getAllowSelfSignedCert(): Boolean = prefs.getBoolean("allow_self_signed_cert", false)

    fun setAllowSelfSignedCert(allow: Boolean) {
        prefs.edit().putBoolean("allow_self_signed_cert", allow).apply()
    }

    fun getSessionId(): String = getSecret("session_id")

    fun setSessionId(id: String) {
        setSecret("session_id", id)
    }

    fun clearSession() {
        prefs.edit().remove("session_id").apply()
    }

    private fun getSecret(key: String): String {
        val stored = prefs.getString(key, "") ?: ""
        if (stored.isEmpty()) {
            return ""
        }
        if (!cipher.isEncrypted(stored)) {
            // 旧バージョンが平文で書いた値。次の書き込みで暗号化される
            return stored
        }
        return cipher.decrypt(stored) ?: ""
    }

    private fun setSecret(key: String, value: String) {
        if (value.isEmpty()) {
            prefs.edit().remove(key).apply()
            return
        }
        // 暗号化できなかった場合は平文で残さず、保存自体を諦める
        val encrypted = cipher.encrypt(value) ?: return
        prefs.edit().putString(key, encrypted).apply()
    }
}
