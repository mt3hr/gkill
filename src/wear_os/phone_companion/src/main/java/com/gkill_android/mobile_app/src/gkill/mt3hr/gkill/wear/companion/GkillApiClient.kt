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
        val kftl_template_struct: JsonElement? = null,
        val rep_struct: JsonElement? = null
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
     * Fetches playing (ongoing) TimeIs list from gkill server.
     * Returns JSON array string of PlaingTimeIsNode, or null on failure.
     *
     * Steps:
     * 1. get_application_config → extract all rep_names from rep_struct tree
     * 2. get_kyous with use_plaing=true → get Kyou IDs of playing items
     * 3. For each Kyou, get_timeis → get the latest TimeIs object
     * 4. Return as JSON array
     */
    fun getPlaingTimeis(sessionId: String): String? {
        val tag = "GkillApiClient"
        try {
            // get_kyous with use_plaing=true, use_reps=false
            val now = java.time.OffsetDateTime.now().format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
            Log.d(tag, "getPlaingTimeis: querying get_kyous with plaing_time=$now")
            val findQuery = buildPlaingFindQuery(now)
            val getKyousBody = JsonObject(mapOf(
                "session_id" to JsonPrimitive(sessionId),
                "query" to findQuery,
                "locale_name" to JsonPrimitive("ja")
            ))
            val kyousReq = Request.Builder()
                .url("$serverUrl/api/get_kyous")
                .post(getKyousBody.toString().toRequestBody(jsonMediaType))
                .build()
            val kyousRespBody = client.newCall(kyousReq).execute().use { resp ->
                if (!resp.isSuccessful) {
                    Log.e(tag, "get_kyous failed: HTTP ${resp.code}")
                    return null
                }
                resp.body?.string() ?: return null
            }
            val kyousJson = json.parseToJsonElement(kyousRespBody).jsonObject
            val errors = kyousJson["errors"]?.let { if (it is JsonNull) null else it.jsonArray }
            if (errors != null && errors.isNotEmpty()) {
                Log.e(tag, "get_kyous errors: $errors")
                return null
            }
            val kyous = kyousJson["kyous"]?.let { if (it is JsonNull) null else it.jsonArray } ?: JsonArray(emptyList())
            Log.d(tag, "getPlaingTimeis: got ${kyous.size} playing kyous")
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
                    "locale_name" to JsonPrimitive("ja")
                ))
                val timeisReq = Request.Builder()
                    .url("$serverUrl/api/get_timeis")
                    .post(getTimeisBody.toString().toRequestBody(jsonMediaType))
                    .build()
                val timeisRespBody = try {
                    client.newCall(timeisReq).execute().use { resp ->
                        if (!resp.isSuccessful) return@use null
                        resp.body?.string()
                    }
                } catch (e: Exception) {
                    Log.e(tag, "get_timeis failed for $kyouId", e)
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
            Log.e(tag, "getPlaingTimeis error", e)
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
                "locale_name" to JsonPrimitive("ja")
            ))
            val timeisReq = Request.Builder()
                .url("$serverUrl/api/get_timeis")
                .post(getTimeisBody.toString().toRequestBody(jsonMediaType))
                .build()
            val timeisRespBody = client.newCall(timeisReq).execute().use { resp ->
                if (!resp.isSuccessful) return "HTTP ${resp.code}"
                resp.body?.string() ?: return "empty response"
            }
            val timeisJson = json.parseToJsonElement(timeisRespBody).jsonObject
            val timeisErrors = timeisJson["errors"]?.let { if (it is JsonNull) null else it.jsonArray }
            if (timeisErrors != null && timeisErrors.isNotEmpty()) {
                return timeisErrors.first().jsonObject["error_message"]?.jsonPrimitive?.content ?: "get_timeis error"
            }
            val histories = timeisJson["timeis_histories"]?.let { if (it is JsonNull) null else it.jsonArray }
            if (histories.isNullOrEmpty()) return "TimeIs not found"

            // Get the latest history entry and modify it
            val latest = histories.last().jsonObject.toMutableMap()
            val now = java.time.OffsetDateTime.now().format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME)
            latest["end_time"] = JsonPrimitive(now)
            latest["update_time"] = JsonPrimitive(now)
            latest["update_app"] = JsonPrimitive("gkill_wear")

            // Step 2: update_timeis
            val updateBody = JsonObject(mapOf(
                "session_id" to JsonPrimitive(sessionId),
                "timeis" to JsonObject(latest),
                "locale_name" to JsonPrimitive("ja"),
                "want_response_kyou" to JsonPrimitive(false)
            ))
            val updateReq = Request.Builder()
                .url("$serverUrl/api/update_timeis")
                .post(updateBody.toString().toRequestBody(jsonMediaType))
                .build()
            return client.newCall(updateReq).execute().use { resp ->
                if (!resp.isSuccessful) return "HTTP ${resp.code}"
                val respBody = resp.body?.string() ?: return "empty response"
                val updateJson = json.parseToJsonElement(respBody).jsonObject
                val updateErrors = updateJson["errors"]?.let { if (it is JsonNull) null else it.jsonArray }
                if (updateErrors != null && updateErrors.isNotEmpty()) {
                    updateErrors.first().jsonObject["error_message"]?.jsonPrimitive?.content ?: "update error"
                } else {
                    null // success
                }
            }
        } catch (e: Exception) {
            Log.e(tag, "endTimeis error", e)
            return e.message ?: "unknown error"
        }
    }

    private fun buildPlaingFindQuery(plaingTime: String): JsonObject {
        return JsonObject(mapOf(
            "update_cache" to JsonPrimitive(false),
            "is_deleted" to JsonPrimitive(false),
            "use_tags" to JsonPrimitive(false),
            "use_reps" to JsonPrimitive(false),
            "use_rep_types" to JsonPrimitive(false),
            "rep_types" to JsonArray(emptyList()),
            "use_ids" to JsonPrimitive(false),
            "use_include_id" to JsonPrimitive(false),
            "ids" to JsonArray(emptyList()),
            "use_words" to JsonPrimitive(false),
            "words" to JsonArray(emptyList()),
            "words_and" to JsonPrimitive(false),
            "not_words" to JsonArray(emptyList()),
            "reps" to JsonArray(emptyList()),
            "tags" to JsonArray(emptyList()),
            "hide_tags" to JsonArray(emptyList()),
            "tags_and" to JsonPrimitive(false),
            "use_timeis" to JsonPrimitive(false),
            "timeis_words" to JsonArray(emptyList()),
            "timeis_not_words" to JsonArray(emptyList()),
            "timeis_words_and" to JsonPrimitive(false),
            "use_timeis_tags" to JsonPrimitive(false),
            "timeis_tags" to JsonArray(emptyList()),
            "hide_timeis_tags" to JsonArray(emptyList()),
            "timeis_tags_and" to JsonPrimitive(false),
            "use_calendar" to JsonPrimitive(false),
            "use_map" to JsonPrimitive(false),
            "map_radius" to JsonPrimitive(0.0),
            "map_latitude" to JsonPrimitive(0.0),
            "map_longitude" to JsonPrimitive(0.0),
            "include_create_mi" to JsonPrimitive(false),
            "include_check_mi" to JsonPrimitive(false),
            "include_limit_mi" to JsonPrimitive(false),
            "include_start_mi" to JsonPrimitive(false),
            "include_end_mi" to JsonPrimitive(false),
            "include_end_timeis" to JsonPrimitive(false),
            "use_plaing" to JsonPrimitive(true),
            "plaing_time" to JsonPrimitive(plaingTime),
            "use_update_time" to JsonPrimitive(false),
            "is_image_only" to JsonPrimitive(false),
            "for_mi" to JsonPrimitive(false),
            "use_mi_board_name" to JsonPrimitive(false),
            "use_period_of_time" to JsonPrimitive(false),
            "mi_board_name" to JsonPrimitive(""),
            "mi_check_state" to JsonPrimitive(""),
            "mi_sort_type" to JsonPrimitive(""),
            "only_latest_data" to JsonPrimitive(false),
            "include_deleted_data" to JsonPrimitive(false)
        ))
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
