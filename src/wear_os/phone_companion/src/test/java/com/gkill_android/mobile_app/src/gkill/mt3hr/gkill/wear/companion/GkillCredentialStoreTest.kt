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

    /**
     * Android Keystoreはユニットテストでは使えないため、
     * 前後を目印で挟むだけの差し替え可能な暗号器を使う。
     * 「秘密値がそのままの形では保存されない」ことを検証するのが目的。
     */
    private class FakeSecretCipher : GkillSecretCipher {
        override fun encrypt(plain: String): String? = "$PREFIX$plain$SUFFIX"

        override fun decrypt(stored: String): String? =
            stored.removePrefix(PREFIX).removeSuffix(SUFFIX)

        override fun isEncrypted(stored: String): Boolean = stored.startsWith(PREFIX)

        companion object {
            const val PREFIX = "fakeenc:"
            const val SUFFIX = ":end"
        }
    }

    private fun encrypted(plain: String): String =
        "${FakeSecretCipher.PREFIX}$plain${FakeSecretCipher.SUFFIX}"

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

        store = GkillCredentialStore(context, FakeSecretCipher())
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
    fun setPasswordSha256_storesEncryptedValue() {
        val hash = "abc123def456"
        store.setPasswordSha256(hash)
        verify { editor.putString("password_sha256", encrypted(hash)) }
        verify { editor.apply() }
        assertNotEquals(hash, storage["password_sha256"])
    }

    @Test
    fun getPasswordSha256_returnsDecryptedValue() {
        storage["password_sha256"] = encrypted("sha256hash")
        assertEquals("sha256hash", store.getPasswordSha256())
    }

    @Test
    fun getPasswordSha256_readsLegacyPlainTextValue() {
        // 暗号化前のバージョンが書いた値も読めること
        storage["password_sha256"] = "sha256hash"
        assertEquals("sha256hash", store.getPasswordSha256())
    }

    @Test
    fun setPasswordSha256_removesKey_whenEmpty() {
        storage["password_sha256"] = encrypted("sha256hash")
        store.setPasswordSha256("")
        verify { editor.remove("password_sha256") }
        assertEquals("", store.getPasswordSha256())
    }

    @Test
    fun getPasswordSha256_returnsEmpty_whenDecryptFails() {
        // 鍵が変わって復号できなくなった場合は空扱いにして再ログインさせる
        storage["password_sha256"] = "${FakeSecretCipher.PREFIX}broken"
        val brokenStore = GkillCredentialStore(context, object : GkillSecretCipher {
            override fun encrypt(plain: String): String? = null
            override fun decrypt(stored: String): String? = null
            override fun isEncrypted(stored: String): Boolean = true
        })
        assertEquals("", brokenStore.getPasswordSha256())
    }

    // -----------------------------------------------------------------------
    // Session ID
    // -----------------------------------------------------------------------
    @Test
    fun getSessionId_returnsEmpty_whenNotSet() {
        assertEquals("", store.getSessionId())
    }

    @Test
    fun setSessionId_storesEncryptedValue() {
        store.setSessionId("session-abc")
        verify { editor.putString("session_id", encrypted("session-abc")) }
        verify { editor.apply() }
        assertNotEquals("session-abc", storage["session_id"])
    }

    @Test
    fun getSessionId_returnsDecryptedValue() {
        storage["session_id"] = encrypted("my-session")
        assertEquals("my-session", store.getSessionId())
    }

    @Test
    fun getSessionId_readsLegacyPlainTextValue() {
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
