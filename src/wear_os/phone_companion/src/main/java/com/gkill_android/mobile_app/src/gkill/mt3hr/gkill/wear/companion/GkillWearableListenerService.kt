package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.util.Log
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.OutOfQuotaPolicy
import androidx.work.WorkManager
import androidx.work.workDataOf
import com.google.android.gms.wearable.MessageEvent
import com.google.android.gms.wearable.WearableListenerService
import java.util.UUID

private const val TAG = "GkillWearSvc"

/**
 * スマホ側で動き、時計からのメッセージを受ける。
 *
 * 受信したら [WearRequestWorker] へ expedited で enqueue するだけにする。
 * 実際の 認証情報ロード → API 呼び出し → 結果送信 はワーカーが担うので、
 * この Service が破棄されてもプロセスが死んでも処理は完了する。
 */
class GkillWearableListenerService : WearableListenerService() {

    override fun onMessageReceived(event: MessageEvent) {
        val path = event.path
        Log.d(TAG, "onMessageReceived path=$path sourceNode=${event.sourceNodeId}")
        if (path !in KNOWN_REQUEST_PATHS) {
            Log.w(TAG, "Unknown path: $path")
            return
        }

        val request = OneTimeWorkRequestBuilder<WearRequestWorker>()
            // 時計は最大20〜30秒しか待たないので優先実行する。クォータ枯渇時は通常実行へ落とす。
            .setExpedited(OutOfQuotaPolicy.RUN_AS_NON_EXPEDITED_WORK_REQUEST)
            .setInputData(
                workDataOf(
                    WearRequestWorker.KEY_PATH to path,
                    WearRequestWorker.KEY_SOURCE_NODE_ID to event.sourceNodeId,
                    WearRequestWorker.KEY_DATA to event.data,
                    // メッセージ1件ごとに採番。WorkRequest の入力は不変なので、この同じ要求の
                    // ワーカー再送では同じキーになり、サーバーが二重登録を畳む。意図的な再送は
                    // 別メッセージ＝別キーなので畳まれない（監査 S3-wear）。
                    WearRequestWorker.KEY_IDEMPOTENCY to UUID.randomUUID().toString(),
                )
            )
            .build()
        WorkManager.getInstance(applicationContext).enqueue(request)
    }
}
