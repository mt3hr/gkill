package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model

import kotlinx.serialization.Serializable

/**
 * Mirrors: src/classes/datas/config/kftl-template-struct-element-data.ts
 *
 * Represents one node in the KFTL template tree.
 * - is_dir=true  → folder node, has children
 * - is_dir=false → leaf node, has a template string to send
 */
@Serializable
data class TemplateNode(
    val name: String = "",
    val id: String? = null,
    val title: String = "",
    val template: String = "",
    val children: List<TemplateNode>? = null,
    val key: String = "",
    val is_dir: Boolean = false,
    val is_open_default: Boolean = false
)
