package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens

import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.wear.compose.foundation.lazy.ScalingLazyColumn
import androidx.wear.compose.foundation.lazy.items
import androidx.wear.compose.material.Chip
import androidx.wear.compose.material.ChipDefaults
import androidx.wear.compose.material.Text
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode

/**
 * Displays a scrollable list of KFTL templates.
 * - Folders (is_dir=true) navigate deeper
 * - Leaves (is_dir=false) show a confirmation screen
 */
@Composable
fun TemplateListScreen(
    nodes: List<TemplateNode>,
    title: String = "テンプレート",
    onNodeSelected: (TemplateNode) -> Unit
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
            Chip(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 8.dp, vertical = 2.dp),
                label = { Text("$prefix$label") },
                onClick = { onNodeSelected(node) },
                colors = ChipDefaults.primaryChipColors()
            )
        }
    }
}
