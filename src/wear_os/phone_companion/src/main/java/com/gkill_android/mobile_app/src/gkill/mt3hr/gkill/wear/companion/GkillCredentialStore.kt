package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.content.Context

/**
 * Stores gkill server credentials and session in SharedPreferences.
 */
class GkillCredentialStore(context: Context) {
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

    fun getPasswordSha256(): String = prefs.getString("password_sha256", "") ?: ""

    fun setPasswordSha256(hash: String) {
        prefs.edit().putString("password_sha256", hash).apply()
    }

    fun getSessionId(): String = prefs.getString("session_id", "") ?: ""

    fun setSessionId(id: String) {
        prefs.edit().putString("session_id", id).apply()
    }

    fun clearSession() {
        prefs.edit().remove("session_id").apply()
    }
}
