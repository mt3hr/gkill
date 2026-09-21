package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch

import android.os.Bundle
import android.util.Log
import androidx.activity.ComponentActivity
import androidx.activity.compose.BackHandler
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Box
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
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.wear.compose.foundation.lazy.AutoCenteringParams
import androidx.wear.compose.foundation.lazy.ScalingLazyColumn
import androidx.wear.compose.foundation.lazy.rememberScalingLazyListState
import androidx.wear.compose.material.Chip
import androidx.wear.compose.material.ChipDefaults
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.GkillWearClient
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.buildLantanaKftlText
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.PlayingTimeIsNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile.TemplateCacheManager
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.components.MenuChip
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.ConfirmScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.LantanaConfirmScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.LantanaSelectScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.LoadingScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.PlayingEndConfirmScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.PlayingTimeIsListScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.ResultScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.TemplateListScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.theme.GkillWearTheme
import com.google.android.gms.wearable.MessageClient
import com.google.android.gms.wearable.MessageEvent
import com.google.android.gms.wearable.Wearable
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout
import java.time.LocalDateTime

private const val TAG = "GkillWatchMain"
private const val TEMPLATE_TIMEOUT_MS = 20_000L
private const val SUBMIT_TIMEOUT_MS = 30_000L
private const val PLAYING_TIMEOUT_MS = 20_000L
private const val END_TIMEIS_TIMEOUT_MS = 30_000L

const val EXTRA_MODE = "mode"
const val MODE_RECORD = "record"
const val MODE_PLAYING = "playing"
const val MODE_LANTANA = "lantana"

private sealed class Screen {
    object HomeMenu : Screen()

    /**
     * テンプレート取得中。
     * force_reload=true なら一覧の「🔄 更新」経由なので、キャッシュを無視して取り直す。
     */
    data class Loading(val force_reload: Boolean = false) : Screen()
    data class TemplateList(
        val nodes: List<TemplateNode>,
        val title: String,
        val breadcrumb: List<Pair<String, List<TemplateNode>>>
    ) : Screen()
    data class Confirm(val node: TemplateNode, val parentList: TemplateList) : Screen()

    /**
     * 送信中。テンプレートと気分記録で経路を1本に保つため、
     * ここから下は「どの画面から来たか」ではなく **KFTL テキスト**だけを持つ。
     */
    data class Submitting(val kftlText: String, val force: Boolean = false) : Screen()

    /**
     * スマホから DUPLICATE が返ったとき（直前に同じ内容を保存済み）の確認。
     * 「それでも送信」で force 付きの再送を行う。
     */
    data class SubmitDuplicateConfirm(val kftlText: String) : Screen()
    data class Result(val success: Boolean, val error: String) : Screen()
    // Playing screens
    object PlayingLoading : Screen()
    data class PlayingList(val nodes: List<PlayingTimeIsNode>) : Screen()
    data class PlayingEndConfirm(val node: PlayingTimeIsNode) : Screen()
    data class PlayingEnding(val node: PlayingTimeIsNode) : Screen()
    // Lantana (mood) screens
    data class LantanaSelect(val mood: Int) : Screen()
    data class LantanaConfirm(val mood: Int) : Screen()
}

class MainActivity : ComponentActivity(), MessageClient.OnMessageReceivedListener {

    private lateinit var wearClient: GkillWearClient

    private var launchedFromTile = false
    private var screenState by mutableStateOf<Screen>(Screen.HomeMenu)

    // CompletableDeferred for awaiting phone responses with timeout
    private var pendingTemplatesDeferred: CompletableDeferred<String>? = null
    private var pendingSubmitDeferred: CompletableDeferred<String>? = null
    private var pendingPlayingTimeisDeferred: CompletableDeferred<String>? = null
    private var pendingEndTimeisDeferred: CompletableDeferred<String>? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        wearClient = GkillWearClient(this)

        // リスナーは onResume/onPause ではなく onCreate/onDestroy に張る。
        // ambient への遷移などで onPause が起きても、スマホからの応答を取りこぼさないため。
        Wearable.getMessageClient(this).addListener(this)

        // Check intent extra for direct mode navigation (tile sets EXTRA_MODE)
        val mode = intent?.getStringExtra(EXTRA_MODE)
        launchedFromTile = (mode != null)
        screenState = when (mode) {
            MODE_RECORD -> Screen.Loading()
            MODE_PLAYING -> Screen.PlayingLoading
            MODE_LANTANA -> Screen.LantanaSelect(0)
            else -> Screen.HomeMenu
        }

        setContent {
            GkillWearTheme {
                when (val s = screenState) {
                    is Screen.HomeMenu -> {
                        HomeMenuScreen(
                            onRecord = { screenState = Screen.Loading() },
                            onPlaying = { screenState = Screen.PlayingLoading },
                            onLantana = { screenState = Screen.LantanaSelect(0) }
                        )
                    }

                    is Screen.Loading -> {
                        LoadingScreen(
                            stringResource(if (s.force_reload) R.string.loading_refresh_templates else R.string.loading)
                        )
                        LaunchedEffect(s.force_reload) {
                            requestTemplates(s.force_reload)
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
                            },
                            onRefresh = {
                                screenState = Screen.Loading(force_reload = true)
                            }
                        )
                    }

                    is Screen.Confirm -> {
                        val label = if (s.node.title.isNotEmpty()) s.node.title else s.node.name
                        ConfirmScreen(
                            templateTitle = label,
                            onConfirm = {
                                screenState = Screen.Submitting(s.node.template)
                            },
                            onCancel = {
                                screenState = s.parentList
                            }
                        )
                    }

                    is Screen.Submitting -> {
                        LoadingScreen(stringResource(R.string.loading_submit))
                        LaunchedEffect(s.kftlText, s.force) {
                            submitKftl(s.kftlText, s.force)
                        }
                    }

                    is Screen.SubmitDuplicateConfirm -> {
                        BackHandler {
                            navigateBackToTopOrFinish()
                        }
                        DuplicateConfirmScreen(
                            onConfirm = {
                                screenState = Screen.Submitting(s.kftlText, force = true)
                            },
                            onCancel = {
                                navigateBackToTopOrFinish()
                            }
                        )
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

                    // ─── Playing screens ─────────────────────────────────────
                    is Screen.PlayingLoading -> {
                        LoadingScreen(stringResource(R.string.loading_playing))
                        LaunchedEffect(Unit) {
                            requestPlayingTimeis()
                        }
                    }

                    is Screen.PlayingList -> {
                        BackHandler {
                            navigateBackToTopOrFinish()
                        }
                        PlayingTimeIsListScreen(
                            nodes = s.nodes,
                            onNodeSelected = { node ->
                                screenState = Screen.PlayingEndConfirm(node)
                            },
                            onRefresh = {
                                screenState = Screen.PlayingLoading
                            }
                        )
                    }

                    is Screen.PlayingEndConfirm -> {
                        BackHandler {
                            screenState = Screen.PlayingList(
                                // Go back to the list; we need to re-fetch or keep state
                                // For simplicity, go to PlayingLoading to refresh
                                emptyList()
                            )
                            screenState = Screen.PlayingLoading
                        }
                        PlayingEndConfirmScreen(
                            title = s.node.title.ifEmpty { s.node.id.take(8) },
                            startTime = s.node.start_time,
                            onConfirm = {
                                screenState = Screen.PlayingEnding(s.node)
                            },
                            onCancel = {
                                screenState = Screen.PlayingLoading
                            }
                        )
                    }

                    is Screen.PlayingEnding -> {
                        LoadingScreen(stringResource(R.string.loading_end_timeis))
                        LaunchedEffect(s.node) {
                            endTimeis(s.node)
                        }
                    }

                    // ─── Lantana (mood) screens ──────────────────────────────
                    is Screen.LantanaSelect -> {
                        BackHandler {
                            navigateBackToTopOrFinish()
                        }
                        LantanaSelectScreen(
                            mood = s.mood,
                            onMoodSelected = { mood ->
                                screenState = Screen.LantanaConfirm(mood)
                            }
                        )
                    }

                    is Screen.LantanaConfirm -> {
                        BackHandler {
                            screenState = Screen.LantanaSelect(s.mood)
                        }
                        LantanaConfirmScreen(
                            mood = s.mood,
                            onConfirm = {
                                // 関連時刻は「✓ を押した瞬間」。送信が WorkManager で後回しに
                                // なっても、タップした時刻が related_time として残る
                                screenState = Screen.Submitting(
                                    buildLantanaKftlText(s.mood, LocalDateTime.now())
                                )
                            },
                            onCancel = {
                                screenState = Screen.LantanaSelect(s.mood)
                            }
                        )
                    }
                }
            }
        }
    }

    override fun onDestroy() {
        super.onDestroy()
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
        Log.d(TAG, "onMessageReceived path=${event.path}")
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
            GkillWearClient.RESPONSE_PATH_PLAYING_TIMEIS -> {
                pendingPlayingTimeisDeferred?.complete(data)
                pendingPlayingTimeisDeferred = null
            }
            GkillWearClient.RESPONSE_PATH_END_TIMEIS_RESULT -> {
                pendingEndTimeisDeferred?.complete(data)
                pendingEndTimeisDeferred = null
            }
        }
    }

    // ─── Private helpers (record) ───────────────────────────────────────────────

    private suspend fun requestTemplates(force_reload: Boolean = false) {
        Log.d(TAG, "requestTemplates: start (force_reload=$force_reload)")

        // 「🔄 更新」で来たとき以外は、キャッシュがあればそれを使う
        val cached = TemplateCacheManager.loadTemplates(this)
        if (!TemplateCacheManager.shouldFetchFromPhone(force_reload, cached.size)) {
            Log.d(TAG, "requestTemplates: using cache (${cached.size} root nodes)")
            screenState = Screen.TemplateList(nodes = cached, title = getString(R.string.template_list_title), breadcrumb = emptyList())
            return
        }

        // ここで screenState を書き換えてはいけない。
        // すでに Screen.Loading の描画中で、force_reload の異なる Loading を代入すると
        // LaunchedEffect が張り直されて取得が繰り返される。
        Log.d(TAG, "requestTemplates: fetching from phone")

        val sent = wearClient.sendGetTemplatesRequest()
        if (sent == null) {
            Log.w(TAG, "requestTemplates: no phone node found")
            useCacheOrError(getString(R.string.error_phone_not_connected_check_pairing))
            return
        }
        Log.d(TAG, "requestTemplates: message sent to $sent, waiting...")

        val deferred = CompletableDeferred<String>()
        pendingTemplatesDeferred = deferred

        val json = try {
            withTimeout(TEMPLATE_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.w(TAG, "requestTemplates: timeout after ${TEMPLATE_TIMEOUT_MS}ms")
            pendingTemplatesDeferred = null
            useCacheOrError(getString(R.string.error_templates_timeout))
            return
        }

        Log.d(TAG, "requestTemplates: received ${json.length} chars")
        if (json.startsWith("ERROR:")) {
            // 「🔄 更新」が失敗しただけで手元の一覧を捨てないよう、キャッシュがあればそれを残す
            useCacheOrError(json.removePrefix("ERROR:"))
            return
        }

        // スマホから取得成功 → キャッシュを更新
        TemplateCacheManager.saveRawJson(this, json)

        val nodes = GkillWearClient.parseTemplates(json)
        screenState = Screen.TemplateList(nodes = nodes, title = getString(R.string.template_list_title), breadcrumb = emptyList())
    }

    private fun useCacheOrError(fallbackErrorMsg: String) {
        val cached = TemplateCacheManager.loadTemplates(this)
        if (cached.isNotEmpty()) {
            Log.i(TAG, "requestTemplates: falling back to cache (${cached.size} root nodes)")
            screenState = Screen.TemplateList(nodes = cached, title = getString(R.string.template_list_title_cached), breadcrumb = emptyList())
        } else {
            screenState = Screen.Result(success = false, error = fallbackErrorMsg)
        }
    }

    /**
     * KFTL テキストをスマホ経由でサーバーへ送る。テンプレート記録と気分記録の共通経路。
     */
    private suspend fun submitKftl(kftlText: String, force: Boolean = false) {
        // KFTL テキストは利用者の記録内容そのものなので logcat へ出さない(指摘 F-008)
        Log.d(TAG, "submitKftl: (force=$force)")
        val sent = wearClient.sendSubmitRequest(kftlText, force)
        if (sent == null) {
            Log.w(TAG, "submitKftl: no phone node found")
            screenState = Screen.Result(success = false, error = getString(R.string.error_phone_not_connected))
            return
        }

        val deferred = CompletableDeferred<String>()
        pendingSubmitDeferred = deferred

        val result = try {
            withTimeout(SUBMIT_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.w(TAG, "submitKftl: timeout")
            pendingSubmitDeferred = null
            screenState = Screen.Result(
                success = false,
                error = getString(R.string.error_submit_timeout)
            )
            return
        }

        when {
            result == "OK" -> {
                screenState = Screen.Result(success = true, error = "")
            }
            // 直前に同じ内容を保存済み。黙って捨てず「それでも送信」の確認を出す
            result == "DUPLICATE" -> {
                screenState = Screen.SubmitDuplicateConfirm(kftlText)
            }
            else -> {
                screenState = Screen.Result(success = false, error = result.removePrefix("ERROR:"))
            }
        }
    }

    // ─── Private helpers (playing) ───────────────────────────────────────────────

    private suspend fun requestPlayingTimeis() {
        Log.d(TAG, "requestPlayingTimeis: start")

        val sent = wearClient.sendGetPlayingTimeisRequest()
        if (sent == null) {
            Log.w(TAG, "requestPlayingTimeis: no phone node found")
            screenState = Screen.Result(
                success = false,
                error = getString(R.string.error_phone_not_connected_check_pairing)
            )
            return
        }
        Log.d(TAG, "requestPlayingTimeis: message sent to $sent, waiting...")

        val deferred = CompletableDeferred<String>()
        pendingPlayingTimeisDeferred = deferred

        val json = try {
            withTimeout(PLAYING_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.w(TAG, "requestPlayingTimeis: timeout after ${PLAYING_TIMEOUT_MS}ms")
            pendingPlayingTimeisDeferred = null
            screenState = Screen.Result(
                success = false,
                error = getString(R.string.error_playing_timeout)
            )
            return
        }

        Log.d(TAG, "requestPlayingTimeis: received ${json.length} chars")
        if (json.startsWith("ERROR:")) {
            screenState = Screen.Result(success = false, error = json.removePrefix("ERROR:"))
            return
        }

        val nodes = GkillWearClient.parsePlayingTimeisList(json)
        screenState = Screen.PlayingList(nodes = nodes)
    }

    private suspend fun endTimeis(node: PlayingTimeIsNode) {
        // Kyou ID は記録と突き合わせられる識別子なので logcat へ出さない(指摘 F-008)
        Log.d(TAG, "endTimeis")
        // Send "id\nrep_name" format
        val payload = "${node.id}\n${node.rep_name}"
        val sent = wearClient.sendEndTimeisRequest(payload)
        if (sent == null) {
            Log.w(TAG, "endTimeis: no phone node found")
            screenState = Screen.Result(success = false, error = getString(R.string.error_phone_not_connected))
            return
        }

        val deferred = CompletableDeferred<String>()
        pendingEndTimeisDeferred = deferred

        val result = try {
            withTimeout(END_TIMEIS_TIMEOUT_MS) { deferred.await() }
        } catch (e: TimeoutCancellationException) {
            Log.w(TAG, "endTimeis: timeout")
            pendingEndTimeisDeferred = null
            screenState = Screen.Result(
                success = false,
                error = getString(R.string.error_end_timeis_timeout)
            )
            return
        }

        if (result == "OK") {
            // 終了成功 → 一覧を再取得
            screenState = Screen.PlayingLoading
        } else {
            screenState = Screen.Result(success = false, error = result.removePrefix("ERROR:"))
        }
    }
}

/**
 * トップメニュー画面。「記録する」「実行中」「気分記録」の3つの選択肢を、
 * タイル（GkillTileService.buildLayout）と同じ配置で出す: MenuChip を隙間なく縦に3つ・画面中央・タイトル無し。
 * 文言もタイルと同じ R.string.home_*。
 *
 * ScalingLazyColumn にしないこと。端の項目が縮小・減光されてタイルと見た目が割れる。
 * 3チップ（52dp × 3）が丸画面に収まることは、タイルが同じ配置で収まっているのが根拠。
 */
@Composable
private fun HomeMenuScreen(
    onRecord: () -> Unit,
    onPlaying: () -> Unit,
    onLantana: () -> Unit
) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            MenuChip(label = stringResource(R.string.home_record), onClick = onRecord)
            MenuChip(label = stringResource(R.string.home_playing), onClick = onPlaying)
            MenuChip(label = stringResource(R.string.home_lantana), onClick = onLantana)
        }
    }
}

/**
 * 直前に同じ内容を保存済みのときの確認画面。「それでも送信」でforce再送する。
 *
 * 3行の本文＋チップ2つは 192dp の丸画面から余白を引いた高さに収まらないので、
 * 固定の Column ではなく ScalingLazyColumn でスクロールできるようにする
 * （固定だと下の「キャンセル」がエラーも出さずに画面外へ落ちる。2026-09-13 に実際に起きた）。
 * 最初は本文（item 0）を中央に置く。既定の item 1 中央だと本文の1行目が丸画面の上端で欠ける。
 * 「それでも送信」は本文の下に見え、「キャンセル」は一段スクロール（Back でもキャンセル扱い）。
 */
@Composable
private fun DuplicateConfirmScreen(
    onConfirm: () -> Unit,
    onCancel: () -> Unit
) {
    ScalingLazyColumn(
        modifier = Modifier.fillMaxSize(),
        state = rememberScalingLazyListState(initialCenterItemIndex = 0),
        autoCentering = AutoCenteringParams(itemIndex = 0)
    ) {
        item {
            Text(
                text = stringResource(R.string.duplicate_confirm_message),
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 4.dp)
            )
        }
        item {
            Chip(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 8.dp),
                label = {
                    Text(
                        text = stringResource(R.string.duplicate_send_anyway),
                        textAlign = TextAlign.Center,
                        modifier = Modifier.fillMaxWidth()
                    )
                },
                onClick = onConfirm,
                colors = ChipDefaults.primaryChipColors()
            )
        }
        item {
            Chip(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 8.dp),
                label = {
                    Text(
                        text = stringResource(R.string.cancel),
                        textAlign = TextAlign.Center,
                        modifier = Modifier.fillMaxWidth()
                    )
                },
                onClick = onCancel,
                colors = ChipDefaults.secondaryChipColors()
            )
        }
    }
}
