package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import android.content.Context
import android.content.SharedPreferences
import io.mockk.every
import io.mockk.mockk
import io.mockk.slot
import io.mockk.verify
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test

/**
 * Unit tests for GkillCredentialStore using MockK to mock SharedPreferences.
 */
class GkillCredentialStoreTest {

    private lateinit var context: Context
    private lateinit var prefs: SharedPreferences
    private lateinit var editor: SharedPreferences.Editor
    private lateinit var store: GkillCredentialStore

    // In-memory storage backing the mocked SharedPreferences
    private val storage = mutableMapOf<String, String?>()

    @Before
    fun setUp() {
        storage.clear()
        editor = mockk(relaxed = true)
        prefs = mockk()
        context = mockk()

        // Mock editor.putString to store values
        val keySlot = slot<String>()
        val valueSlot = slot<String>()
        every { editor.putString(capture(keySlot), capture(valueSlot)) } answers {
            storage[keySlot.captured] = valueSlot.captured
            editor
        }

        // Mock editor.remove to clear values
        val removeKeySlot = slot<String>()
        every { editor.remove(capture(removeKeySlot)) } answers {
            storage.remove(removeKeySlot.captured)
            editor
        }

        every { prefs.edit() } returns editor
        every { prefs.getString(any(), any()) } answers {
            val key = firstArg<String>()
            val default = secondArg<String?>()
            storage[key] ?: default
        }

        every {
            context.getSharedPreferences("gkill_wear_prefs", Context.MODE_PRIVATE)
        } returns prefs

        store = GkillCredentialStore(context)
    }

    // -----------------------------------------------------------------------
    // Server URL
    // -----------------------------------------------------------------------
    @Test
    fun getServerUrl_returnsDefault_whenNotSet() {
        val url = store.getServerUrl()
        assertEquals("http://localhost:9999", url)
    }

    @Test
    fun setServerUrl_storesValue() {
        store.setServerUrl("https://example.com:8080")
        verify { editor.putString("server_url", "https://example.com:8080") }
        verify { editor.apply() }
    }

    @Test
    fun getServerUrl_returnsStoredValue() {
        storage["server_url"] = "https://myserver.local"
        val url = store.getServerUrl()
        assertEquals("https://myserver.local", url)
    }

    // -----------------------------------------------------------------------
    // User ID
    // -----------------------------------------------------------------------
    @Test
    fun getUserId_returnsEmpty_whenNotSet() {
        val userId = store.getUserId()
        assertEquals("", userId)
    }

    @Test
    fun setUserId_storesValue() {
        store.setUserId("admin")
        verify { editor.putString("user_id", "admin") }
        verify { editor.apply() }
    }

    @Test
    fun getUserId_returnsStoredValue() {
        storage["user_id"] = "testuser"
        assertEquals("testuser", store.getUserId())
    }

    // -----------------------------------------------------------------------
    // Password SHA256
    // -----------------------------------------------------------------------
    @Test
    fun getPasswordSha256_returnsEmpty_whenNotSet() {
        assertEquals("", store.getPasswordSha256())
    }

    @Test
    fun setPasswordSha256_storesValue() {
        val hash = "abc123def456"
        store.setPasswordSha256(hash)
        verify { editor.putString("password_sha256", hash) }
        verify { editor.apply() }
    }

    @Test
    fun getPasswordSha256_returnsStoredValue() {
        storage["password_sha256"] = "sha256hash"
        assertEquals("sha256hash", store.getPasswordSha256())
    }

    // -----------------------------------------------------------------------
    // Session ID
    // -----------------------------------------------------------------------
    @Test
    fun getSessionId_returnsEmpty_whenNotSet() {
        assertEquals("", store.getSessionId())
    }

    @Test
    fun setSessionId_storesValue() {
        store.setSessionId("session-abc")
        verify { editor.putString("session_id", "session-abc") }
        verify { editor.apply() }
    }

    @Test
    fun getSessionId_returnsStoredValue() {
        storage["session_id"] = "my-session"
        assertEquals("my-session", store.getSessionId())
    }

    // -----------------------------------------------------------------------
    // clearSession
    // -----------------------------------------------------------------------
    @Test
    fun clearSession_removesSessionId() {
        storage["session_id"] = "to-be-cleared"
        store.clearSession()
        verify { editor.remove("session_id") }
        verify { editor.apply() }
    }

    // -----------------------------------------------------------------------
    // SharedPreferences name
    // -----------------------------------------------------------------------
    @Test
    fun usesCorrectPreferencesName() {
        // The constructor calls getSharedPreferences with "gkill_wear_prefs"
        verify { context.getSharedPreferences("gkill_wear_prefs", Context.MODE_PRIVATE) }
    }
}
