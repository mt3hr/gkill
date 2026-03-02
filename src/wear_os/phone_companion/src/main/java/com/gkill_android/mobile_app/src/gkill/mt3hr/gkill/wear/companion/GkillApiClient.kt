package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.security.SecureRandom
import java.security.cert.X509Certificate
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLContext
import javax.net.ssl.TrustManager
import javax.net.ssl.X509TrustManager

/** Minimal gkill API client using OkHttp (blocking, call from coroutine/thread). */
class GkillApiClient(private val serverUrl: String) {

    private val client = buildOkHttpClient()

    /**
     * Build an OkHttpClient that trusts all certificates.
     * gkill typically runs on localhost with a self-signed certificate,
     * so standard certificate validation would always fail.
     */
    private fun buildOkHttpClient(): OkHttpClient {
        val trustAll = object : X509TrustManager {
            override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String) {}
            override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String) {}
            override fun getAcceptedIssuers(): Array<X509Certificate> = arrayOf()
        }
        val sslContext = SSLContext.getInstance("TLS").apply {
            init(null, arrayOf<TrustManager>(trustAll), SecureRandom())
        }
        return OkHttpClient.Builder()
            .sslSocketFactory(sslContext.socketFactory, trustAll)
            .hostnameVerifier { _, _ -> true }
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    private val json = Json { ignoreUnknownKeys = true }
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    // ─── Data classes ──────────────────────────────────────────────────────────

    @Serializable
    data class LoginRequest(val user_id: String, val password_sha256: String)

    @Serializable
    data class LoginResponse(
        val session_id: String = "",
        val errors: List<GkillError>? = null
    )

    @Serializable
    data class GetApplicationConfigRequest(val session_id: String, val locale_name: String = "ja")

    @Serializable
    data class GetApplicationConfigResponse(
        val application_config: ApplicationConfigPartial? = null,
        val errors: List<GkillError>? = null
    )

    @Serializable
    data class ApplicationConfigPartial(
        val kftl_template_struct: JsonElement? = null
    )

    @Serializable
    data class SubmitKFTLTextRequest(
        val session_id: String,
        val kftl_text: String,
        val locale_name: String = "ja"
    )

    @Serializable
    data class SubmitKFTLTextResponse(
        val errors: List<GkillError>? = null
    )

    @Serializable
    data class GkillError(
        val error_code: String = "",
        val error_message: String = ""
    )

    // ─── API calls ─────────────────────────────────────────────────────────────

    /**
     * Logs in and returns the session_id, or null on failure.
     */
    fun login(userId: String, passwordSha256: String): String? = loginWithError(userId, passwordSha256).first

    /**
     * Logs in and returns Pair(session_id, errorMessage).
     * session_id is null on failure; errorMessage is empty on success.
     */
    fun loginWithError(userId: String, passwordSha256: String): Pair<String?, String> {
        val reqJson = json.encodeToString(LoginRequest.serializer(), LoginRequest(userId, passwordSha256))
        val body = reqJson.toRequestBody(jsonMediaType)
        val req = Request.Builder()
            .url("$serverUrl/api/login")
            .post(body)
            .build()
        return try {
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return Pair(null, "HTTP ${resp.code}")
                val respJson = resp.body?.string() ?: return Pair(null, "レスポンスが空です")
                val loginResp = json.decodeFromString(LoginResponse.serializer(), respJson)
                if (!loginResp.errors.isNullOrEmpty()) {
                    Pair(null, loginResp.errors.first().error_message)
                } else {
                    val sid = loginResp.session_id.ifEmpty { null }
                    if (sid != null) Pair(sid, "") else Pair(null, "セッションIDが空です")
                }
            }
        } catch (e: Exception) {
            Pair(null, e.message ?: "不明なエラー")
        }
    }

    /**
     * Fetches the application config and returns the kftl_template_struct JSON string,
     * or null on failure.
     */
    fun getKftlTemplateStructJson(sessionId: String): String? {
        val reqJson = json.encodeToString(
            GetApplicationConfigRequest.serializer(),
            GetApplicationConfigRequest(sessionId)
        )
        val body = reqJson.toRequestBody(jsonMediaType)
        val req = Request.Builder()
            .url("$serverUrl/api/get_application_config")
            .post(body)
            .build()
        return try {
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return null
                val respJson = resp.body?.string() ?: return null
                val configResp = json.decodeFromString(GetApplicationConfigResponse.serializer(), respJson)
                if (!configResp.errors.isNullOrEmpty()) return null
                configResp.application_config?.kftl_template_struct?.toString()
            }
        } catch (_: Exception) {
            null
        }
    }

    /**
     * Submits KFTL text and returns null on success, or an error message on failure.
     */
    fun submitKFTLText(sessionId: String, kftlText: String): String? {
        val reqJson = json.encodeToString(
            SubmitKFTLTextRequest.serializer(),
            SubmitKFTLTextRequest(sessionId, kftlText)
        )
        val body = reqJson.toRequestBody(jsonMediaType)
        val req = Request.Builder()
            .url("$serverUrl/api/submit_kftl_text")
            .post(body)
            .build()
        return try {
            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) return "HTTP ${resp.code}"
                val respJson = resp.body?.string() ?: return "empty response"
                val submitResp = json.decodeFromString(SubmitKFTLTextResponse.serializer(), respJson)
                if (!submitResp.errors.isNullOrEmpty()) submitResp.errors.first().error_message else null
            }
        } catch (e: Exception) {
            e.message ?: "unknown error"
        }
    }
}
