package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch

import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.wear.compose.material.Chip
import androidx.wear.compose.material.ChipDefaults
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.GkillWearClient
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.PlaingTimeIsNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile.TemplateCacheManager
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.ConfirmScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.LoadingScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.PlaingEndConfirmScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.PlaingTimeIsListScreen
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
private const val PLAING_TIMEOUT_MS = 20_000L
private const val END_TIMEIS_TIMEOUT_MS = 30_000L

const val EXTRA_MODE = "mode"
const val MODE_RECORD = "record"
const val MODE_PLAING = "plaing"

private sealed class Screen {
    object HomeMenu : Screen()
    object Loading : Screen()
    data class TemplateList(
        val nodes: List<TemplateNode>,
        val title: String,
        val breadcrumb: List<Pair<String, List<TemplateNode>>>
    ) : Screen()
    data class Confirm(val node: TemplateNode, val parentList: TemplateList) : Screen()
    data class Submitting(val node: TemplateNode) : Screen()
    data class Result(val success: Boolean, val error: String) : Screen()
    // Plaing screens
    object PlaingLoading : Screen()
    data class PlaingList(val nodes: List<PlaingTimeIsNode>) : Screen()
    data class PlaingEndConfirm(val node: PlaingTimeIsNode) : Screen()
    data class PlaingEnding(val node: PlaingTimeIsNode) : Screen()
}

class MainActivity : ComponentActivity(), MessageClient.OnMessageReceivedListener {

    private lateinit var wearClient: GkillWearClient

    private var launchedFromTile = false
    private var screenState by mutableStateOf<Screen>(Screen.HomeMenu)

    // CompletableDeferred for awaiting phone responses with timeout
    private var pendingTemplatesDeferred: CompletableDeferred<String>? = null
    private var pendingSubmitDeferred: CompletableDeferred<String>? = null
    private var pendingPlaingTimeisDeferred: CompletableDeferred<String>? = null
    private var pendingEndTimeisDeferred: CompletableDeferred<String>? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        wearClient = GkillWearClient(this)

        // Check intent extra for direct mode navigation (tile sets EXTRA_MODE)
        val mode = intent?.getStringExtra(EXTRA_MODE)
        launchedFromTile = (mode != null)
        screenState = when (mode) {
            MODE_RECORD -> Screen.Loading
            MODE_PLAING -> Screen.PlaingLoading
            else -> Screen.HomeMenu
        }

        setContent {
            GkillWearTheme {
                when (val s = screenState) {
                    is Screen.HomeMenu -> {
                        HomeMenuScreen(
                            onRecord = { screenState = Screen.Loading },
                            onPlaing = { screenState = Screen.PlaingLoading }
                        )
                    }

                    is Screen.Loading -> {
                        LoadingScreen()
                        LaunchedEffect(Unit) {
                            requestTemplates()
                        }
                    }

                    is Screen.TemplateList -> {
                        BackHandler {
                            if (s.breadcrumb.isNotEmpty()) {
                                val (prevTitle, prevNodes) = s.breadcrumb.last()
                                screenState = Screen.TemplateList(
                                    nodes = prevNodes,
                                    title = prevTitle,
                                    breadcrumb = s.breadcrumb.dropLast(1)
                                )
                            } else {
                                navigateBackToTopOrFinish()
                            }
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
                        BackHandler {
                            navigateBackToTopOrFinish()
                        }
                        ResultScreen(
                            success = s.success,
                            errorMessage = s.error,
                            onDismiss = {
                                navigateBackToTopOrFinish()
                            }
                        )
                    }

                    // ─── Plaing screens ─────────────────────────────────────
                    is Screen.PlaingLoading -> {
                        LoadingScreen("実行中を取得中...")
                        LaunchedEffect(Unit) {
                            requestPlaingTimeis()
                        }
                    }

                    is Screen.PlaingList -> {
                        BackHandler {
                            navigateBackToTopOrFinish()
                        }
                        PlaingTimeIsListScreen(
                            nodes = s.nodes,
                            onNodeSelected = { node ->
                                screenState = Screen.PlaingEndConfirm(node)
                            },
                            onRefresh = {
                                screenState = Screen.PlaingLoading
                            }
                        )
                    }

                    is Screen.PlaingEndConfirm -> {
                        BackHandler {
                            screenState = Screen.PlaingList(
                                // Go back to the list; we need to re-fetch or keep state
                                // For simplicity, go to PlaingLoading to refresh
                                emptyList()
                            )
                            screenState = Screen.PlaingLoading
                        }
                        PlaingEndConfirmScreen(
                            title = s.node.title.ifEmpty { s.node.id.take(8) },
                            startTime = s.node.start_time,
                            onConfirm = {
                                screenState = Screen.PlaingEnding(s.node)
                            },
                            onCancel = {
                                screenState = Screen.PlaingLoading
                            }
                        )
                    }

                    is Screen.PlaingEnding -> {
                        LoadingScreen("終了中...")
                        LaunchedEffect(s.node) {
                            endTimeis(s.node)
                        }
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

    private fun navigateBackToTopOrFinish() {
        if (launchedFromTile) {
            finish()
        } else {
            screenState = Screen.HomeMenu
        }
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
            GkillWearClient.RESPONSE_PATH_PLAING_TIMEIS -> {
                pendingPlaingTimeisDeferred?.complete(data)
                pendingPlaingTimeisDeferred = null
            }
            GkillWearClient.RESPONSE_PATH_END_TIMEIS_RESULT -> {
                pendingEndTimeisDeferred?.complete(data)
                pendingEndTimeisDeferred = null
            }
        }
    }

    // ─── Private helpers (record) ───────────────────────────────────────────────

    private suspend fun requestTemplates() {
        Log.i(TAG, "requestTemplates: start")

        // キャッシュがある場合はキャッシュを使用
        val cached = TemplateCacheManager.loadTemplates(this)
        if (cached.isNotEmpty()) {
            Log.i(TAG, "requestTemplates: using cache (${cached.size} root nodes)")
            screenState = Screen.TemplateList(nodes = cached, title = "テンプレート", breadcrumb = emptyList())
            return
        }

        // キャッシュがない場合のみスマホから取得
        Log.i(TAG, "requestTemplates: no cache, fetching from phone")
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

    // ─── Private helpers (plaing) ───────────────────────────────────────────────

    private suspend fun requestPlaingTimeis() {
        Log.i(TAG, "requestPlaingTimeis: start")

        val sent = wearClient.sendGetPlaingTimeisRequest()
        if (sent == null) {
            Log.e(TAG, "requestPlaingTimeis: no phone node found")
            screenState = Screen.Result(
                success = false,
                error = "スマホに接続できません。\nPixel Watch 2とスマホのペアリングを確認してください。"
            )
            return
        }
        Log.i(TAG, "requestPlaingTimeis: message sent to $sent, waiting...")

        val deferred = CompletableDeferred<String>()
        pendingPlaingTimeisDeferred = deferred

        val json = try {
            withTimeout(PLAING_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.e(TAG, "requestPlaingTimeis: timeout after ${PLAING_TIMEOUT_MS}ms")
            pendingPlaingTimeisDeferred = null
            screenState = Screen.Result(
                success = false,
                error = "スマホからの応答がタイムアウトしました。"
            )
            return
        }

        Log.i(TAG, "requestPlaingTimeis: received (${json.take(60)})")
        if (json.startsWith("ERROR:")) {
            screenState = Screen.Result(success = false, error = json.removePrefix("ERROR:"))
            return
        }

        val nodes = GkillWearClient.parsePlaingTimeisList(json)
        screenState = Screen.PlaingList(nodes = nodes)
    }

    private suspend fun endTimeis(node: PlaingTimeIsNode) {
        Log.i(TAG, "endTimeis: ${node.id}")
        // Send "id\nrep_name" format
        val payload = "${node.id}\n${node.rep_name}"
        val sent = wearClient.sendEndTimeisRequest(payload)
        if (sent == null) {
            Log.e(TAG, "endTimeis: no phone node found")
            screenState = Screen.Result(success = false, error = "スマホに接続できません。")
            return
        }

        val deferred = CompletableDeferred<String>()
        pendingEndTimeisDeferred = deferred

        val result = try {
            withTimeout(END_TIMEIS_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.e(TAG, "endTimeis: timeout")
            pendingEndTimeisDeferred = null
            screenState = Screen.Result(
                success = false,
                error = "終了タイムアウト。スマホの状態を確認してください。"
            )
            return
        }

        if (result == "OK") {
            // 終了成功 → 一覧を再取得
            screenState = Screen.PlaingLoading
        } else {
            screenState = Screen.Result(success = false, error = result.removePrefix("ERROR:"))
        }
    }
}

/**
 * トップメニュー画面。「記録する」「再生中」の2つの選択肢を表示する。
 */
@Composable
private fun HomeMenuScreen(
    onRecord: () -> Unit,
    onPlaing: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = "gkill",
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 12.dp)
        )
        Chip(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            label = { Text("📝 記録する") },
            onClick = onRecord,
            colors = ChipDefaults.primaryChipColors()
        )
        Chip(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 8.dp, vertical = 4.dp),
            label = { Text("▶ 実行中") },
            onClick = onPlaing,
            colors = ChipDefaults.secondaryChipColors()
        )
    }
}
