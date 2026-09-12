package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.wear.compose.foundation.lazy.ScalingLazyColumn
import androidx.wear.compose.foundation.lazy.items
import androidx.wear.compose.material.CompactChip
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.R
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.components.MenuChip

/**
 * Displays a scrollable list of KFTL templates.
 * - Folders (is_dir=true) navigate deeper
 * - Leaves (is_dir=false) show a confirmation screen
 * - 末尾の「🔄 更新」（R.string.refresh）でスマホ経由のテンプレート再取得を要求する
 */
@Composable
fun TemplateListScreen(
    nodes: List<TemplateNode>,
    title: String = stringResource(R.string.template_list_title),
    onNodeSelected: (TemplateNode) -> Unit,
    onRefresh: () -> Unit
) {
    ScalingLazyColumn(modifier = Modifier.fillMaxSize()) {
        item {
            Text(
                text = title,
                textAlign = TextAlign.Center,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 4.dp)
            )
        }
        items(nodes) { node ->
            val label = if (node.title.isNotEmpty()) node.title else node.name
            val prefix = if (node.is_dir) "📁 " else ""
            // タイルと同じ見た目（140dp・中央揃え）。ScalingLazyColumn の既定で中央に並ぶ
            MenuChip(
                label = "$prefix$label",
                onClick = { onNodeSelected(node) }
            )
        }

        item {
            CompactChip(
                label = { Text(stringResource(R.string.refresh)) },
                onClick = onRefresh
            )
        }
    }
}
