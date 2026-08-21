package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import kotlinx.serialization.Serializable
import kotlinx.serialization.builtins.ListSerializer
import kotlinx.serialization.json.Json

/**
 * KFTL 送信の重複対策台帳。直近に「保存が成功した」テキストを覚えておき、完全一致で重複を判定する。
 *
 * Web Share Target の share-target-dedup 契約を電話側へ写したもの:
 * - 台帳へ載せるのは保存が成功したときだけ（[recordSuccess] は成功後にだけ呼ぶ）。
 * - 再配送と意図的な再送は内容から区別できないので、内容の完全一致でしか判定できない。
 * - 直近100件・24時間のみ保持する。
 *
 * プロセス死やサービス破棄を跨いで効かせるため、[storage] 経由で永続化する（本番は SharedPreferences）。
 * ロジックはここに集約し、[storage] と [clock] を差し替えて JVM 単体テストできるようにしてある。
 */
class WearSubmitLedger(
    private val storage: Storage,
    private val maxEntries: Int = 100,
    private val ttlMillis: Long = 24L * 60 * 60 * 1000,
    private val clock: () -> Long = System::currentTimeMillis,
) {

    /** 台帳の保存先。本番は SharedPreferences、テストはインメモリ。 */
    interface Storage {
        fun read(): String
        fun write(value: String)
    }

    @Serializable
    private data class Entry(val text: String, val at: Long)

    @Synchronized
    fun isDuplicate(kftlText: String): Boolean {
        val threshold = clock() - ttlMillis
        return load().any { it.at >= threshold && it.text == kftlText }
    }

    @Synchronized
    fun recordSuccess(kftlText: String) {
        val now = clock()
        val threshold = now - ttlMillis
        // 期限切れと同一テキストの旧エントリを落としてから末尾へ追加し、上限で古い順に切る
        val kept = load().filter { it.at >= threshold && it.text != kftlText }
        val updated = (kept + Entry(kftlText, now)).takeLast(maxEntries)
        storage.write(Json.encodeToString(ListSerializer(Entry.serializer()), updated))
    }

    private fun load(): List<Entry> = try {
        val raw = storage.read()
        if (raw.isEmpty()) {
            emptyList()
        } else {
            Json.decodeFromString(ListSerializer(Entry.serializer()), raw)
        }
    } catch (_: Exception) {
        emptyList()
    }
}
