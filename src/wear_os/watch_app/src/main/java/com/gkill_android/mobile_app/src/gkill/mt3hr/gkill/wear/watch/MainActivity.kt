package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch

import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.GkillWearClient
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile.TemplateCacheManager
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.ConfirmScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.LoadingScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.ResultScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.TemplateListScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.theme.GkillWearTheme
import com.google.android.gms.wearable.MessageClient
import com.google.android.gms.wearable.MessageEvent
import com.google.android.gms.wearable.Wearable
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout

private const val TAG = "GkillWatchMain"
private const val TEMPLATE_TIMEOUT_MS = 20_000L
private const val SUBMIT_TIMEOUT_MS = 30_000L

private sealed class Screen {
    object Loading : Screen()
    data class TemplateList(
        val nodes: List<TemplateNode>,
        val title: String,
        val breadcrumb: List<Pair<String, List<TemplateNode>>>
    ) : Screen()
    data class Confirm(val node: TemplateNode, val parentList: TemplateList) : Screen()
    data class Submitting(val node: TemplateNode) : Screen()
    data class Result(val success: Boolean, val error: String) : Screen()
}

class MainActivity : ComponentActivity(), MessageClient.OnMessageReceivedListener {

    private lateinit var wearClient: GkillWearClient

    private var screenState by mutableStateOf<Screen>(Screen.Loading)

    // CompletableDeferred for awaiting phone responses with timeout
    private var pendingTemplatesDeferred: CompletableDeferred<String>? = null
    private var pendingSubmitDeferred: CompletableDeferred<String>? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        wearClient = GkillWearClient(this)

        setContent {
            GkillWearTheme {
                when (val s = screenState) {
                    is Screen.Loading -> {
                        LoadingScreen()
                        // LaunchedEffect(Unit) fires once per composition entry — correct Compose pattern
                        LaunchedEffect(Unit) {
                            requestTemplates()
                        }
                    }

                    is Screen.TemplateList -> {
                        BackHandler(enabled = s.breadcrumb.isNotEmpty()) {
                            val (prevTitle, prevNodes) = s.breadcrumb.last()
                            screenState = Screen.TemplateList(
                                nodes = prevNodes,
                                title = prevTitle,
                                breadcrumb = s.breadcrumb.dropLast(1)
                            )
                        }
                        TemplateListScreen(
                            nodes = s.nodes,
                            title = s.title,
                            onNodeSelected = { node ->
                                if (node.is_dir) {
                                    val label = if (node.title.isNotEmpty()) node.title else node.name
                                    val newBreadcrumb = s.breadcrumb + listOf(Pair(s.title, s.nodes))
                                    screenState = Screen.TemplateList(
                                        nodes = node.children ?: emptyList(),
                                        title = label,
                                        breadcrumb = newBreadcrumb
                                    )
                                } else {
                                    screenState = Screen.Confirm(node, s)
                                }
                            }
                        )
                    }

                    is Screen.Confirm -> {
                        val label = if (s.node.title.isNotEmpty()) s.node.title else s.node.name
                        ConfirmScreen(
                            templateTitle = label,
                            onConfirm = {
                                screenState = Screen.Submitting(s.node)
                            },
                            onCancel = {
                                screenState = s.parentList
                            }
                        )
                    }

                    is Screen.Submitting -> {
                        LoadingScreen("送信中...")
                        LaunchedEffect(s.node) {
                            submitTemplate(s.node)
                        }
                    }

                    is Screen.Result -> {
                        ResultScreen(
                            success = s.success,
                            errorMessage = s.error,
                            onDismiss = {
                                screenState = Screen.Loading
                            }
                        )
                    }
                }
            }
        }
    }

    override fun onResume() {
        super.onResume()
        Wearable.getMessageClient(this).addListener(this)
    }

    override fun onPause() {
        super.onPause()
        Wearable.getMessageClient(this).removeListener(this)
    }

    // ─── MessageClient callback ────────────────────────────────────────────────

    override fun onMessageReceived(event: MessageEvent) {
        Log.i(TAG, "onMessageReceived path=${event.path}")
        val data = String(event.data, Charsets.UTF_8)
        when (event.path) {
            GkillWearClient.RESPONSE_PATH_TEMPLATES -> {
                pendingTemplatesDeferred?.complete(data)
                pendingTemplatesDeferred = null
            }
            GkillWearClient.RESPONSE_PATH_SUBMIT_RESULT -> {
                pendingSubmitDeferred?.complete(data)
                pendingSubmitDeferred = null
            }
        }
    }

    // ─── Private helpers ───────────────────────────────────────────────────────

    private suspend fun requestTemplates() {
        Log.i(TAG, "requestTemplates: start")
        screenState = Screen.Loading

        val sent = wearClient.sendGetTemplatesRequest()
        if (sent == null) {
            Log.e(TAG, "requestTemplates: no phone node found")
            useCacheOrError("スマホに接続できません。\nPixel Watch 2とスマホのペアリングを確認してください。")
            return
        }
        Log.i(TAG, "requestTemplates: message sent to $sent, waiting...")

        val deferred = CompletableDeferred<String>()
        pendingTemplatesDeferred = deferred

        val json = try {
            withTimeout(TEMPLATE_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.e(TAG, "requestTemplates: timeout after ${TEMPLATE_TIMEOUT_MS}ms")
            pendingTemplatesDeferred = null
            useCacheOrError("スマホからの応答がタイムアウトしました。\nスマホでgkill Wear設定アプリを開き、接続テストを行ってください。")
            return
        }

        Log.i(TAG, "requestTemplates: received (${json.take(60)})")
        if (json.startsWith("ERROR:")) {
            screenState = Screen.Result(success = false, error = json.removePrefix("ERROR:"))
            return
        }

        // スマホから取得成功 → キャッシュを更新
        TemplateCacheManager.saveRawJson(this, json)

        val nodes = GkillWearClient.parseTemplates(json)
        screenState = Screen.TemplateList(nodes = nodes, title = "テンプレート", breadcrumb = emptyList())
    }

    private fun useCacheOrError(fallbackErrorMsg: String) {
        val cached = TemplateCacheManager.loadTemplates(this)
        if (cached.isNotEmpty()) {
            Log.i(TAG, "requestTemplates: falling back to cache (${cached.size} root nodes)")
            screenState = Screen.TemplateList(nodes = cached, title = "テンプレート(キャッシュ)", breadcrumb = emptyList())
        } else {
            screenState = Screen.Result(success = false, error = fallbackErrorMsg)
        }
    }

    private suspend fun submitTemplate(node: TemplateNode) {
        Log.i(TAG, "submitTemplate: ${node.name}")
        val sent = wearClient.sendSubmitRequest(node.template)
        if (sent == null) {
            Log.e(TAG, "submitTemplate: no phone node found")
            screenState = Screen.Result(success = false, error = "スマホに接続できません。")
            return
        }

        val deferred = CompletableDeferred<String>()
        pendingSubmitDeferred = deferred

        val result = try {
            withTimeout(SUBMIT_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.e(TAG, "submitTemplate: timeout")
            pendingSubmitDeferred = null
            screenState = Screen.Result(
                success = false,
                error = "送信タイムアウト。スマホの状態を確認してください。"
            )
            return
        }

        if (result == "OK") {
            screenState = Screen.Result(success = true, error = "")
        } else {
            screenState = Screen.Result(success = false, error = result.removePrefix("ERROR:"))
        }
    }
}
