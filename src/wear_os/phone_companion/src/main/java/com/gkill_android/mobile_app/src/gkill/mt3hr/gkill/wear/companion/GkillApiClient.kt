package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.util.Log
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonArray
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonNull
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.security.SecureRandom
import java.util.concurrent.TimeUnit
import javax.net.ssl.SSLContext
import javax.net.ssl.TrustManager

/**
 * Minimal gkill API client using OkHttp (blocking, call from coroutine/thread).
 *
 * @param allowSelfSignedCert When true, the client runs in "pinned self-signed"
 * mode: the platform default TrustManager is still tried first, and a self-signed
 * certificate is only accepted when its SHA-256 fingerprint matches [pinnedCertSha256].
 * Certificate/hostname verification is never disabled outright (no trust-all).
 * Defaults to false (standard platform validation only).
 * @param pinnedCertSha256 Saved leaf-certificate SHA-256 fingerprint (hex, no
 * separators) for this server's host. null means no pin is stored yet, so a
 * self-signed certificate will be rejected until the user pins it via the settings
 * screen. Ignored when [allowSelfSignedCert] is false.
 * @param localeName 全リクエストの `locale_name` に載せる言語コード（ja / en / zh / ko / es / fr / de）。
 * サーバーはこれで `error_message` を訳して返す。本番は [GkillLocale.serverLocaleName] で
 * UI が解決したロケールを渡す。既定値を `Locale.getDefault()` 由来にしないのは、JVM 単体テストの
 * 結果が実行機の OS 言語で変わるから（既定 = サーバー側フォールバックと同じ ja）。
 */
class GkillApiClient(
    private val serverUrl: String,
    private val allowSelfSignedCert: Boolean = false,
    private val pinnedCertSha256: String? = null,
    private val localeName: String = GkillLocale.DEFAULT_LOCALE_NAME
) {

    companion object {
        /**
         * 記録の `create_app` / `update_app` に載せるアプリ名。KFTL 送信の `create_app` と
         * 打刻終了の `update_app` の両方で使う。表示名ではなく機械的な識別子なので、
         * アプリ名の改名に追随させてはいけない（検索の `create_apps` 絞り込みの語彙になる）。
         */
        const val APP_NAME = "gkill_wear"
    }

    private val serverTrust: GkillServerTrust? =
        if (allowSelfSignedCert) GkillServerTrust(pinnedCertSha256) else null

    private val client = buildOkHttpClient()

    /** 直近のハンドシェイクで提示されたリーフ証明書の SHA-256。学習UIでの表示に使う。 */
    val lastServerCertSha256: String?
        get() = serverTrust?.lastLeafSha256

    /**
     * 直近のハンドシェイクでリーフ証明書が拒否されたか（証明書を提示されたが信頼できなかった）。
     * true のときだけピン留めの確認を利用者に出す。
     */
    val lastServerCertRejected: Boolean
        get() = serverTrust?.let { it.lastLeafSha256 != null && !it.lastServerTrusted } ?: false

    private fun buildOkHttpClient(): OkHttpClient {
        val builder = OkHttpClient.Builder()
            .connectTimeout(10, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
        val trust = serverTrust
        if (trust != null) {
            val tm = trust.trustManager()
            val sslContext = SSLContext.getInstance("TLS").apply {
                init(null, arrayOf<TrustManager>(tm), SecureRandom())
            }
            builder.sslSocketFactory(sslContext.socketFactory, tm)
                .hostnameVerifier(trust.hostnameVerifier())
        }
        return builder.build()
    }

    private val json = Json { ignoreUnknownKeys = true }
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    // ─── Data classes ──────────────────────────────────────────────────────────

    // locale_name に既定値を置かない: Json は encodeDefaults=false なので、既定値のままだと
    // キーごと送られずサーバー側フォールバック(ja)になる。常に明示して送る。
    @Serializable
    data class LoginRequest(val user_id: String, val password_sha256: String, val locale_name: String)

    @Serializable
    data class LoginResponse(
        val session_id: String = "",
        val errors: List<GkillError>? = null
    )

    @Serializable
    data class GetApplicationConfigRequest(val session_id: String, val locale_name: String)

    @Serializable
    data class GetApplicationConfigResponse(
        val application_config: ApplicationConfigPartial? = null,
        val errors: List<GkillError>? = null
    )

    @Serializable
    data class ApplicationConfigPartial(
        val kftl_template_struct: JsonElement? = null,
        val rep_struct: JsonElement? = null
    )

    @Serializable
    data class SubmitKFTLTextRequest(
        val session_id: String,
        val kftl_text: String,
        val locale_name: String,
        // 空文字はサーバーの omitempty と Json の encodeDefaults=false で送信時に落ちる。
        // ワーカー再送で同じキーを送ると二重登録にならない（監査 S3-wear）。
        val idempotency_key: String = "",
        // 記録の create_app / update_app に載る。既定値を置かない: encodeDefaults=false で
        // キーごと落ちると、サーバーはエラーを出さずに "gkill_kftl"（メモ帳と同じ値）へ戻す。
        val create_app: String
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

    @Serializable
    data class GkillMessage(
        val message_code: String = "",
        val message: String = ""
    )

    // ─── API calls ─────────────────────────────────────────────────────────────

    /**
     * Logs in and returns the session_id, or null on failure.
     */
    fun login(userId: String, passwordSha256: String): String? = loginWithError(userId, passwordSha256).first

    /**
     * Logs in and returns Pair(session_id, errorMessage).
     * session_id is null on failure; errorMessage is empty on success.
     *
     * 失敗時の文字列はサーバーの `error_message`（locale_name で訳済み）か、
     * このクラス自身が返す ASCII コード（[WIRE_ERR_EMPTY_RESPONSE] 等。表示前に
     * [GkillErrorText.localize] で訳す）か、`HTTP <code>` / 例外メッセージのいずれか。
     */
    fun loginWithError(userId: String, passwordSha256: String): Pair<String?, String> {
        val reqJson = json.encodeToString(LoginRequest.serializer(), LoginRequest(userId, passwordSha256, localeName))
        val body = reqJson.toRequestBody(jsonMediaType)
        val req = Request.Builder()
            .url("$serverUrl/api/login")
            .post(body)
            .build()
        return try {
            client.newCall(req).execute().use { resp ->
                // gkillは異常時に4xx/5xxを返すが、理由(error_message)は本文にしか入っていない。
                // ステータスで打ち切ると時計に「HTTP 401」としか出せないので、先に本文を読む。
                val respJson = resp.body.string().ifEmpty {
                    return Pair(null, if (resp.isSuccessful) WIRE_ERR_EMPTY_RESPONSE else "HTTP ${resp.code}")
                }
                val loginResp = json.decodeFromString(LoginResponse.serializer(), respJson)
                if (!loginResp.errors.isNullOrEmpty()) {
                    Pair(null, loginResp.errors.first().error_message)
                } else {
                    val sid = loginResp.session_id.ifEmpty { null }
                    if (sid != null) Pair(sid, "") else Pair(null, WIRE_ERR_EMPTY_SESSION_ID)
                }
            }
        } catch (e: Exception) {
            Pair(null, e.message ?: WIRE_ERR_UNKNOWN)
        }
    }

    /**
     * Fetches the application config and returns the kftl_template_struct JSON string,
     * or null on failure.
     */
    fun getKftlTemplateStructJson(sessionId: String): String? {
        val reqJson = json.encodeToString(
            GetApplicationConfigRequest.serializer(),
            GetApplicationConfigRequest(sessionId, localeName)
        )
        val body = reqJson.toRequestBody(jsonMediaType)
        val req = Request.Builder()
            .url("$serverUrl/api/get_application_config")
            .post(body)
            .build()
        return try {
            client.newCall(req).execute().use { resp ->
                // ステータスで打ち切らず本文を読む(gkillのerrorsは本文にしかない)。
                val respJson = resp.body.string().ifEmpty { return null }
                val configResp = json.decodeFromString(GetApplicationConfigResponse.serializer(), respJson)
                if (!configResp.errors.isNullOrEmpty()) return null
                configResp.application_config?.kftl_template_struct?.toString()
            }
        } catch (_: Exception) {
            null
        }
    }

    /**
     * Fetches playing (ongoing) TimeIs list from gkill server.
     * Returns JSON array string of PlayingTimeIsNode, or null on failure.
     *
     * Steps:
     * 1. get_application_config → extract all rep_names from rep_struct tree
     * 2. get_kyous with a non-null playing_time → get Kyou IDs of playing items
     * 3. For each Kyou, get_timeis → get the latest TimeIs object
     * 4. Return as JSON array
     */
    fun getPlayingTimeis(sessionId: String): String? {
        val tag = "GkillApiClient"
        try {
            // get_kyous with a non-null playing_time (all other filters unused = omitted)
            val now = java.time.OffsetDateTime.now().format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
            Log.d(tag, "getPlayingTimeis: querying get_kyous")
            val findQuery = buildPlayingFindQuery(now)
            val getKyousBody = JsonObject(mapOf(
                "session_id" to JsonPrimitive(sessionId),
                "query" to findQuery,
                "locale_name" to JsonPrimitive(localeName)
            ))
            val kyousReq = Request.Builder()
                .url("$serverUrl/api/get_kyous")
                .post(getKyousBody.toString().toRequestBody(jsonMediaType))
                .build()
            val kyousRespBody = client.newCall(kyousReq).execute().use { resp ->
                if (!resp.isSuccessful) {
                    // 打ち切らずに本文まで読む。gkillは4xx/5xxでもerrors配列を返すので、
                    // ステータスだけをログに残すと原因が追えない。
                    Log.e(tag, "get_kyous failed: HTTP ${resp.code}")
                }
                resp.body.string().ifEmpty { return null }
            }
            val kyousJson = json.parseToJsonElement(kyousRespBody).jsonObject
            val errors = kyousJson["errors"]?.let { if (it is JsonNull) null else it.jsonArray }
            if (errors != null && errors.isNotEmpty()) {
                // errors の本文には環境依存の文言が入りうるので error_code だけを残す(2026-08-30 監査 F-008)
                val codes = errors.mapNotNull { it.jsonObject["error_code"]?.jsonPrimitive?.content }
                Log.e(tag, "get_kyous errors: ${codes.joinToString(",")}")
                return null
            }
            val kyous = kyousJson["kyous"]?.let { if (it is JsonNull) null else it.jsonArray } ?: JsonArray(emptyList())
            Log.d(tag, "getPlayingTimeis: got ${kyous.size} playing kyous")
            if (kyous.isEmpty()) return "[]"

            // Step 3: For each Kyou, get_timeis to get details
            val resultList = mutableListOf<JsonObject>()
            for (kyou in kyous) {
                val kyouObj = kyou.jsonObject
                val kyouId = kyouObj["id"]?.jsonPrimitive?.content ?: continue
                val repName = kyouObj["rep_name"]?.jsonPrimitive?.content ?: ""

                val getTimeisBody = JsonObject(mapOf(
                    "session_id" to JsonPrimitive(sessionId),
                    "id" to JsonPrimitive(kyouId),
                    "locale_name" to JsonPrimitive(localeName)
                ))
                val timeisReq = Request.Builder()
                    .url("$serverUrl/api/get_timeis")
                    .post(getTimeisBody.toString().toRequestBody(jsonMediaType))
                    .build()
                val timeisRespBody = try {
                    client.newCall(timeisReq).execute().use { resp ->
                        // ステータスで打ち切らない(本文のerrorsが唯一の手がかり)。
                        resp.body.string().ifEmpty { null }
                    }
                } catch (e: Exception) {
                    // Kyou ID は記録と突き合わせられる識別子なので logcat へ出さない(2026-08-30 監査 F-008)
                    Log.e(tag, "get_timeis failed", e)
                    null
                }
                if (timeisRespBody == null) continue

                val timeisJson = json.parseToJsonElement(timeisRespBody).jsonObject
                val histories = timeisJson["timeis_histories"]?.let { if (it is JsonNull) null else it.jsonArray }
                if (histories.isNullOrEmpty()) continue

                // Get the latest history entry (last element)
                val latest = histories.last().jsonObject
                val title = latest["title"]?.jsonPrimitive?.content ?: ""
                val startTime = latest["start_time"]?.jsonPrimitive?.content ?: ""
                val dataType = latest["data_type"]?.jsonPrimitive?.content ?: ""
                val isDeleted = latest["is_deleted"]?.jsonPrimitive?.boolean ?: false

                resultList.add(JsonObject(mapOf(
                    "id" to JsonPrimitive(kyouId),
                    "rep_name" to JsonPrimitive(repName),
                    "title" to JsonPrimitive(title),
                    "start_time" to JsonPrimitive(startTime),
                    "data_type" to JsonPrimitive(dataType),
                    "is_deleted" to JsonPrimitive(isDeleted)
                )))
            }

            return JsonArray(resultList).toString()
        } catch (e: Exception) {
            Log.e(tag, "getPlayingTimeis error", e)
            return null
        }
    }

    /**
     * Ends (stops) a playing TimeIs by setting its end_time to now.
     * Returns null on success, or error message on failure.
     *
     * Steps:
     * 1. get_timeis to get the full latest TimeIs object
     * 2. Set end_time to now, update_time to now, update_app to "gkill_wear"
     * 3. update_timeis to save
     */
    fun endTimeis(sessionId: String, timeisId: String, repName: String): String? {
        val tag = "GkillApiClient"
        try {
            // Step 1: Get the full TimeIs object
            val getTimeisBody = JsonObject(mapOf(
                "session_id" to JsonPrimitive(sessionId),
                "id" to JsonPrimitive(timeisId),
                "locale_name" to JsonPrimitive(localeName)
            ))
            val timeisReq = Request.Builder()
                .url("$serverUrl/api/get_timeis")
                .post(getTimeisBody.toString().toRequestBody(jsonMediaType))
                .build()
            val timeisRespBody = client.newCall(timeisReq).execute().use { resp ->
                // 本文のerrorsを優先して読む。空のときだけステータスを出す。
                resp.body.string().ifEmpty {
                    return if (resp.isSuccessful) WIRE_ERR_EMPTY_RESPONSE else "HTTP ${resp.code}"
                }
            }
            val timeisJson = json.parseToJsonElement(timeisRespBody).jsonObject
            val timeisErrors = timeisJson["errors"]?.let { if (it is JsonNull) null else it.jsonArray }
            if (timeisErrors != null && timeisErrors.isNotEmpty()) {
                return timeisErrors.first().jsonObject["error_message"]?.jsonPrimitive?.content ?: WIRE_ERR_GET_TIMEIS_FAILED
            }
            val histories = timeisJson["timeis_histories"]?.let { if (it is JsonNull) null else it.jsonArray }
            if (histories.isNullOrEmpty()) return WIRE_ERR_TIMEIS_NOT_FOUND

            // Get the latest history entry and modify it
            val latest = histories.last().jsonObject.toMutableMap()
            val now = java.time.OffsetDateTime.now().format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
            latest["end_time"] = JsonPrimitive(now)
            latest["update_time"] = JsonPrimitive(now)
            latest["update_app"] = JsonPrimitive(APP_NAME)

            // Step 2: update_timeis
            val updateBody = JsonObject(mapOf(
                "session_id" to JsonPrimitive(sessionId),
                "timeis" to JsonObject(latest),
                "locale_name" to JsonPrimitive(localeName),
                "want_response_kyou" to JsonPrimitive(false)
            ))
            val updateReq = Request.Builder()
                .url("$serverUrl/api/update_timeis")
                .post(updateBody.toString().toRequestBody(jsonMediaType))
                .build()
            return client.newCall(updateReq).execute().use { resp ->
                // 本文のerrorsを優先して読む。空のときだけステータスを出す。
                val respBody = resp.body.string().ifEmpty {
                    return if (resp.isSuccessful) WIRE_ERR_EMPTY_RESPONSE else "HTTP ${resp.code}"
                }
                val updateJson = json.parseToJsonElement(respBody).jsonObject
                val updateErrors = updateJson["errors"]?.let { if (it is JsonNull) null else it.jsonArray }
                if (updateErrors != null && updateErrors.isNotEmpty()) {
                    updateErrors.first().jsonObject["error_message"]?.jsonPrimitive?.content ?: WIRE_ERR_UPDATE_TIMEIS_FAILED
                } else {
                    null // success
                }
            }
        } catch (e: Exception) {
            Log.e(tag, "endTimeis error", e)
            return e.message ?: WIRE_ERR_UNKNOWN
        }
    }

    // FindQuery is null-based: a filter group is active only when its value field is
    // present and non-null. Omitted keys mean "filter unused", so the playing query
    // sends only playing_time. Never send empty arrays for unused filters — [] means
    // "filter enabled with zero selections", which matches nothing.
    private fun buildPlayingFindQuery(playingTime: String): JsonObject {
        return JsonObject(mapOf(
            "playing_time" to JsonPrimitive(playingTime)
        ))
    }

    /**
     * Submits KFTL text and returns null on success, or an error message on failure.
     *
     * [idempotencyKey] を渡すと、同じキーの再送はサーバー側で1回の登録に畳まれる
     * （ワーカー再送で結果だけ届かなかった場合の二重登録を防ぐ）。null/空なら従来どおり毎回登録。
     */
    fun submitKFTLText(sessionId: String, kftlText: String, idempotencyKey: String? = null): String? {
        val reqJson = json.encodeToString(
            SubmitKFTLTextRequest.serializer(),
            SubmitKFTLTextRequest(sessionId, kftlText, localeName, idempotency_key = idempotencyKey ?: "", create_app = APP_NAME)
        )
        val body = reqJson.toRequestBody(jsonMediaType)
        val req = Request.Builder()
            .url("$serverUrl/api/submit_kftl_text")
            .post(body)
            .build()
        return try {
            client.newCall(req).execute().use { resp ->
                // 本文のerrorsを優先して読む。空のときだけステータスを出す。
                val respJson = resp.body.string().ifEmpty {
                    return if (resp.isSuccessful) WIRE_ERR_EMPTY_RESPONSE else "HTTP ${resp.code}"
                }
                val submitResp = json.decodeFromString(SubmitKFTLTextResponse.serializer(), respJson)
                if (!submitResp.errors.isNullOrEmpty()) submitResp.errors.first().error_message else null
            }
        } catch (e: Exception) {
            e.message ?: WIRE_ERR_UNKNOWN
        }
    }
}
