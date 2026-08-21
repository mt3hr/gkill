package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.content.Context
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.work.CoroutineWorker
import androidx.work.ForegroundInfo
import androidx.work.WorkerParameters
import com.google.android.gms.wearable.Wearable
import kotlinx.coroutines.tasks.await

private const val TAG = "GkillWearWorker"

/**
 * 時計から届いた1件の要求を実際に処理するワーカー。
 *
 * [GkillWearableListenerService] は受信をここへ enqueue するだけにして、
 * 認証情報のロード → API 呼び出し → 結果送信 をワーカーが担う。こうすることで
 * Service が破棄されてもプロセスが死んでも WorkManager が処理を完了できる。
 *
 * expedited（優先）で実行し、処理内容は Android 非依存の [WearRequestHandler] に委譲する。
 */
class WearRequestWorker(
    appContext: Context,
    params: WorkerParameters,
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        val path = inputData.getString(KEY_PATH) ?: return Result.failure()
        val nodeId = inputData.getString(KEY_SOURCE_NODE_ID) ?: return Result.failure()
        val data = inputData.getByteArray(KEY_DATA) ?: ByteArray(0)
        // WorkRequest の入力は不変なので、同じ要求のワーカー再送では同じ値になる。
        // サーバーはこのキーで KFTL 送信の再送を1回の登録に畳む（監査 S3-wear）。
        val idempotencyKey = inputData.getString(KEY_IDEMPOTENCY)
        Log.d(TAG, "doWork path=$path nodeId=$nodeId")

        val store = GkillCredentialStore(applicationContext)
        val hostKey = GkillServerTrust.hostKeyOf(store.getServerUrl())
        val pin = store.getPinnedCertSha256(hostKey).ifEmpty { null }
        val apiClient = GkillApiClient(store.getServerUrl(), store.getAllowSelfSignedCert(), pin)

        val ledger = WearSubmitLedger(SharedPrefsLedgerStorage(applicationContext))
        val handler = WearRequestHandler(
            apiClient = apiClient,
            sessionProvider = { getOrRefreshSessionId(store, apiClient) },
            submitLedger = ledger,
            idempotencyKey = idempotencyKey,
        )

        val response = handler.handle(path, data)
        if (response == null) {
            Log.w(TAG, "Unknown path: $path")
            return Result.success()
        }
        sendMessage(nodeId, response.path, response.data)
        return Result.success()
    }

    /**
     * 有効なセッションIDを返す。キャッシュを先に試し、駄目なら再ログインする。
     */
    private fun getOrRefreshSessionId(
        store: GkillCredentialStore,
        apiClient: GkillApiClient,
    ): String? {
        val cached = store.getSessionId()
        if (cached.isNotEmpty()) {
            val test = apiClient.getKftlTemplateStructJson(cached)
            if (test != null) return cached
        }
        val newSession = apiClient.login(store.getUserId(), store.getPasswordSha256())
        if (newSession != null) {
            store.setSessionId(newSession)
        }
        return newSession
    }

    private suspend fun sendMessage(nodeId: String, path: String, data: ByteArray) {
        try {
            Wearable.getMessageClient(applicationContext).sendMessage(nodeId, path, data).await()
        } catch (e: Exception) {
            Log.e(TAG, "Failed to send message path=$path", e)
        }
    }

    /**
     * expedited work は API 31 未満では前景サービスとして走るため、通知が必須。
     * 目立たない低重要度の通知を出すだけにする。
     */
    override suspend fun getForegroundInfo(): ForegroundInfo {
        val nm = applicationContext.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                NOTIF_CHANNEL_ID,
                "gkill Wear 同期",
                NotificationManager.IMPORTANCE_LOW,
            )
            nm.createNotificationChannel(channel)
        }
        val notification: Notification = NotificationCompat.Builder(applicationContext, NOTIF_CHANNEL_ID)
            .setContentTitle("gkill Wear")
            .setContentText("時計からの要求を処理中")
            .setSmallIcon(android.R.drawable.stat_notify_sync)
            .setOngoing(true)
            .build()
        return ForegroundInfo(NOTIF_ID, notification)
    }

    /** 本番用の台帳保存先。認証情報と同じ SharedPreferences に専用キーで置く。 */
    private class SharedPrefsLedgerStorage(context: Context) : WearSubmitLedger.Storage {
        private val prefs = context.getSharedPreferences("gkill_wear_prefs", Context.MODE_PRIVATE)
        override fun read(): String = prefs.getString(KEY, "") ?: ""
        override fun write(value: String) {
            prefs.edit().putString(KEY, value).apply()
        }

        private companion object {
            const val KEY = "submit_dedup_ledger"
        }
    }

    companion object {
        const val KEY_PATH = "path"
        const val KEY_SOURCE_NODE_ID = "source_node_id"
        const val KEY_DATA = "data"
        const val KEY_IDEMPOTENCY = "idempotency_key"

        private const val NOTIF_CHANNEL_ID = "gkill_wear_sync"
        private const val NOTIF_ID = 42
    }
}
