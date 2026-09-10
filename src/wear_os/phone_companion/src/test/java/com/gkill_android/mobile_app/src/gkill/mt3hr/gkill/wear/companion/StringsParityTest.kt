package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import org.junit.Assert.*
import org.junit.Test
import org.w3c.dom.Element
import java.io.File
import javax.xml.parsers.DocumentBuilderFactory

/**
 * `src/main/res/values` と `values-xx` の `strings.xml` の7言語セットが揃っていることの検査。
 * Web の `i18n-completeness.test.ts` と同じ役目で、Android のリソース解決は
 * キーが欠けても例外を出さず既定言語（日本語）へ静かに落ちるので、ここでしか守れない。
 *
 * watch_app にも同型のテストがある（モジュールをまたぐ共有はしない。verify_docs の件数集計が
 * モジュール単位のため）。直すときは両方へ。
 */
class StringsParityTest {

    private data class Entry(val value: String, val translatable: Boolean)

    private companion object {
        /** 既定 values/（ja）以外に置く翻訳。gkill 本体の src/locales と同じ7言語。 */
        val EXPECTED_LOCALES = setOf("en", "zh", "ko", "es", "fr", "de")

        /** 言語修飾子のディレクトリ。values-night 等の非言語修飾子は対象外。 */
        val LANG_DIR = Regex("""^values-([a-z]{2})(-r[A-Z]{2})?$""")

        /** `%1$s` / `%s` / `%d` 等。`%%` は別扱い。 */
        val PLACEHOLDER = Regex("""%(\d+\$)?[sd]""")
    }

    /** Gradle の単体テストは projectDir で走るので相対パスで届く。念のため親からも探す。 */
    private val resDir: File by lazy {
        val candidates = listOf("src/main/res", "phone_companion/src/main/res")
        candidates.map { File(it) }.firstOrNull { File(it, "values/strings.xml").isFile }
            ?: fail("strings.xml not found from user.dir=${System.getProperty("user.dir")}") as Nothing
    }

    private val defaultStrings: Map<String, Entry> by lazy { load(File(resDir, "values/strings.xml")) }

    private val localeStrings: Map<String, Map<String, Entry>> by lazy {
        resDir.listFiles()!!
            .filter { it.isDirectory && LANG_DIR.matches(it.name) }
            .associate { dir -> dir.name.removePrefix("values-") to load(File(dir, "strings.xml")) }
    }

    private fun load(file: File): Map<String, Entry> {
        assertTrue("missing ${file.path}", file.isFile)
        val doc = DocumentBuilderFactory.newInstance().newDocumentBuilder().parse(file)
        val nodes = doc.getElementsByTagName("string")
        val out = LinkedHashMap<String, Entry>()
        for (i in 0 until nodes.length) {
            val el = nodes.item(i) as Element
            val name = el.getAttribute("name")
            assertFalse("duplicate key '$name' in ${file.path}", out.containsKey(name))
            out[name] = Entry(el.textContent, el.getAttribute("translatable") != "false")
        }
        return out
    }

    private fun translatableKeys(m: Map<String, Entry>): Set<String> =
        m.filterValues { it.translatable }.keys

    // -----------------------------------------------------------------------

    @Test
    fun localeDirectories_areExactlyTheSixTranslations() {
        assertEquals(EXPECTED_LOCALES, localeStrings.keys)
    }

    @Test
    fun everyLocale_hasExactlyTheTranslatableKeysOfDefault() {
        val expected = translatableKeys(defaultStrings)
        assertTrue("default strings.xml has no translatable keys", expected.isNotEmpty())
        for ((locale, strings) in localeStrings) {
            val actual = strings.keys
            val missing = expected - actual
            val extra = actual - expected
            assertTrue(
                "values-$locale: missing=$missing extra=$extra",
                missing.isEmpty() && extra.isEmpty()
            )
        }
    }

    @Test
    fun nonTranslatableKeys_areNotRepeatedInLocales() {
        val fixed = defaultStrings.filterValues { !it.translatable }.keys
        assertTrue("expected app_name to be translatable=false", "app_name" in fixed)
        for ((locale, strings) in localeStrings) {
            val leaked = fixed.intersect(strings.keys)
            assertTrue("values-$locale repeats non-translatable keys $leaked", leaked.isEmpty())
        }
    }

    @Test
    fun noValueIsBlank() {
        val all = mapOf("values" to defaultStrings) + localeStrings.mapKeys { "values-${it.key}" }
        for ((dir, strings) in all) {
            for ((key, entry) in strings) {
                assertTrue("$dir/$key is blank", entry.value.isNotBlank())
            }
        }
    }

    @Test
    fun placeholders_matchDefaultInEveryLocale() {
        for ((locale, strings) in localeStrings) {
            for ((key, entry) in strings) {
                val expected = PLACEHOLDER.findAll(defaultStrings.getValue(key).value).map { it.value }.sorted().toList()
                val actual = PLACEHOLDER.findAll(entry.value).map { it.value }.sorted().toList()
                assertEquals("values-$locale/$key placeholders differ from default", expected, actual)
                // 位置指定なしの %s が2つ以上あると aapt2 が拒否する
                val unpositioned = actual.count { !it.contains("$") }
                assertTrue("values-$locale/$key has $unpositioned unpositioned placeholders", unpositioned <= 1)
            }
        }
    }

    @Test
    fun values_haveNoUnescapedQuotesOrLonePercent() {
        // aapt2 は assemble でしか走らず test では黙っているので、ここで前倒しに落とす。
        val all = mapOf("values" to defaultStrings) + localeStrings.mapKeys { "values-${it.key}" }
        val unescapedApos = Regex("""(?<!\\)'""")
        val unescapedQuote = Regex("""(?<!\\)"""")
        val lonePercent = Regex("""(?<!%)%(?![\d%sd])""")
        for ((dir, strings) in all) {
            for ((key, entry) in strings) {
                val v = entry.value
                val wrapped = v.length >= 2 && v.startsWith("\"") && v.endsWith("\"")
                if (!wrapped) {
                    assertFalse("$dir/$key has unescaped ' (write \\')", unescapedApos.containsMatchIn(v))
                    assertFalse("$dir/$key has unescaped \" (write \\\")", unescapedQuote.containsMatchIn(v))
                }
                assertFalse("$dir/$key has a lone % (write %% or a placeholder)", lonePercent.containsMatchIn(v))
            }
        }
    }

    @Test
    fun serverLocaleName_matchesItsDirectory() {
        // GkillLocale.serverLocaleName はこの値をサーバーへ送る。ディレクトリ名とずれると
        // UI とサーバー文言の言語が割れる（既定は ja = サーバーのフォールバックと同じ）。
        assertEquals("ja", defaultStrings.getValue("server_locale_name").value)
        for ((locale, strings) in localeStrings) {
            assertEquals("values-$locale", locale, strings.getValue("server_locale_name").value)
        }
    }
}
