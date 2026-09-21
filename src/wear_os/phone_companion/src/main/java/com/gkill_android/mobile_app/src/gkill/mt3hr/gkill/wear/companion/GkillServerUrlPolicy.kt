package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import java.net.URI

/**
 * サーバーURLの受け入れ判定。
 *
 * 平文HTTPはループバック(localhost / 127.0.0.0/8 / ::1)だけ許可する。
 * それ以外のホストへ http:// で繋ぐと password_sha256 とセッションIDが平文で流れ、
 * 経路上の第三者に再利用可能な資格情報を渡すことになる(指摘 F-007)。
 * network_security_config でも遮断されるが、保存前にここで拒否して理由を示す
 * (文言は `R.string.url_policy_rejection`。この object は Android 非依存に保つ)。
 * HTTPS は制限しない(自己署名は GkillServerTrust の TOFU/ピン留めが受け持つ)。
 */
object GkillServerUrlPolicy {

    fun isAllowed(serverUrl: String): Boolean {
        val uri = try {
            URI(serverUrl)
        } catch (_: Exception) {
            return false
        }
        val host = uri.host ?: return false
        return when (uri.scheme?.lowercase()) {
            "https" -> true
            "http" -> isLoopbackHost(host)
            else -> false
        }
    }

    /**
     * 名前解決はしない(保存ボタンの同期処理で I/O をしない)。
     * 文字列として判定できるループバックだけを許可する。
     */
    private fun isLoopbackHost(host: String): Boolean {
        // Java の URI.getHost() は IPv6 リテラルを角括弧付きで返す
        val h = host.trim('[', ']').lowercase()
        if (h == "localhost") return true
        if (h == "::1" || h == "0:0:0:0:0:0:0:1") return true
        return Regex("""^127\.\d{1,3}\.\d{1,3}\.\d{1,3}$""").matches(h)
    }
}
