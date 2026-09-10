package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.graphics.drawscope.clipRect
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.TextUnit
import androidx.compose.ui.unit.dp
import androidx.wear.compose.material.Button
import androidx.wear.compose.material.MaterialTheme
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.R
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.LANTANA_STAR_COUNT
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.StarFill
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.moodForHalf
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.starFill

// 星の字形。★ 1文字を「薄い下地」と「クリップした濃い上層」の2層に重ねて半分を表現する。
// ☆ と ★ を上下に重ねる方式は字形の送り幅がフォント次第でずれるので採らない
// （Web 版 lantana-flower.vue が下層をグレースケールにしているのと同じ考え方）。
private const val STAR_GLYPH = "★"

/** 選択画面の星の大きさ。5個で 160dp。丸画面の中央行に収まる幅。 */
private val SELECT_STAR_SIZE = 32.dp

/** 確認画面の星の大きさ（読み取り専用なので小さくてよい）。5個で 120dp。 */
private val CONFIRM_STAR_SIZE = 24.dp

/** 左右半分の当たり判定は星の幅の半分しかないので、縦方向を広げて押しやすくする。 */
private val TAP_AREA_EXTRA_HEIGHT = 24.dp

/**
 * 気分値を星5個で表す行。
 *
 * [onHalfTapped] が null なら読み取り専用（確認画面用）。非 null なら星ごとに
 * 左半分/右半分の当たり判定を重ね、`(星の番号, 右半分か)` を返す。
 */
@Composable
fun LantanaStars(
    mood: Int,
    starSize: Dp,
    onHalfTapped: ((starIndex: Int, rightHalf: Boolean) -> Unit)? = null
) {
    val fontSize = with(LocalDensity.current) { starSize.toSp() }
    val boxHeight = if (onHalfTapped != null) starSize + TAP_AREA_EXTRA_HEIGHT else starSize
    Row(
        horizontalArrangement = Arrangement.Center,
        verticalAlignment = Alignment.CenterVertically
    ) {
        for (starIndex in 1..LANTANA_STAR_COUNT) {
            Box(
                contentAlignment = Alignment.Center,
                modifier = Modifier.size(width = starSize, height = boxHeight)
            ) {
                LantanaStar(fill = starFill(starIndex, mood), fontSize = fontSize)
                if (onHalfTapped != null) {
                    Row(modifier = Modifier.matchParentSize()) {
                        Box(
                            modifier = Modifier
                                .weight(1f)
                                .fillMaxHeight()
                                .clickable { onHalfTapped(starIndex, false) }
                        )
                        Box(
                            modifier = Modifier
                                .weight(1f)
                                .fillMaxHeight()
                                .clickable { onHalfTapped(starIndex, true) }
                        )
                    }
                }
            }
        }
    }
}

/** 星1つ。none/half/full を 0f/0.5f/1f の横クリップで描き分ける。 */
@Composable
private fun LantanaStar(fill: StarFill, fontSize: TextUnit) {
    val fraction = when (fill) {
        StarFill.NONE -> 0f
        StarFill.HALF -> 0.5f
        StarFill.FULL -> 1f
    }
    Box(contentAlignment = Alignment.Center) {
        Text(
            text = STAR_GLYPH,
            fontSize = fontSize,
            color = MaterialTheme.colors.onSurface.copy(alpha = 0.25f)
        )
        if (fraction > 0f) {
            Text(
                text = STAR_GLYPH,
                fontSize = fontSize,
                color = MaterialTheme.colors.primary,
                modifier = Modifier.drawWithContent {
                    // 受け側を名前で束ねる。this@drawWithContent と書くと
                    // verify_docs の個人情報検査がメールアドレスとして拾う
                    val contentScope = this
                    clipRect(right = size.width * fraction) {
                        contentScope.drawContent()
                    }
                }
            )
        }
    }
}

/**
 * 気分を選ぶ画面。星の左半分/右半分をタップすると 1-10 の気分値が決まる
 * （Web 版の気分記録画面と同じ刻み）。
 */
@Composable
fun LantanaSelectScreen(
    mood: Int,
    onMoodSelected: (Int) -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(8.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = stringResource(R.string.lantana_title),
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 4.dp)
        )
        LantanaStars(
            mood = mood,
            starSize = SELECT_STAR_SIZE,
            onHalfTapped = { starIndex, rightHalf ->
                onMoodSelected(moodForHalf(starIndex, rightHalf))
            }
        )
    }
}

/**
 * 送信前の確認画面。選んだ気分は**数値ではなく星の絵で**見せる。
 */
@Composable
fun LantanaConfirmScreen(
    mood: Int,
    onConfirm: () -> Unit,
    onCancel: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = stringResource(R.string.lantana_confirm),
            textAlign = TextAlign.Center,
            modifier = Modifier
                .fillMaxWidth()
                .padding(bottom = 8.dp)
        )
        LantanaStars(mood = mood, starSize = CONFIRM_STAR_SIZE)
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 12.dp),
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
