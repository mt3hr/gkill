package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * テンプレート一覧の階層ごとのスクロール位置の記憶。
 *
 * 守りたい約束は3つだけ:
 * - 親→子（フォルダを開く）は先頭から
 * - 子→親（Back で戻る）は親で見ていた位置へ戻る
 * - それ以外（読み込み直し・確認画面からの復帰・ホームへ戻る）は先頭から
 */
class TemplateScrollMemoryTest {

    private val middle = TemplateScrollPosition(7, 12)
    private val deeper = TemplateScrollPosition(3, 5)

    @Test
    fun `initial position is the top`() {
        assertEquals(TemplateScrollPosition.TOP, TemplateScrollMemory().takeInitialPosition())
    }

    @Test
    fun `entering a folder shows the child from the top`() {
        val memory = TemplateScrollMemory()
        memory.enterFolder(middle)
        assertEquals(TemplateScrollPosition.TOP, memory.takeInitialPosition())
    }

    @Test
    fun `going back to the parent restores the parent position`() {
        val memory = TemplateScrollMemory()
        memory.enterFolder(middle)
        memory.takeInitialPosition()
        memory.backToParent()
        assertEquals(middle, memory.takeInitialPosition())
    }

    @Test
    fun `the restored position is used once only`() {
        val memory = TemplateScrollMemory()
        memory.enterFolder(middle)
        memory.backToParent()
        memory.takeInitialPosition()
        // 確認画面から戻ったときなど、同じ階層を作り直すだけなら先頭から
        assertEquals(TemplateScrollPosition.TOP, memory.takeInitialPosition())
    }

    @Test
    fun `nested folders come back in order`() {
        val memory = TemplateScrollMemory()
        memory.enterFolder(middle)
        memory.enterFolder(deeper)
        memory.backToParent()
        assertEquals(deeper, memory.takeInitialPosition())
        memory.backToParent()
        assertEquals(middle, memory.takeInitialPosition())
    }

    @Test
    fun `reset forgets every stacked position`() {
        val memory = TemplateScrollMemory()
        memory.enterFolder(middle)
        memory.reset()
        memory.backToParent()
        assertEquals(TemplateScrollPosition.TOP, memory.takeInitialPosition())
    }

    @Test
    fun `going back with nothing stacked stays at the top`() {
        val memory = TemplateScrollMemory()
        memory.backToParent()
        assertEquals(TemplateScrollPosition.TOP, memory.takeInitialPosition())
    }

    @Test
    fun `reset also clears a pending restored position`() {
        val memory = TemplateScrollMemory()
        memory.enterFolder(middle)
        memory.backToParent()
        memory.reset()
        assertEquals(TemplateScrollPosition.TOP, memory.takeInitialPosition())
    }

    @Test
    fun `entering a folder clears an unconsumed pending position`() {
        val memory = TemplateScrollMemory()
        memory.enterFolder(middle)
        memory.backToParent()
        memory.enterFolder(deeper)
        assertEquals(TemplateScrollPosition.TOP, memory.takeInitialPosition())
    }
}
