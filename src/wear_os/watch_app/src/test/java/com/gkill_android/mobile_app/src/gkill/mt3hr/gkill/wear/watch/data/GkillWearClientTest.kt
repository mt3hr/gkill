package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data

import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode
import org.junit.Assert.*
import org.junit.Test

/**
 * Unit tests for GkillWearClient companion object static methods:
 * parseTemplates and parsePlaingTimeisList.
 */
class GkillWearClientTest {

    // ─── parseTemplates ────────────────────────────────────────────────────

    @Test
    fun parseTemplates_validJson_returnsFilteredTemplates() {
        // Root node with children: one directory containing a leaf ending with "！"
        val json = """{
            "name": "root",
            "title": "Root",
            "template": "",
            "key": "root",
            "is_dir": true,
            "is_open_default": false,
            "children": [
                {
                    "name": "memo_dir",
                    "title": "Memo",
                    "template": "",
                    "key": "memo_dir",
                    "is_dir": true,
                    "is_open_default": false,
                    "children": [
                        {
                            "name": "quick_memo",
                            "title": "Quick Memo",
                            "template": "/m \u30e1\u30e2\uff01",
                            "key": "quick_memo",
                            "is_dir": false,
                            "is_open_default": false
                        },
                        {
                            "name": "no_bang",
                            "title": "No Bang",
                            "template": "/m memo without bang",
                            "key": "no_bang",
                            "is_dir": false,
                            "is_open_default": false
                        }
                    ]
                }
            ]
        }"""

        val result = GkillWearClient.parseTemplates(json)

        // Only templates ending with "！" pass the filter
        assertEquals(1, result.size)
        assertTrue(result[0].is_dir)
        assertEquals("memo_dir", result[0].name)
        assertEquals(1, result[0].children?.size)
        assertEquals("quick_memo", result[0].children!![0].name)
    }

    @Test
    fun parseTemplates_emptyChildren_returnsEmptyList() {
        val json = """{
            "name": "root",
            "title": "Root",
            "template": "",
            "key": "root",
            "is_dir": true,
            "is_open_default": false,
            "children": []
        }"""

        val result = GkillWearClient.parseTemplates(json)

        assertTrue(result.isEmpty())
    }

    @Test
    @org.junit.Ignore("android.util.Log not available in JVM unit tests - parseTemplates catches exception internally but Log.e throws")
    fun parseTemplates_malformedJson_returnsEmptyList() {
        val result = GkillWearClient.parseTemplates("{invalid json")

        assertTrue(result.isEmpty())
    }

    @Test
    fun parseTemplates_errorPrefix_returnsEmptyList() {
        val result = GkillWearClient.parseTemplates("ERROR:something went wrong")

        assertTrue(result.isEmpty())
    }

    @Test
    fun parseTemplates_dirWithNoMatchingLeaves_isFilteredOut() {
        val json = """{
            "name": "root",
            "title": "Root",
            "template": "",
            "key": "root",
            "is_dir": true,
            "is_open_default": false,
            "children": [
                {
                    "name": "empty_dir",
                    "title": "Empty Dir",
                    "template": "",
                    "key": "empty_dir",
                    "is_dir": true,
                    "is_open_default": false,
                    "children": [
                        {
                            "name": "no_bang",
                            "title": "No Bang",
                            "template": "/m just a memo",
                            "key": "no_bang",
                            "is_dir": false,
                            "is_open_default": false
                        }
                    ]
                }
            ]
        }"""

        val result = GkillWearClient.parseTemplates(json)

        assertTrue(result.isEmpty())
    }

    @Test
    fun parseTemplates_templateWithTrailingNewlineAndBang_isIncluded() {
        val json = """{
            "name": "root",
            "title": "Root",
            "template": "",
            "key": "root",
            "is_dir": true,
            "is_open_default": false,
            "children": [
                {
                    "name": "leaf",
                    "title": "Leaf",
                    "template": "/m test\uff01\n",
                    "key": "leaf",
                    "is_dir": false,
                    "is_open_default": false
                }
            ]
        }"""

        val result = GkillWearClient.parseTemplates(json)

        // trimEnd('\n') then check endsWith("！") -- should pass
        assertEquals(1, result.size)
        assertEquals("leaf", result[0].name)
    }

    // ─── parsePlaingTimeisList ─────────────────────────────────────────────

    @Test
    fun parsePlaingTimeisList_validJson_returnsList() {
        val json = """[
            {
                "id": "uuid-1",
                "rep_name": "rep1",
                "title": "Working",
                "start_time": "2026-03-21T10:00:00+09:00",
                "data_type": "timeis",
                "is_deleted": false
            },
            {
                "id": "uuid-2",
                "rep_name": "rep2",
                "title": "Meeting",
                "start_time": "2026-03-21T14:00:00+09:00",
                "data_type": "timeis",
                "is_deleted": false
            }
        ]"""

        val result = GkillWearClient.parsePlaingTimeisList(json)

        assertEquals(2, result.size)
        assertEquals("uuid-1", result[0].id)
        assertEquals("Working", result[0].title)
        assertEquals("uuid-2", result[1].id)
        assertEquals("Meeting", result[1].title)
    }

    @Test
    fun parsePlaingTimeisList_emptyArray_returnsEmptyList() {
        val result = GkillWearClient.parsePlaingTimeisList("[]")

        assertTrue(result.isEmpty())
    }

    @Test
    @org.junit.Ignore("android.util.Log not available in JVM unit tests - parsePlaingTimeisList catches exception internally but Log.e throws")
    fun parsePlaingTimeisList_malformedJson_returnsEmptyList() {
        val result = GkillWearClient.parsePlaingTimeisList("not json at all")

        assertTrue(result.isEmpty())
    }

    @Test
    fun parsePlaingTimeisList_errorPrefix_returnsEmptyList() {
        val result = GkillWearClient.parsePlaingTimeisList("ERROR:server error")

        assertTrue(result.isEmpty())
    }

    // ─── Companion constants ───────────────────────────────────────────────

    @Test
    fun responsePathConstants_areCorrect() {
        assertEquals("/gkill/templates", GkillWearClient.RESPONSE_PATH_TEMPLATES)
        assertEquals("/gkill/submit_result", GkillWearClient.RESPONSE_PATH_SUBMIT_RESULT)
        assertEquals("/gkill/plaing_timeis", GkillWearClient.RESPONSE_PATH_PLAING_TIMEIS)
        assertEquals("/gkill/end_timeis_result", GkillWearClient.RESPONSE_PATH_END_TIMEIS_RESULT)
    }
}
