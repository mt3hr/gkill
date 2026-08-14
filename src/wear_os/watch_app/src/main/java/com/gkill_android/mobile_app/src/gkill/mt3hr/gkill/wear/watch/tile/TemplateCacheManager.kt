package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile

import android.content.Context
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.GkillWearClient
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode

object TemplateCacheManager {
    private const val PREF_NAME = "gkill_tile_cache"
    private const val KEY_TEMPLATES_JSON = "templates_json"
    private const val KEY_LAST_UPDATED = "last_updated"

    fun saveRawJson(context: Context, json: String) {
        context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE).edit()
            .putString(KEY_TEMPLATES_JSON, json)
            .putLong(KEY_LAST_UPDATED, System.currentTimeMillis())
            .apply()
    }

    fun loadTemplates(context: Context): List<TemplateNode> {
        val json = context.getSharedPreferences(PREF_NAME, Context.MODE_PRIVATE)
            .getString(KEY_TEMPLATES_JSON, null) ?: return emptyList()
        return GkillWearClient.parseTemplates(json)
    }

    /**
     * スマホへテンプレートを取りに行くかどうかの判定。
     *
     * 一覧の「🔄 更新」が押されたとき（forceReload=true）は、キャッシュが載っていても
     * 必ず取りに行く。これを飛ばすとサーバ側でテンプレートを直しても永久に反映されない。
     */
    fun shouldFetchFromPhone(forceReload: Boolean, cachedSize: Int): Boolean =
        forceReload || cachedSize == 0
}
