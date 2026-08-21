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

    // ─── ピン留め証明書のフィンガープリント ──────────────────────────────────
    // 自己署名サーバー向けの TOFU（Trust On First Use）で承認したリーフ証明書の
    // SHA-256（16進）をホストごとに保存する。証明書のハッシュは秘密情報ではないので
    // [cipher] を通さず平文で保存する。[hostKey] は [GkillServerTrust.hostKeyOf] で作る。

    fun getPinnedCertSha256(hostKey: String): String =
        prefs.getString(pinnedCertKey(hostKey), "") ?: ""

    fun setPinnedCertSha256(hostKey: String, sha256: String) {
        if (sha256.isEmpty()) {
            prefs.edit().remove(pinnedCertKey(hostKey)).apply()
            return
        }
        prefs.edit().putString(pinnedCertKey(hostKey), sha256).apply()
    }

    fun clearPinnedCertSha256(hostKey: String) {
        prefs.edit().remove(pinnedCertKey(hostKey)).apply()
    }

    private fun pinnedCertKey(hostKey: String): String = "pinned_cert_sha256:$hostKey"

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
