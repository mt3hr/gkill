package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertThrows
import org.junit.Test
import java.time.LocalDateTime

/**
 * 気分記録（Lantana）のヘルパーのテスト。
 *
 * ここが守るのは「Web 版と同じ刻みで値が決まること」と「サーバの KFTL パーサが
 * 実際に読める文字列を作ること」の2点。どちらも破れてもアプリはクラッシュせず、
 * 記録が黙って別の値になる／黙って Kmemo になるだけなので、文字列は完全一致で固定する。
 *
 * 対のサーバ側検査: src/server/gkill/api/kftl/kftl_statement_test.go の
 * TestStatement_LantanaFromWearOS
 */
class LantanaKftlTest {

    // ─── starFill: 値 → 星の塗り ────────────────────────────────────────────────

    @Test
    fun `mood 0 leaves every star empty`() {
        for (starIndex in 1..LANTANA_STAR_COUNT) {
            assertEquals("star $starIndex", StarFill.NONE, starFill(starIndex, 0))
        }
    }

    @Test
    fun `mood 10 fills every star`() {
        for (starIndex in 1..LANTANA_STAR_COUNT) {
            assertEquals("star $starIndex", StarFill.FULL, starFill(starIndex, LANTANA_MOOD_MAX))
        }
    }

    @Test
    fun `odd mood makes the last lit star a half`() {
        // mood 7 -> 星1,2,3 が満、星4 が半分、星5 は空（Web 版 use-lantana-flowers-view.ts と同じ）
        assertEquals(StarFill.FULL, starFill(1, 7))
        assertEquals(StarFill.FULL, starFill(2, 7))
        assertEquals(StarFill.FULL, starFill(3, 7))
        assertEquals(StarFill.HALF, starFill(4, 7))
        assertEquals(StarFill.NONE, starFill(5, 7))
    }

    @Test
    fun `even mood makes the last lit star a full`() {
        // mood 8 -> 星1〜4 が満、星5 は空
        assertEquals(StarFill.FULL, starFill(4, 8))
        assertEquals(StarFill.NONE, starFill(5, 8))
    }

    @Test
    fun `starFill matches the web formula for every mood and star`() {
        // Web 版: flower_state_N = mood >= 2N ? full : (mood >= 2N-1 ? half : none)
        for (mood in 0..LANTANA_MOOD_MAX) {
            for (starIndex in 1..LANTANA_STAR_COUNT) {
                val expected = when {
                    mood >= starIndex * 2 -> StarFill.FULL
                    mood >= starIndex * 2 - 1 -> StarFill.HALF
                    else -> StarFill.NONE
                }
                assertEquals("mood=$mood star=$starIndex", expected, starFill(starIndex, mood))
            }
        }
    }

    // ─── moodForHalf: タップ位置 → 値 ───────────────────────────────────────────

    @Test
    fun `left half of each star is the odd value`() {
        assertEquals(1, moodForHalf(1, rightHalf = false))
        assertEquals(3, moodForHalf(2, rightHalf = false))
        assertEquals(5, moodForHalf(3, rightHalf = false))
        assertEquals(7, moodForHalf(4, rightHalf = false))
        assertEquals(9, moodForHalf(5, rightHalf = false))
    }

    @Test
    fun `right half of each star is the even value`() {
        assertEquals(2, moodForHalf(1, rightHalf = true))
        assertEquals(4, moodForHalf(2, rightHalf = true))
        assertEquals(6, moodForHalf(3, rightHalf = true))
        assertEquals(8, moodForHalf(4, rightHalf = true))
        assertEquals(10, moodForHalf(5, rightHalf = true))
    }

    @Test
    fun `the ten halves cover 1 to 10 with no gap and no overlap`() {
        val reachable = (1..LANTANA_STAR_COUNT)
            .flatMap { listOf(moodForHalf(it, false), moodForHalf(it, true)) }
            .sorted()
        assertEquals((1..LANTANA_MOOD_MAX).toList(), reachable)
    }

    @Test
    fun `starFill and moodForHalf agree - tapping a half lights that half`() {
        for (starIndex in 1..LANTANA_STAR_COUNT) {
            assertEquals(
                "left half of star $starIndex",
                StarFill.HALF,
                starFill(starIndex, moodForHalf(starIndex, rightHalf = false))
            )
            assertEquals(
                "right half of star $starIndex",
                StarFill.FULL,
                starFill(starIndex, moodForHalf(starIndex, rightHalf = true))
            )
        }
    }

    // ─── buildLantanaKftlText: KFTL テキストの完全一致 ──────────────────────────

    @Test
    fun `builds the exact KFTL text the server parser accepts`() {
        val text = buildLantanaKftlText(7, LocalDateTime.of(2026, 9, 10, 14, 32, 5))
        assertEquals("?2026-09-10 14:32:05\n/mood\n7", text)
    }

    @Test
    fun `related time is zero padded to seconds`() {
        // 1桁の月日時分秒がゼロ埋めされないと dateFormats のどれにも一致せず、
        // 送信全体が「関連時刻がパースできません」で落ちる
        val text = buildLantanaKftlText(1, LocalDateTime.of(2026, 1, 2, 3, 4, 5))
        assertEquals("?2026-01-02 03:04:05\n/mood\n1", text)
    }

    @Test
    fun `uses ascii prefixes so a mistyped full width character cannot silently become a memo`() {
        val lines = buildLantanaKftlText(10, LocalDateTime.of(2026, 9, 10, 0, 0, 0)).split("\n")
        assertEquals(3, lines.size)
        assertEquals("?", lines[0].take(1))
        assertEquals("/mood", lines[1])
        assertEquals("10", lines[2])
    }

    @Test
    fun `mood 0 is rejected instead of writing the lowest mood silently`() {
        // KFTL パーサ側は 0 を受理するので、ここで止めないと「未選択のまま送信」が
        // 気分値0(最低)の記録になる
        assertThrows(IllegalArgumentException::class.java) {
            buildLantanaKftlText(0, LocalDateTime.of(2026, 9, 10, 14, 32, 5))
        }
    }

    @Test
    fun `mood above the maximum is rejected`() {
        assertThrows(IllegalArgumentException::class.java) {
            buildLantanaKftlText(LANTANA_MOOD_MAX + 1, LocalDateTime.of(2026, 9, 10, 14, 32, 5))
        }
    }

    @Test
    fun `negative mood is rejected`() {
        assertThrows(IllegalArgumentException::class.java) {
            buildLantanaKftlText(-1, LocalDateTime.of(2026, 9, 10, 14, 32, 5))
        }
    }

    @Test
    fun `two records a second apart differ so the submit ledger does not fold them`() {
        // WearSubmitLedger はテキスト完全一致・TTL24時間で重複判定する。
        // 関連時刻が入っていないと「今日2回目の同じ気分値」が毎回 DUPLICATE になる
        val first = buildLantanaKftlText(7, LocalDateTime.of(2026, 9, 10, 14, 32, 5))
        val second = buildLantanaKftlText(7, LocalDateTime.of(2026, 9, 10, 14, 32, 6))
        org.junit.Assert.assertNotEquals(first, second)
    }
}
