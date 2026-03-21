package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model

import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

/**
 * Unit tests for TemplateNode serialization/deserialization using kotlinx.serialization.
 */
class TemplateNodeTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun deserialize_leafNode() {
        val jsonStr = """{
            "name": "quick_memo",
            "id": "id-123",
            "title": "Quick Memo",
            "template": "/m test",
            "key": "quick_memo_key",
            "is_dir": false,
            "is_open_default": false
        }"""

        val node = json.decodeFromString<TemplateNode>(jsonStr)

        assertEquals("quick_memo", node.name)
        assertEquals("id-123", node.id)
        assertEquals("Quick Memo", node.title)
        assertEquals("/m test", node.template)
        assertEquals("quick_memo_key", node.key)
        assertFalse(node.is_dir)
        assertFalse(node.is_open_default)
        assertNull(node.children)
    }

    @Test
    fun deserialize_directoryNodeWithChildren() {
        val jsonStr = """{
            "name": "memo_dir",
            "title": "Memo",
            "template": "",
            "key": "memo_dir_key",
            "is_dir": true,
            "is_open_default": true,
            "children": [
                {
                    "name": "child1",
                    "title": "Child 1",
                    "template": "/m child1 text",
                    "key": "child1_key",
                    "is_dir": false,
                    "is_open_default": false
                }
            ]
        }"""

        val node = json.decodeFromString<TemplateNode>(jsonStr)

        assertEquals("memo_dir", node.name)
        assertTrue(node.is_dir)
        assertTrue(node.is_open_default)
        assertNotNull(node.children)
        assertEquals(1, node.children!!.size)
        assertEquals("child1", node.children!![0].name)
        assertEquals("/m child1 text", node.children!![0].template)
    }

    @Test
    fun deserialize_withNullId() {
        val jsonStr = """{
            "name": "no_id",
            "title": "No ID",
            "template": "",
            "key": "key",
            "is_dir": false,
            "is_open_default": false
        }"""

        val node = json.decodeFromString<TemplateNode>(jsonStr)

        assertNull(node.id)
    }

    @Test
    fun deserialize_withNullChildren() {
        val jsonStr = """{
            "name": "leaf",
            "title": "Leaf",
            "template": "/m test",
            "key": "key",
            "is_dir": false,
            "is_open_default": false
        }"""

        val node = json.decodeFromString<TemplateNode>(jsonStr)

        assertNull(node.children)
    }

    @Test
    fun serialize_leafNode() {
        val node = TemplateNode(
            name = "test_leaf",
            id = "id-456",
            title = "Test Leaf",
            template = "/m hello",
            key = "test_key",
            is_dir = false,
            is_open_default = false
        )

        val jsonStr = json.encodeToString(TemplateNode.serializer(), node)

        assertTrue(jsonStr.contains("\"name\":\"test_leaf\""))
        assertTrue(jsonStr.contains("\"title\":\"Test Leaf\""))
        assertTrue(jsonStr.contains("\"template\":\"/m hello\""))
        // Note: kotlinx.serialization may omit default values (is_dir=false, id=null)
    }

    @Test
    fun serialize_directoryNodeWithChildren() {
        val child = TemplateNode(
            name = "child",
            title = "Child",
            template = "/m child text",
            key = "child_key",
            is_dir = false,
            is_open_default = false
        )
        val parent = TemplateNode(
            name = "parent_dir",
            title = "Parent",
            template = "",
            children = listOf(child),
            key = "parent_key",
            is_dir = true,
            is_open_default = true
        )

        val jsonStr = json.encodeToString(TemplateNode.serializer(), parent)

        assertTrue(jsonStr.contains("\"is_dir\":true"))
        assertTrue(jsonStr.contains("\"children\":["))
        assertTrue(jsonStr.contains("\"name\":\"child\""))
    }

    @Test
    fun roundTrip_preservesAllFields() {
        val original = TemplateNode(
            name = "roundtrip",
            id = "rt-id",
            title = "Round Trip",
            template = "/m roundtrip test",
            children = listOf(
                TemplateNode(
                    name = "inner",
                    title = "Inner",
                    template = "/t inner task",
                    key = "inner_key",
                    is_dir = false,
                    is_open_default = false
                )
            ),
            key = "rt_key",
            is_dir = true,
            is_open_default = true
        )

        val jsonStr = json.encodeToString(TemplateNode.serializer(), original)
        val deserialized = json.decodeFromString<TemplateNode>(jsonStr)

        assertEquals(original, deserialized)
    }

    @Test
    fun defaultValues_areApplied() {
        val node = TemplateNode()

        assertEquals("", node.name)
        assertNull(node.id)
        assertEquals("", node.title)
        assertEquals("", node.template)
        assertNull(node.children)
        assertEquals("", node.key)
        assertFalse(node.is_dir)
        assertFalse(node.is_open_default)
    }

    @Test
    fun copy_modifiesSelectedFields() {
        val original = TemplateNode(
            name = "original",
            title = "Original",
            template = "/m original",
            key = "orig_key",
            is_dir = false,
            is_open_default = false
        )

        val modified = original.copy(
            name = "modified",
            children = listOf(TemplateNode(name = "new_child"))
        )

        assertEquals("modified", modified.name)
        assertEquals("Original", modified.title)
        assertNotNull(modified.children)
        assertEquals(1, modified.children!!.size)
    }

    @Test
    fun deserialize_unknownFields_areIgnored() {
        val jsonStr = """{
            "name": "test",
            "title": "Test",
            "template": "",
            "key": "key",
            "is_dir": false,
            "is_open_default": false,
            "unknown_field": "should be ignored",
            "another_unknown": 42
        }"""

        val node = json.decodeFromString<TemplateNode>(jsonStr)

        assertEquals("test", node.name)
    }
}
