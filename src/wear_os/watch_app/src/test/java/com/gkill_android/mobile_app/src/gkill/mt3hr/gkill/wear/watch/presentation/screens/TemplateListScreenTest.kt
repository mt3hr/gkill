package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.presentation.screens

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/**
 * テンプレート一覧のスクロール位置の配線を、ソースの字面で守る検査。
 *
 * watch_app には Compose UI テストの依存（ui-test-junit4 / Robolectric）が無く、`@Composable` の
 * 実挙動を JVM テストで動かせない。方針そのもの（どの移動で位置を引き継ぐか）は
 * `TemplateScrollMemoryTest` が本物のテストで守るので、ここは **それが画面につながっていること**
 * だけを見る。配線が外れると、エラーも警告も出ないまま掘っても戻っても前の階層の位置が残る。
 *
 * 字面で見ているので、書式を変えるときはこの期待値も一緒に直すこと。
 */
class TemplateListScreenTest {

    private companion object {
        const val BASE = "java/com/gkill_android/mobile_app/src/gkill/mt3hr/gkill/wear/watch"
        const val SCREEN = "$BASE/presentation/screens/TemplateListScreen.kt"
        const val ACTIVITY = "$BASE/MainActivity.kt"
    }

    /** Gradle の単体テストは projectDir で走るので相対パスで届く。念のため親からも探す。 */
    private fun source(rel: String): String {
        val candidates = listOf("src/main/$rel", "watch_app/src/main/$rel")
        val file = candidates.map { File(it) }.firstOrNull { it.isFile }
            ?: error("$rel not found from user.dir=" + System.getProperty("user.dir"))
        return file.readText()
    }

    /**
     * コメントを落とした本体の行。KDoc が「書いてはいけない字面」にも触れているので、
     * 検査はコードの行だけを見る。
     */
    private fun codeLines(rel: String): List<String> =
        source(rel).lineSequence()
            .map { it.trimStart() }
            .filterNot { it.startsWith("//") || it.startsWith("*") || it.startsWith("/*") }
            .toList()

    private fun List<String>.has(text: String): Boolean = any { it.contains(text) }

    @Test
    fun `template list screen takes the list state from its caller`() {
        val screen = codeLines(SCREEN)
        assertTrue(
            "TemplateListScreen は state を引数で受け取ること",
            screen.has("state: ScalingLazyListState,")
        )
        assertTrue(
            "受け取った state を ScalingLazyColumn へ渡すこと",
            screen.has("ScalingLazyColumn(modifier = Modifier.fillMaxSize(), state = state)")
        )
        assertFalse(
            "画面の中で state を作らないこと（既定の state は1つの呼び出し位置に remember され、" +
                "掘っても戻っても前の階層のスクロール位置を引き継ぐ）",
            screen.has("rememberScalingLazyListState") || screen.has("rememberSaveable")
        )
    }

    @Test
    fun `main activity wires the scroll memory to the template list`() {
        val activity = codeLines(ACTIVITY)
        assertTrue(
            "階層が変わるたびに state を作り直すこと",
            activity.has("val listState = remember(s.breadcrumb.size, s.title, s.nodes)")
        )
        assertTrue(
            "初期位置は TemplateScrollMemory から取ること",
            activity.has("templateScroll.takeInitialPosition()")
        )
        assertTrue(
            "作った state を画面へ渡すこと",
            activity.has("state = listState,")
        )
        assertTrue(
            "フォルダを掘るときに親の位置を積むこと",
            activity.has("templateScroll.enterFolder(")
        )
        assertTrue(
            "Back で親へ戻るときに積んだ位置を取り出すこと",
            activity.has("templateScroll.backToParent()")
        )
        assertEquals(
            "一覧の読み込み直し（requestTemplates）とホームへ戻るとき（navigateBackToTopOrFinish）の" +
                "2箇所で、積んだ位置を捨てること",
            2,
            activity.count { it.contains("templateScroll.reset()") }
        )
    }
}
