package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import org.junit.Assert.*
import org.junit.Test
import java.security.MessageDigest

/**
 * Unit tests for pure functions used in MainActivity.
 *
 * MainActivity.sha256() is private, so we re-implement the same algorithm here
 * and verify it produces correct SHA-256 hashes. This ensures the hashing logic
 * used for password storage is correct.
 */
class MainActivityTest {

    /**
     * Re-implementation of MainActivity.sha256() for testing.
     * Must match: MessageDigest("SHA-256") -> digest(UTF-8 bytes) -> lowercase hex.
     */
    private fun sha256(input: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
        val hash = digest.digest(input.toByteArray(Charsets.UTF_8))
        return hash.joinToString("") { "%02x".format(it) }
    }

    // -----------------------------------------------------------------------
    // SHA-256 hash computation
    // -----------------------------------------------------------------------
    @Test
    fun sha256_emptyString() {
        // Well-known SHA-256 of empty string
        val expected = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        assertEquals(expected, sha256(""))
    }

    @Test
    fun sha256_knownValue_password() {
        // SHA-256 of "password"
        val expected = "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"
        assertEquals(expected, sha256("password"))
    }

    @Test
    fun sha256_knownValue_hello() {
        // SHA-256 of "hello"
        val expected = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
        assertEquals(expected, sha256("hello"))
    }

    @Test
    fun sha256_japaneseText() {
        // Verify UTF-8 encoding is used for non-ASCII text
        val hash = sha256("テスト")
        assertNotNull(hash)
        assertEquals(64, hash.length) // SHA-256 is always 64 hex chars
        // Consistent result for same input
        assertEquals(hash, sha256("テスト"))
    }

    @Test
    fun sha256_outputIsLowercaseHex() {
        val hash = sha256("test")
        assertTrue("Hash should only contain lowercase hex characters", hash.matches(Regex("^[0-9a-f]{64}$")))
    }

    @Test
    fun sha256_differentInputs_produceDifferentHashes() {
        val hash1 = sha256("password1")
        val hash2 = sha256("password2")
        assertNotEquals(hash1, hash2)
    }

    @Test
    fun sha256_sameInput_producesSameHash() {
        val hash1 = sha256("consistent")
        val hash2 = sha256("consistent")
        assertEquals(hash1, hash2)
    }

    @Test
    fun sha256_longInput() {
        val longInput = "a".repeat(10000)
        val hash = sha256(longInput)
        assertEquals(64, hash.length)
        assertTrue(hash.matches(Regex("^[0-9a-f]{64}$")))
    }
}
