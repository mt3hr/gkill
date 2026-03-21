package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model

import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

/**
 * Unit tests for PlaingTimeIsNode serialization/deserialization using kotlinx.serialization.
 */
class PlaingTimeIsNodeTest {

    private val json = Json { ignoreUnknownKeys = true }

    // -----------------------------------------------------------------------
    // Deserialization
    // -----------------------------------------------------------------------
    @Test
    fun deserialize_fullNode() {
        val jsonStr = """{
            "id": "timeis-001",
            "rep_name": "my_repo",
            "title": "Working on gkill",
            "start_time": "2026-03-21T10:00:00+09:00",
            "data_type": "timeis",
            "is_deleted": false
        }"""

        val node = json.decodeFromString<PlaingTimeIsNode>(jsonStr)

        assertEquals("timeis-001", node.id)
        assertEquals("my_repo", node.rep_name)
        assertEquals("Working on gkill", node.title)
        assertEquals("2026-03-21T10:00:00+09:00", node.start_time)
        assertEquals("timeis", node.data_type)
        assertFalse(node.is_deleted)
    }

    @Test
    fun deserialize_deletedNode() {
        val jsonStr = """{
            "id": "timeis-002",
            "rep_name": "repo",
            "title": "Deleted task",
            "start_time": "2026-01-01T00:00:00Z",
            "data_type": "timeis",
            "is_deleted": true
        }"""

        val node = json.decodeFromString<PlaingTimeIsNode>(jsonStr)

        assertTrue(node.is_deleted)
    }

    @Test
    fun deserialize_unknownFields_areIgnored() {
        val jsonStr = """{
            "id": "timeis-003",
            "rep_name": "repo",
            "title": "Test",
            "start_time": "",
            "data_type": "timeis",
            "is_deleted": false,
            "extra_field": "should be ignored",
            "another_extra": 42
        }"""

        val node = json.decodeFromString<PlaingTimeIsNode>(jsonStr)

        assertEquals("timeis-003", node.id)
    }

    @Test
    fun deserialize_emptyStrings() {
        val jsonStr = """{
            "id": "",
            "rep_name": "",
            "title": "",
            "start_time": "",
            "data_type": "",
            "is_deleted": false
        }"""

        val node = json.decodeFromString<PlaingTimeIsNode>(jsonStr)

        assertEquals("", node.id)
        assertEquals("", node.rep_name)
        assertEquals("", node.title)
        assertEquals("", node.start_time)
        assertEquals("", node.data_type)
    }

    // -----------------------------------------------------------------------
    // Serialization
    // -----------------------------------------------------------------------
    @Test
    fun serialize_fullNode() {
        val node = PlaingTimeIsNode(
            id = "timeis-010",
            rep_name = "test_repo",
            title = "Serialization test",
            start_time = "2026-03-21T15:30:00+09:00",
            data_type = "timeis",
            is_deleted = false
        )

        val jsonStr = json.encodeToString(PlaingTimeIsNode.serializer(), node)

        assertTrue(jsonStr.contains("\"id\":\"timeis-010\""))
        assertTrue(jsonStr.contains("\"rep_name\":\"test_repo\""))
        assertTrue(jsonStr.contains("\"title\":\"Serialization test\""))
        assertTrue(jsonStr.contains("\"start_time\":\"2026-03-21T15:30:00+09:00\""))
        assertTrue(jsonStr.contains("\"data_type\":\"timeis\""))
        // Note: is_deleted=false may be omitted by kotlinx.serialization (default value)
    }

    @Test
    fun serialize_deletedNode() {
        val node = PlaingTimeIsNode(
            id = "timeis-011",
            rep_name = "repo",
            title = "Deleted",
            start_time = "",
            data_type = "timeis",
            is_deleted = true
        )

        val jsonStr = json.encodeToString(PlaingTimeIsNode.serializer(), node)

        assertTrue(jsonStr.contains("\"is_deleted\":true"))
    }

    // -----------------------------------------------------------------------
    // Round-trip
    // -----------------------------------------------------------------------
    @Test
    fun roundTrip_preservesAllFields() {
        val original = PlaingTimeIsNode(
            id = "roundtrip-001",
            rep_name = "rt_repo",
            title = "Round trip test",
            start_time = "2026-06-15T08:00:00+09:00",
            data_type = "timeis",
            is_deleted = false
        )

        val jsonStr = json.encodeToString(PlaingTimeIsNode.serializer(), original)
        val deserialized = json.decodeFromString<PlaingTimeIsNode>(jsonStr)

        assertEquals(original, deserialized)
    }

    @Test
    fun roundTrip_deletedNode() {
        val original = PlaingTimeIsNode(
            id = "roundtrip-002",
            rep_name = "repo",
            title = "Deleted round trip",
            start_time = "2026-01-01T00:00:00Z",
            data_type = "timeis",
            is_deleted = true
        )

        val jsonStr = json.encodeToString(PlaingTimeIsNode.serializer(), original)
        val deserialized = json.decodeFromString<PlaingTimeIsNode>(jsonStr)

        assertEquals(original, deserialized)
    }

    // -----------------------------------------------------------------------
    // Default values
    // -----------------------------------------------------------------------
    @Test
    fun defaultValues_areApplied() {
        val node = PlaingTimeIsNode()

        assertEquals("", node.id)
        assertEquals("", node.rep_name)
        assertEquals("", node.title)
        assertEquals("", node.start_time)
        assertEquals("", node.data_type)
        assertFalse(node.is_deleted)
    }

    // -----------------------------------------------------------------------
    // Data class features
    // -----------------------------------------------------------------------
    @Test
    fun copy_modifiesSelectedFields() {
        val original = PlaingTimeIsNode(
            id = "copy-001",
            rep_name = "repo",
            title = "Original",
            start_time = "2026-03-21T10:00:00+09:00",
            data_type = "timeis",
            is_deleted = false
        )

        val modified = original.copy(
            title = "Modified",
            is_deleted = true
        )

        assertEquals("copy-001", modified.id)
        assertEquals("Modified", modified.title)
        assertTrue(modified.is_deleted)
        // Unmodified fields remain the same
        assertEquals("repo", modified.rep_name)
        assertEquals("2026-03-21T10:00:00+09:00", modified.start_time)
    }

    @Test
    fun equals_sameValues_areEqual() {
        val node1 = PlaingTimeIsNode(id = "eq-001", title = "Same")
        val node2 = PlaingTimeIsNode(id = "eq-001", title = "Same")
        assertEquals(node1, node2)
    }

    @Test
    fun equals_differentValues_areNotEqual() {
        val node1 = PlaingTimeIsNode(id = "eq-001", title = "One")
        val node2 = PlaingTimeIsNode(id = "eq-002", title = "Two")
        assertNotEquals(node1, node2)
    }

    @Test
    fun hashCode_sameValues_areEqual() {
        val node1 = PlaingTimeIsNode(id = "hc-001", title = "Hash")
        val node2 = PlaingTimeIsNode(id = "hc-001", title = "Hash")
        assertEquals(node1.hashCode(), node2.hashCode())
    }
}
