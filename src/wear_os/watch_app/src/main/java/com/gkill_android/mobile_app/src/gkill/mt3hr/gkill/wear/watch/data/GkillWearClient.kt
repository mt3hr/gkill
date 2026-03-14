package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data

import android.content.Context
import android.util.Log
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.PlaingTimeIsNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode
import com.google.android.gms.wearable.CapabilityClient
import com.google.android.gms.wearable.Wearable
import kotlinx.coroutines.tasks.await
import kotlinx.serialization.json.Json

private const val TAG = "GkillWearClient"

// Message paths (must match phone_companion GkillWearableListenerService)
private const val PATH_GET_TEMPLATES = "/gkill/get_templates"
private const val PATH_TEMPLATES     = "/gkill/templates"
private const val PATH_SUBMIT              = "/gkill/submit"
private const val PATH_SUBMIT_RESULT       = "/gkill/submit_result"
private const val PATH_GET_PLAING_TIMEIS   = "/gkill/get_plaing_timeis"
private const val PATH_PLAING_TIMEIS       = "/gkill/plaing_timeis"
private const val PATH_END_TIMEIS          = "/gkill/end_timeis"
private const val PATH_END_TIMEIS_RESULT   = "/gkill/end_timeis_result"

private val json = Json { ignoreUnknownKeys = true }

/**
 * Communicates with the Phone Companion app via Wearable MessageClient.
 */
class GkillWearClient(private val context: Context) {

    private val messageClient get() = Wearable.getMessageClient(context)
    private val nodeClient get() = Wearable.getNodeClient(context)

    /**
     * Returns the ID of the connected phone node, or null if none found.
     */
    private suspend fun getPhoneNodeId(): String? {
        return try {
            val nodes = nodeClient.connectedNodes.await()
            nodes.firstOrNull { it.isNearby }?.id ?: nodes.firstOrNull()?.id
        } catch (e: Exception) {
            Log.e(TAG, "Failed to get connected nodes", e)
            null
        }
    }

    /**
     * Fetches KFTL templates from the phone companion.
     * Returns a list of root TemplateNode, or throws on error.
     */
    suspend fun fetchTemplates(): List<TemplateNode> {
        val nodeId = getPhoneNodeId()
            ?: throw IllegalStateException("スマホが接続されていません")

        messageClient.sendMessage(nodeId, PATH_GET_TEMPLATES, ByteArray(0)).await()

        // Wait for response via MessageListener (handled in MainActivity via ChannelHelper)
        // This is handled by registering a one-shot MessageListener in the ViewModel.
        // The actual waiting is in WatchViewModel using a CompletableDeferred.
        throw UnsupportedOperationException("Use WatchViewModel.fetchTemplates() instead")
    }

    /**
     * Sends a message to the phone to get templates.
     * The response will arrive via onMessageReceived in the caller.
     */
    suspend fun sendGetTemplatesRequest(): String? {
        val nodeId = getPhoneNodeId() ?: return null
        return try {
            messageClient.sendMessage(nodeId, PATH_GET_TEMPLATES, ByteArray(0)).await()
            nodeId
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send get_templates request", e)
            null
        }
    }

    /**
     * Sends a request to get the list of currently playing TimeIs.
     * Returns the nodeId if sent successfully, null otherwise.
     */
    suspend fun sendGetPlaingTimeisRequest(): String? {
        val nodeId = getPhoneNodeId() ?: return null
        return try {
            messageClient.sendMessage(nodeId, PATH_GET_PLAING_TIMEIS, ByteArray(0)).await()
            nodeId
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send get_plaing_timeis request", e)
            null
        }
    }

    /**
     * Sends an end-timeis request to the phone.
     * Returns the nodeId if sent successfully, null otherwise.
     */
    suspend fun sendEndTimeisRequest(timeisJson: String): String? {
        val nodeId = getPhoneNodeId() ?: return null
        return try {
            messageClient.sendMessage(nodeId, PATH_END_TIMEIS, timeisJson.toByteArray(Charsets.UTF_8)).await()
            nodeId
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send end_timeis request", e)
            null
        }
    }

    /**
     * Sends a KFTL text submission request to the phone.
     * Returns the nodeId if sent successfully, null otherwise.
     */
    suspend fun sendSubmitRequest(kftlText: String): String? {
        val nodeId = getPhoneNodeId() ?: return null
        return try {
            messageClient.sendMessage(nodeId, PATH_SUBMIT, kftlText.toByteArray(Charsets.UTF_8)).await()
            nodeId
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send submit request", e)
            null
        }
    }

    companion object {
        const val RESPONSE_PATH_TEMPLATES          = PATH_TEMPLATES
        const val RESPONSE_PATH_SUBMIT_RESULT      = PATH_SUBMIT_RESULT
        const val RESPONSE_PATH_PLAING_TIMEIS      = PATH_PLAING_TIMEIS
        const val RESPONSE_PATH_END_TIMEIS_RESULT  = PATH_END_TIMEIS_RESULT

        fun parsePlaingTimeisList(jsonStr: String): List<PlaingTimeIsNode> {
            return try {
                if (jsonStr.startsWith("ERROR:")) return emptyList()
                json.decodeFromString<List<PlaingTimeIsNode>>(jsonStr)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to parse plaing timeis JSON: ${e.message}", e)
                emptyList()
            }
        }

        fun parseTemplates(json_str: String): List<TemplateNode> {
            return try {
                if (json_str.startsWith("ERROR:")) return emptyList()
                val root = json.decodeFromString<TemplateNode>(json_str)
                filterNodes(root.children ?: emptyList())
            } catch (e: Exception) {
                Log.e(TAG, "Failed to parse templates JSON: ${e.message}", e)
                emptyList()
            }
        }

        private fun filterNodes(nodes: List<TemplateNode>): List<TemplateNode> {
            return nodes.mapNotNull { node ->
                if (node.is_dir) {
                    val filteredChildren = filterNodes(node.children ?: emptyList())
                    if (filteredChildren.isEmpty()) null
                    else node.copy(children = filteredChildren)
                } else {
                    if (node.template.trimEnd('\n').endsWith("！")) node else null
                }
            }
        }
    }
}
