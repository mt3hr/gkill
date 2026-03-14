package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.util.Log
import com.google.android.gms.wearable.MessageClient
import com.google.android.gms.wearable.MessageEvent
import com.google.android.gms.wearable.Wearable
import com.google.android.gms.wearable.WearableListenerService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

private const val TAG = "GkillWearSvc"

// Message paths (must match watch_app GkillWearClient)
private const val PATH_GET_TEMPLATES  = "/gkill/get_templates"
private const val PATH_TEMPLATES      = "/gkill/templates"
private const val PATH_SUBMIT              = "/gkill/submit"
private const val PATH_SUBMIT_RESULT       = "/gkill/submit_result"
private const val PATH_GET_PLAING_TIMEIS   = "/gkill/get_plaing_timeis"
private const val PATH_PLAING_TIMEIS       = "/gkill/plaing_timeis"
private const val PATH_END_TIMEIS          = "/gkill/end_timeis"
private const val PATH_END_TIMEIS_RESULT   = "/gkill/end_timeis_result"

/**
 * Runs on the phone, listens for messages from the watch.
 * - /gkill/get_templates → fetches KFTL template JSON from gkill server, sends back
 * - /gkill/submit       → submits KFTL text to gkill server, sends result back
 */
class GkillWearableListenerService : WearableListenerService() {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    override fun onMessageReceived(event: MessageEvent) {
        Log.d(TAG, "onMessageReceived path=${event.path} sourceNode=${event.sourceNodeId}")
        val store = GkillCredentialStore(this)
        val apiClient = GkillApiClient(store.getServerUrl())

        scope.launch {
            when (event.path) {
                PATH_GET_TEMPLATES     -> handleGetTemplates(event, store, apiClient)
                PATH_SUBMIT            -> handleSubmit(event, store, apiClient)
                PATH_GET_PLAING_TIMEIS -> handleGetPlaingTimeis(event, store, apiClient)
                PATH_END_TIMEIS        -> handleEndTimeis(event, store, apiClient)
                else                   -> Log.w(TAG, "Unknown path: ${event.path}")
            }
        }
    }

    private suspend fun handleGetTemplates(
        event: MessageEvent,
        store: GkillCredentialStore,
        apiClient: GkillApiClient
    ) {
        val sessionId = getOrRefreshSessionId(store, apiClient)
        if (sessionId == null) {
            sendMessage(event.sourceNodeId, PATH_TEMPLATES, "ERROR:login_failed".toByteArray())
            return
        }
        val templatesJson = apiClient.getKftlTemplateStructJson(sessionId)
        if (templatesJson == null) {
            sendMessage(event.sourceNodeId, PATH_TEMPLATES, "ERROR:get_config_failed".toByteArray())
            return
        }
        sendMessage(event.sourceNodeId, PATH_TEMPLATES, templatesJson.toByteArray(Charsets.UTF_8))
    }

    private suspend fun handleSubmit(
        event: MessageEvent,
        store: GkillCredentialStore,
        apiClient: GkillApiClient
    ) {
        val kftlText = String(event.data, Charsets.UTF_8)
        Log.d(TAG, "handleSubmit kftlText=${kftlText.take(80)}")
        val sessionId = getOrRefreshSessionId(store, apiClient)
        if (sessionId == null) {
            Log.e(TAG, "handleSubmit: login failed")
            sendMessage(event.sourceNodeId, PATH_SUBMIT_RESULT, "ERROR:login_failed".toByteArray())
            return
        }
        val error = apiClient.submitKFTLText(sessionId, kftlText)
        val result = if (error == null) "OK" else "ERROR:$error"
        Log.d(TAG, "handleSubmit result=$result")
        sendMessage(event.sourceNodeId, PATH_SUBMIT_RESULT, result.toByteArray(Charsets.UTF_8))
    }

    private suspend fun handleGetPlaingTimeis(
        event: MessageEvent,
        store: GkillCredentialStore,
        apiClient: GkillApiClient
    ) {
        try {
            Log.d(TAG, "handleGetPlaingTimeis: start")
            val sessionId = getOrRefreshSessionId(store, apiClient)
            if (sessionId == null) {
                Log.e(TAG, "handleGetPlaingTimeis: login failed")
                sendMessage(event.sourceNodeId, PATH_PLAING_TIMEIS, "ERROR:login_failed".toByteArray())
                return
            }
            Log.d(TAG, "handleGetPlaingTimeis: sessionId obtained")
            val result = apiClient.getPlaingTimeis(sessionId)
            Log.d(TAG, "handleGetPlaingTimeis: result=${result?.take(100)}")
            if (result == null) {
                sendMessage(event.sourceNodeId, PATH_PLAING_TIMEIS, "ERROR:get_plaing_timeis_failed".toByteArray())
                return
            }
            sendMessage(event.sourceNodeId, PATH_PLAING_TIMEIS, result.toByteArray(Charsets.UTF_8))
        } catch (e: Exception) {
            Log.e(TAG, "handleGetPlaingTimeis: exception", e)
            sendMessage(event.sourceNodeId, PATH_PLAING_TIMEIS, "ERROR:${e.message}".toByteArray())
        }
    }

    private suspend fun handleEndTimeis(
        event: MessageEvent,
        store: GkillCredentialStore,
        apiClient: GkillApiClient
    ) {
        val data = String(event.data, Charsets.UTF_8)
        Log.d(TAG, "handleEndTimeis data=${data.take(80)}")

        // data format: "id\nrep_name"
        val parts = data.split("\n", limit = 2)
        val timeisId = parts.getOrNull(0) ?: ""
        val repName = parts.getOrNull(1) ?: ""

        if (timeisId.isEmpty()) {
            sendMessage(event.sourceNodeId, PATH_END_TIMEIS_RESULT, "ERROR:empty_timeis_id".toByteArray())
            return
        }

        val sessionId = getOrRefreshSessionId(store, apiClient)
        if (sessionId == null) {
            sendMessage(event.sourceNodeId, PATH_END_TIMEIS_RESULT, "ERROR:login_failed".toByteArray())
            return
        }

        val error = apiClient.endTimeis(sessionId, timeisId, repName)
        val result = if (error == null) "OK" else "ERROR:$error"
        Log.d(TAG, "handleEndTimeis result=$result")
        sendMessage(event.sourceNodeId, PATH_END_TIMEIS_RESULT, result.toByteArray(Charsets.UTF_8))
    }

    /**
     * Returns a valid session ID.
     * Tries the cached session first; if it fails, logs in again.
     */
    private fun getOrRefreshSessionId(
        store: GkillCredentialStore,
        apiClient: GkillApiClient
    ): String? {
        val cached = store.getSessionId()
        if (cached.isNotEmpty()) {
            // Quick check: try to get config; if it fails, re-login
            val test = apiClient.getKftlTemplateStructJson(cached)
            if (test != null) return cached
        }
        // Re-login
        val newSession = apiClient.login(store.getUserId(), store.getPasswordSha256())
        if (newSession != null) {
            store.setSessionId(newSession)
        }
        return newSession
    }

    private suspend fun sendMessage(nodeId: String, path: String, data: ByteArray) {
        try {
            Wearable.getMessageClient(this).sendMessage(nodeId, path, data).await()
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send message path=$path to node=$nodeId", e)
        }
    }

    override fun onDestroy() {
        super.onDestroy()
        scope.cancel()
    }
}
