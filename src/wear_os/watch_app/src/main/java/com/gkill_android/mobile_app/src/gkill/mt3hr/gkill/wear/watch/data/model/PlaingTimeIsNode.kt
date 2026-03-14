package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model

import kotlinx.serialization.Serializable

/**
 * 再生中（進行中）のTimeIsを表すデータモデル。
 * Phone Companionから受信したJSON配列の各要素に対応する。
 */
@Serializable
data class PlaingTimeIsNode(
    val id: String = "",
    val rep_name: String = "",
    val title: String = "",
    val start_time: String = "",  // ISO 8601 / RFC 3339
    val data_type: String = "",
    val is_deleted: Boolean = false
)
