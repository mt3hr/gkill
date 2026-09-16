package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile

import android.content.Context
import android.content.Intent
import androidx.wear.tiles.TileService
import androidx.wear.tiles.TileUpdateRequester
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkStatic
import io.mockk.unmockkStatic
import io.mockk.verify
import org.junit.After
import org.junit.Before
import org.junit.Test

/**
 * 端末の言語変更でタイルの再描画を要求するレシーバのテスト。
 *
 * タイルのレイアウトは onTileRequest の時点で文字列ごと固定されるので、ここが requestUpdate を
 * 投げないと言語を切り替えてもタイルだけ古い言語のまま残る（Activity は再生成で追従する）。
 * 逆に LOCALE_CHANGED 以外まで拾うと、manifest のフィルタを広げたときに無関係な
 * ブロードキャストで再描画が走る。TileService.getUpdater は静的メソッドなので MockK で差し替える。
 */
class LocaleChangedReceiverTest {

    private lateinit var context: Context
    private lateinit var updater: TileUpdateRequester

    @Before
    fun setUp() {
        context = mockk()
        updater = mockk(relaxed = true)
        mockkStatic(TileService::class)
        every { TileService.getUpdater(context) } returns updater
    }

    @After
    fun tearDown() {
        unmockkStatic(TileService::class)
    }

    private fun intentWithAction(action: String?): Intent {
        val intent = mockk<Intent>()
        every { intent.action } returns action
        return intent
    }

    @Test
    fun `locale change requests a tile update for GkillTileService`() {
        LocaleChangedReceiver().onReceive(context, intentWithAction(Intent.ACTION_LOCALE_CHANGED))

        verify(exactly = 1) { updater.requestUpdate(GkillTileService::class.java) }
    }

    @Test
    fun `other broadcasts are ignored without touching the tile updater`() {
        LocaleChangedReceiver().onReceive(context, intentWithAction(Intent.ACTION_TIME_CHANGED))
        LocaleChangedReceiver().onReceive(context, intentWithAction(null))

        verify(exactly = 0) { TileService.getUpdater(any()) }
        verify(exactly = 0) { updater.requestUpdate(any()) }
    }
}
