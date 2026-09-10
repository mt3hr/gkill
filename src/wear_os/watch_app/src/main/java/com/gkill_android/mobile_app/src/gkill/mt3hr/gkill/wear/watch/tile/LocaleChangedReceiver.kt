package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import androidx.wear.tiles.TileService

/**
 * 端末の言語が変わったらタイルの再描画を要求する。
 *
 * タイルのレイアウトは [GkillTileService.onTileRequest] の時点で文字列ごと固定され、
 * ホストが次に要求し直すまで古い言語のまま残る（Activity は再生成で追従するが、タイルは追従しない）。
 * `ACTION_LOCALE_CHANGED` は暗黙ブロードキャスト制限の除外対象なので manifest 登録で受けられる。
 */
class LocaleChangedReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_LOCALE_CHANGED) return
        TileService.getUpdater(context).requestUpdate(GkillTileService::class.java)
    }
}
