package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.components

import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.width
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.wear.compose.material.Chip
import androidx.wear.compose.material.ChipDefaults
import androidx.wear.compose.material.Text

/**
 * タイル（GkillTileService）とアプリのチップで共有する幅（dp）。
 * 見た目を揃えるための値なので、片方だけ変えないこと。
 */
const val MENU_CHIP_WIDTH_DP = 140f

/**
 * タイルのチップと同じ見た目の Compose チップ。トップメニューと一覧画面の項目に使う。
 *
 * タイルは protolayout-material の Chip をほぼ既定のまま使っている:
 * 140dp × 52dp・primary 色・文字は中央揃え・副ラベル無しなら最大2行/有りなら1行で末尾省略。
 * 高さ 52dp・水平パディング 14dp・ピル形状は Wear Compose の Chip 既定が同じなので指定しない。
 * ラベルの Text に fillMaxWidth が無いと Row 内で左寄せのままになる。
 */
@Composable
fun MenuChip(
    label: String,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
    secondaryLabel: String? = null
) {
    Chip(
        modifier = modifier.width(MENU_CHIP_WIDTH_DP.dp),
        label = {
            Text(
                text = label,
                textAlign = TextAlign.Center,
                maxLines = if (secondaryLabel == null) 2 else 1,
                overflow = TextOverflow.Ellipsis,
                modifier = Modifier.fillMaxWidth()
            )
        },
        secondaryLabel = secondaryLabel?.let {
            {
                Text(
                    text = it,
                    textAlign = TextAlign.Center,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                    modifier = Modifier.fillMaxWidth()
                )
            }
        },
        onClick = onClick,
        colors = ChipDefaults.primaryChipColors()
    )
}
