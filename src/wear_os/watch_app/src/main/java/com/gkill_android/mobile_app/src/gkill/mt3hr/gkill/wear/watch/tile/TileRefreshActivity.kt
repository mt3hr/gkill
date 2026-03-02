package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.lifecycle.lifecycleScope
import androidx.wear.tiles.TileService
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.GkillWearClient
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens.LoadingScreen
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.theme.GkillWearTheme
import com.google.android.gms.wearable.MessageEvent
import com.google.android.gms.wearable.Wearable
import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout

class TileRefreshActivity : ComponentActivity(), com.google.android.gms.wearable.MessageClient.OnMessageReceivedListener {

    private val wearClient by lazy { GkillWearClient(this) }
    private var pendingDeferred: CompletableDeferred<String>? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent { GkillWearTheme { LoadingScreen("テンプレートを更新中...") } }

        lifecycleScope.launch {
            val sent = wearClient.sendGetTemplatesRequest()
            if (sent == null) { finish(); return@launch }

            val deferred = CompletableDeferred<String>()
            pendingDeferred = deferred

            try {
                val json = withTimeout(20_000L) { deferred.await() }
                if (!json.startsWith("ERROR:")) {
                    TemplateCacheManager.saveRawJson(this@TileRefreshActivity, json)
                    TileService.getUpdater(this@TileRefreshActivity)
                        .requestUpdate(GkillTileService::class.java)
                }
            } catch (_: TimeoutCancellationException) {
                // タイムアウト → 何もせず終了
            } finally {
                finish()
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

    override fun onMessageReceived(event: MessageEvent) {
        if (event.path == GkillWearClient.RESPONSE_PATH_TEMPLATES) {
            pendingDeferred?.complete(String(event.data, Charsets.UTF_8))
            pendingDeferred = null
        }
    }
}
