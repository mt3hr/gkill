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
private const val PATH_SUBMIT         = "/gkill/submit"
private const val PATH_SUBMIT_RESULT  = "/gkill/submit_result"

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
                PATH_GET_TEMPLATES -> handleGetTemplates(event, store, apiClient)
                PATH_SUBMIT        -> handleSubmit(event, store, apiClient)
                else               -> Log.w(TAG, "Unknown path: ${event.path}")
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
