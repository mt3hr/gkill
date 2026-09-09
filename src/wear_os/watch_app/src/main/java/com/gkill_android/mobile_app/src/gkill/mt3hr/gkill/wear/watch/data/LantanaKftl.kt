package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data

import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

/**
 * Mood (Lantana) recording helpers, kept free of Android APIs so they stay JVM-testable.
 *
 * 気分値は Web 版と同じ 0-10 の整数で、星5個を左半分/右半分に割って 1-10 を選ぶ。
 * 0 は「未入力」の意味なので、ウォッチからは送らない。
 */

/** 星1つの塗り具合。Web 版の LantanaFlowerState と対。 */
enum class StarFill { NONE, HALF, FULL }

/** 星の数。Web 版の花と同じ5個。 */
const val LANTANA_STAR_COUNT = 5

/** 気分値の上限。KFTL パーサ側の検査 (kftl_lantana.go) と同じ。 */
const val LANTANA_MOOD_MAX = 10

/**
 * 星 [starIndex] 個目(1-[LANTANA_STAR_COUNT])の塗り具合を返す。
 *
 * Mirrors: src/client/classes/use-lantana-flowers-view.ts
 *   flower_state_N = mood >= 2N ? full : (mood >= 2N-1 ? half : none)
 */
fun starFill(starIndex: Int, mood: Int): StarFill = when {
    mood >= starIndex * 2 -> StarFill.FULL
    mood >= starIndex * 2 - 1 -> StarFill.HALF
    else -> StarFill.NONE
}

/**
 * 星 [starIndex] 個目(1-[LANTANA_STAR_COUNT])の左半分/右半分をタップしたときの気分値。
 *
 * Mirrors: src/client/pages/views/lantana-flowers-view.vue
 *   clicked_left -> set_mood(2N-1) / clicked_right -> set_mood(2N)
 */
fun moodForHalf(starIndex: Int, rightHalf: Boolean): Int =
    starIndex * 2 - if (rightHalf) 0 else 1

// 関連時刻の書式。KFTL パーサ (kftl_related_time_statement_line.go の dateFormats) が
// 受け付ける形のうち秒まで持つもの。**オフセット付き ISO8601 は dateFormats に無い**ので
// 使ってはいけない（パースに失敗して送信全体が行エラーになる）。
private val RELATED_TIME_FORMAT: DateTimeFormatter =
    DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm:ss")

/**
 * 気分値を KFTL テキストへ組み立てる。組み立てた文字列はそのまま
 * [GkillWearClient.sendSubmitRequest] -> `/gkill/submit` -> `POST /api/submit_kftl_text` へ流れる。
 *
 * 出力例: `?2026-09-10 14:32:05\n/mood\n7`
 *
 * - プレフィックスは **ASCII を使う**。全角の `ーら` は先頭が長音符(U+30FC)で、漢数字の一や
 *   全角ハイフンと取り違えるとプレフィックス判定(完全一致)が外れ、エラーも警告も出ないまま
 *   Kmemo が2件書かれる。
 * - 関連時刻の行を必ず付ける。付けないと同じ気分値の記録が WearSubmitLedger
 *   (テキスト完全一致・TTL24時間) に重複と判定されて毎回確認画面が出る。
 *   スマホが圏外で送信が遅れたとき、タップ時刻ではなくサーバ受信時刻が残る問題も防ぐ。
 * - 保存文字 `！` は不要。KFTL では「以降の行を打ち切る」マーカーであって保存トリガではない
 *   (kftl_statement.go の save character の分岐)。
 *
 * 対の検査: `kftl_statement_test.go` の `TestStatement_LantanaFromWearOS`
 */
fun buildLantanaKftlText(mood: Int, at: LocalDateTime): String {
    // 値そのものは例外メッセージにも載せない（記録内容が logcat へ出るのを避ける。2026-08-30 監査 F-008）
    require(mood in 1..LANTANA_MOOD_MAX) { "lantana mood must be 1..$LANTANA_MOOD_MAX" }
    return "?" + at.format(RELATED_TIME_FORMAT) + "\n/mood\n" + mood
}
