package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.wear.compose.foundation.lazy.ScalingLazyColumn
import androidx.wear.compose.foundation.lazy.items
import androidx.wear.compose.material.CompactChip
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.R
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.PlayingTimeIsNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.components.MenuChip
import kotlinx.coroutines.delay
import java.time.OffsetDateTime
import java.time.format.DateTimeFormatter
import java.util.Locale

/**
 * 再生中のTimeIs一覧を表示する画面。
 * 各アイテムにタイトル・開始時刻・経過時間を表示し、タップで終了確認へ遷移する。
 */
@Composable
fun PlayingTimeIsListScreen(
    nodes: List<PlayingTimeIsNode>,
    onNodeSelected: (PlayingTimeIsNode) -> Unit,
    onRefresh: () -> Unit
) {
    // 毎秒更新用のtick
    var tickSeconds by remember { mutableLongStateOf(System.currentTimeMillis() / 1000) }
    LaunchedEffect(Unit) {
        while (true) {
            delay(1000)
            tickSeconds = System.currentTimeMillis() / 1000
        }
    }

    ScalingLazyColumn(modifier = Modifier.fillMaxSize()) {
        item {
            Text(
                text = stringResource(R.string.playing_title),
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 4.dp)
            )
        }

        if (nodes.isEmpty()) {
            item {
                Text(
                    text = stringResource(R.string.playing_empty),
                    textAlign = TextAlign.Center,
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 8.dp)
                )
            }
        } else {
            items(nodes) { node ->
                val startLabel = formatStartTime(node.start_time)
                val elapsed = formatElapsed(node.start_time, tickSeconds)
                // タイルと同じ見た目（140dp・中央揃え）。ScalingLazyColumn の既定で中央に並ぶ
                MenuChip(
                    label = node.title.ifEmpty { node.id.take(8) },
                    secondaryLabel = "$startLabel  $elapsed",
                    onClick = { onNodeSelected(node) }
                )
            }
        }

        item {
            CompactChip(
                label = { Text(stringResource(R.string.refresh)) },
                onClick = onRefresh
            )
        }
    }
}

private fun formatStartTime(isoTime: String): String {
    return try {
        val odt = OffsetDateTime.parse(isoTime, DateTimeFormatter.ISO_OFFSET_DATE_TIME)
        odt.format(DateTimeFormatter.ofPattern("HH:mm"))
    } catch (_: Exception) {
        try {
            // Try without offset (LocalDateTime format)
            val ldt = java.time.LocalDateTime.parse(isoTime, DateTimeFormatter.ISO_LOCAL_DATE_TIME)
            ldt.format(DateTimeFormatter.ofPattern("HH:mm"))
        } catch (_: Exception) {
            isoTime.take(5)
        }
    }
}

private fun formatElapsed(isoTime: String, tickSeconds: Long): String {
    // tickSeconds is used to trigger recomposition every second
    val nowMs = tickSeconds * 1000
    val startMs = try {
        OffsetDateTime.parse(isoTime, DateTimeFormatter.ISO_OFFSET_DATE_TIME)
            .toInstant().toEpochMilli()
    } catch (_: Exception) {
        try {
            java.time.LocalDateTime.parse(isoTime, DateTimeFormatter.ISO_LOCAL_DATE_TIME)
                .atZone(java.time.ZoneId.systemDefault())
                .toInstant().toEpochMilli()
        } catch (_: Exception) {
            return ""
        }
    }
    val diffSec = (nowMs - startMs) / 1000
    if (diffSec < 0) return ""
    val h = diffSec / 3600
    val m = (diffSec % 3600) / 60
    val s = diffSec % 60
    // 経過時間の桁は言語に依存させない（既定ロケールだと桁が現地化されうる）
    return if (h > 0) String.format(Locale.ROOT, "%d:%02d:%02d", h, m, s) else String.format(Locale.ROOT, "%d:%02d", m, s)
}
