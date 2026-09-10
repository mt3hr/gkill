package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.wear.compose.material.Button
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.R
import kotlinx.coroutines.delay
import java.time.OffsetDateTime
import java.time.format.DateTimeFormatter
import java.util.Locale

/**
 * TimeIs終了確認画面。
 * タイトル・経過時間を表示し、✓で終了、✕でキャンセル。
 */
@Composable
fun PlayingEndConfirmScreen(
    title: String,
    startTime: String,
    onConfirm: () -> Unit,
    onCancel: () -> Unit
) {
    var tickSeconds by remember { mutableLongStateOf(System.currentTimeMillis() / 1000) }
    LaunchedEffect(Unit) {
        while (true) {
            delay(1000)
            tickSeconds = System.currentTimeMillis() / 1000
        }
    }

    val elapsed = formatElapsedConfirm(startTime, tickSeconds)

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = stringResource(R.string.confirm_end_timeis, title),
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 4.dp)
        )
        if (elapsed.isNotEmpty()) {
            Text(
                text = stringResource(R.string.elapsed, elapsed),
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 12.dp)
            )
        }
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceEvenly
        ) {
            Button(onClick = onCancel) {
                Text("✕")
            }
            Button(onClick = onConfirm) {
                Text("✓")
            }
        }
    }
}

private fun formatElapsedConfirm(isoTime: String, tickSeconds: Long): String {
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
