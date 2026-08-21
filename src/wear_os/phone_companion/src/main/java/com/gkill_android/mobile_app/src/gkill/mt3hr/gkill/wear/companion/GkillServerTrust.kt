package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import okhttp3.OkHttpClient
import java.net.URI
import java.security.KeyStore
import java.security.MessageDigest
import java.security.cert.CertificateException
import java.security.cert.X509Certificate
import javax.net.ssl.HostnameVerifier
import javax.net.ssl.TrustManagerFactory
import javax.net.ssl.X509TrustManager

/**
 * 自己署名サーバー向けの TOFU（Trust On First Use）+ ピン留めを担う。
 *
 * かつては `allow_self_signed_cert` が true のとき証明書検証を丸ごと無効化した
 * trust-all TrustManager と常時 true の HostnameVerifier を使っていたが、
 * それは中間者攻撃を素通しするため廃止した。現在の約束:
 *
 * - まずプラットフォーム既定の TrustManager で検証する（正規の CA 証明書はそのまま通る）。
 * - 失敗したときだけ「保存済み SHA-256 フィンガープリントとリーフ証明書の完全一致」で許可する。
 * - ピンが未保存なら自己署名証明書は拒否される（安全側の既定）。
 *   ピンの学習はコンパニオンの「保存 & 接続テスト」で利用者が明示承認したときだけ行い、
 *   [GkillWearableListenerService] 経路（ワーカー）は保存済みピンの照合しかしない。
 * - HostnameVerifier は OkHttp 既定を維持し、既定が失敗したときだけ
 *   「提示されたリーフ証明書がピン済みとバイト一致」する場合に限って許可する
 *   （SAN 欠落の自己署名証明書対策）。
 *
 * [pinnedCertSha256] は対象ホストの保存済みフィンガープリント（16進、区切りなし）。
 * null なら「ピンなし」＝プラットフォーム既定のみで検証する。
 */
class GkillServerTrust(private val pinnedCertSha256: String?) {

    /** 直近のハンドシェイクで提示されたリーフ証明書の SHA-256（16進）。学習UIでの表示に使う。 */
    @Volatile
    var lastLeafSha256: String? = null
        private set

    /** 直近のハンドシェイクでリーフ証明書が信頼された（プラットフォームまたはピン一致）か。 */
    @Volatile
    var lastServerTrusted: Boolean = false
        private set

    private val platformTrustManager: X509TrustManager = defaultPlatformTrustManager()

    fun trustManager(): X509TrustManager = object : X509TrustManager {
        override fun checkClientTrusted(chain: Array<out X509Certificate>?, authType: String?) {
            platformTrustManager.checkClientTrusted(chain, authType)
        }

        override fun checkServerTrusted(chain: Array<out X509Certificate>?, authType: String?) {
            lastServerTrusted = false
            val leaf = chain?.firstOrNull()
            // 失敗時も学習UIで表示できるよう、拒否する前に必ずフィンガープリントを控える
            if (leaf != null) {
                lastLeafSha256 = certSha256Hex(leaf)
            }
            try {
                platformTrustManager.checkServerTrusted(chain, authType)
                lastServerTrusted = true
                return
            } catch (e: CertificateException) {
                val pin = pinnedCertSha256
                if (pin != null && leaf != null && fingerprintsEqual(certSha256Hex(leaf), pin)) {
                    lastServerTrusted = true
                    return
                }
                throw e
            }
        }

        override fun getAcceptedIssuers(): Array<X509Certificate> = platformTrustManager.acceptedIssuers
    }

    fun hostnameVerifier(): HostnameVerifier = HostnameVerifier { hostname, session ->
        // まず OkHttp 既定のホスト名検証（正規証明書のホスト名一致はここで通る）
        if (defaultHostnameVerifier.verify(hostname, session)) {
            return@HostnameVerifier true
        }
        // 既定が失敗した場合のみ、提示証明書がピン済みとバイト一致するなら許可する。
        // SAN を持たない自己署名証明書はホスト名一致で必ず落ちるため、この経路が要る。
        val pin = pinnedCertSha256 ?: return@HostnameVerifier false
        val leaf = try {
            session.peerCertificates.firstOrNull() as? X509Certificate
        } catch (_: Exception) {
            null
        } ?: return@HostnameVerifier false
        fingerprintsEqual(certSha256Hex(leaf), pin)
    }

    companion object {
        // OkHttp 既定の HostnameVerifier。内部クラス OkHostnameVerifier は他モジュールから
        // internal 可視性で参照できないため、既定クライアントの公開プロパティ経由で取り出す。
        private val defaultHostnameVerifier: HostnameVerifier by lazy { OkHttpClient().hostnameVerifier }

        /** リーフ証明書（DER 全体）の SHA-256 を小文字16進で返す。 */
        fun certSha256Hex(cert: X509Certificate): String {
            val digest = MessageDigest.getInstance("SHA-256").digest(cert.encoded)
            return digest.joinToString("") { "%02x".format(it) }
        }

        /** 表示用に `AB:CD:...` 形式へ整形する。 */
        fun formatFingerprint(hex: String): String =
            hex.replace(":", "").chunked(2).joinToString(":").uppercase()

        /**
         * 2つのフィンガープリントが等しいか。区切りコロンと大文字小文字を無視する。
         * 公開情報だが慣習として定時間比較を使う。
         */
        fun fingerprintsEqual(a: String, b: String): Boolean {
            val na = a.replace(":", "").lowercase()
            val nb = b.replace(":", "").lowercase()
            if (na.isEmpty() || na.length != nb.length) {
                return false
            }
            return MessageDigest.isEqual(
                na.toByteArray(Charsets.US_ASCII),
                nb.toByteArray(Charsets.US_ASCII)
            )
        }

        /**
         * サーバーURLからピン保存キー用のホスト識別子を作る。
         * 明示ポートがあれば `host:port`、無ければ `host`。パース不能ならURL全体を小文字化して使う。
         */
        fun hostKeyOf(url: String): String {
            return try {
                val uri = URI(url.trim())
                val host = uri.host?.lowercase()
                when {
                    host.isNullOrEmpty() -> url.trim().lowercase()
                    uri.port > 0 -> "$host:${uri.port}"
                    else -> host
                }
            } catch (_: Exception) {
                url.trim().lowercase()
            }
        }

        private fun defaultPlatformTrustManager(): X509TrustManager {
            val tmf = TrustManagerFactory.getInstance(TrustManagerFactory.getDefaultAlgorithm())
            tmf.init(null as KeyStore?)
            return tmf.trustManagers.filterIsInstance<X509TrustManager>().first()
        }
    }
}
