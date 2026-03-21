package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.tile

import android.content.Context
import android.content.SharedPreferences
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.GkillWearClient
import com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch.data.model.TemplateNode
import io.mockk.every
import io.mockk.mockk
import io.mockk.mockkObject
import io.mockk.slot
import io.mockk.unmockkObject
import io.mockk.verify
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

/**
 * Unit tests for TemplateCacheManager.
 * Uses MockK to mock SharedPreferences and Context since they are unavailable in JVM tests.
 */
class TemplateCacheManagerTest {

    private lateinit var context: Context
    private lateinit var prefs: SharedPreferences
    private lateinit var editor: SharedPreferences.Editor

    private val storage = mutableMapOf<String, Any?>()

    @Before
    fun setUp() {
        storage.clear()
        editor = mockk(relaxed = true)
        prefs = mockk()
        context = mockk()

        // Mock editor.putString
        val strKeySlot = slot<String>()
        val strValueSlot = slot<String>()
        every { editor.putString(capture(strKeySlot), capture(strValueSlot)) } answers {
            storage[strKeySlot.captured] = strValueSlot.captured
            editor
        }

        // Mock editor.putLong
        val longKeySlot = slot<String>()
        val longValueSlot = slot<Long>()
        every { editor.putLong(capture(longKeySlot), capture(longValueSlot)) } answers {
            storage[longKeySlot.captured] = longValueSlot.captured
            editor
        }

        every { prefs.edit() } returns editor

        every { prefs.getString(any(), any()) } answers {
            storage[firstArg<String>()] as? String ?: secondArg()
        }

        every { prefs.getLong(any(), any()) } answers {
            storage[firstArg<String>()] as? Long ?: secondArg()
        }

        every {
            context.getSharedPreferences("gkill_tile_cache", Context.MODE_PRIVATE)
        } returns prefs

        // Mock GkillWearClient.parseTemplates since it's called by loadTemplates
        mockkObject(GkillWearClient)
    }

    @After
    fun tearDown() {
        unmockkObject(GkillWearClient)
    }

    // -----------------------------------------------------------------------
    // saveRawJson
    // -----------------------------------------------------------------------
    @Test
    fun saveRawJson_storesJsonAndTimestamp() {
        val testJson = """{"name":"root","children":[]}"""

        TemplateCacheManager.saveRawJson(context, testJson)

        verify { editor.putString("templates_json", testJson) }
        verify { editor.putLong("last_updated", any()) }
        verify { editor.apply() }
    }

    @Test
    fun saveRawJson_usesCorrectPreferenceName() {
        TemplateCacheManager.saveRawJson(context, "{}")

        verify { context.getSharedPreferences("gkill_tile_cache", Context.MODE_PRIVATE) }
    }

    // -----------------------------------------------------------------------
    // loadTemplates
    // -----------------------------------------------------------------------
    @Test
    fun loadTemplates_returnsEmptyList_whenNoDataStored() {
        // getString returns null (no stored JSON)
        every { prefs.getString("templates_json", null) } returns null

        val result = TemplateCacheManager.loadTemplates(context)

        assertTrue(result.isEmpty())
    }

    @Test
    fun loadTemplates_callsParseTemplates_whenDataExists() {
        val storedJson = """{"name":"root","children":[{"name":"memo","title":"Memo","template":"/m test","key":"k","is_dir":false,"is_open_default":false}]}"""
        every { prefs.getString("templates_json", null) } returns storedJson

        val expectedTemplates = listOf(
            TemplateNode(name = "memo", title = "Memo", template = "/m test", key = "k")
        )
        every { GkillWearClient.parseTemplates(storedJson) } returns expectedTemplates

        val result = TemplateCacheManager.loadTemplates(context)

        assertEquals(1, result.size)
        assertEquals("memo", result[0].name)
        verify { GkillWearClient.parseTemplates(storedJson) }
    }

    @Test
    fun loadTemplates_returnsEmptyList_whenParseReturnsEmpty() {
        val storedJson = """{"name":"root"}"""
        every { prefs.getString("templates_json", null) } returns storedJson
        every { GkillWearClient.parseTemplates(storedJson) } returns emptyList()

        val result = TemplateCacheManager.loadTemplates(context)

        assertTrue(result.isEmpty())
    }

    // -----------------------------------------------------------------------
    // Constants
    // -----------------------------------------------------------------------
    @Test
    fun preferenceName_isGkillTileCache() {
        // Verified indirectly: saveRawJson and loadTemplates both use "gkill_tile_cache"
        TemplateCacheManager.saveRawJson(context, "{}")
        verify { context.getSharedPreferences("gkill_tile_cache", Context.MODE_PRIVATE) }
    }
}
