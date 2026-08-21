package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import okhttp3.tls.HeldCertificate
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.security.cert.CertificateException

/**
 * Unit tests for GkillServerTrust (fingerprint calculation and pin matching).
 *
 * Uses okhttp-tls HeldCertificate to generate throwaway self-signed certificates.
 * No real server certificate, SAN, or hostname is used.
 */
class GkillServerTrustTest {

    private fun selfSigned() = HeldCertificate.Builder().commonName("gkill-test").build()

    // ─── fingerprint calculation ─────────────────────────────────────────────

    @Test
    fun certSha256Hex_is64LowercaseHexChars() {
        val cert = selfSigned().certificate
        val fp = GkillServerTrust.certSha256Hex(cert)
        assertEquals(64, fp.length)
        assertTrue("fingerprint must be lowercase hex", fp.matches(Regex("^[0-9a-f]{64}$")))
    }

    @Test
    fun certSha256Hex_isDeterministic() {
        val cert = selfSigned().certificate
        assertEquals(GkillServerTrust.certSha256Hex(cert), GkillServerTrust.certSha256Hex(cert))
    }

    @Test
    fun certSha256Hex_differsBetweenCertificates() {
        val a = GkillServerTrust.certSha256Hex(selfSigned().certificate)
        val b = GkillServerTrust.certSha256Hex(selfSigned().certificate)
        assertNotEquals(a, b)
    }

    @Test
    fun formatFingerprint_insertsColonsAndUppercases() {
        val hex = "ab" + "cd".repeat(31)
        val formatted = GkillServerTrust.formatFingerprint(hex)
        assertTrue(formatted.startsWith("AB:CD:"))
        // 32 bytes -> 31 separators
        assertEquals(31, formatted.count { it == ':' })
    }

    // ─── fingerprint comparison ──────────────────────────────────────────────

    @Test
    fun fingerprintsEqual_ignoresCaseAndColons() {
        val hex = GkillServerTrust.certSha256Hex(selfSigned().certificate)
        assertTrue(GkillServerTrust.fingerprintsEqual(hex, hex.uppercase()))
        assertTrue(GkillServerTrust.fingerprintsEqual(hex, GkillServerTrust.formatFingerprint(hex)))
    }

    @Test
    fun fingerprintsEqual_falseForDifferentValues() {
        val a = GkillServerTrust.certSha256Hex(selfSigned().certificate)
        val b = "00".repeat(32)
        assertFalse(GkillServerTrust.fingerprintsEqual(a, b))
    }

    @Test
    fun fingerprintsEqual_falseForEmptyOrLengthMismatch() {
        assertFalse(GkillServerTrust.fingerprintsEqual("", ""))
        assertFalse(GkillServerTrust.fingerprintsEqual("abcd", "ab"))
    }

    // ─── trust manager: pin match ────────────────────────────────────────────

    @Test
    fun trustManager_pinMatch_isAccepted() {
        val cert = selfSigned().certificate
        val fp = GkillServerTrust.certSha256Hex(cert)
        val trust = GkillServerTrust(fp)

        // Must not throw: platform rejects the self-signed cert, then the pin matches.
        trust.trustManager().checkServerTrusted(arrayOf(cert), "RSA")

        assertTrue(trust.lastServerTrusted)
        assertEquals(fp, trust.lastLeafSha256)
    }

    // ─── trust manager: pin mismatch ─────────────────────────────────────────

    @Test
    fun trustManager_pinMismatch_isRejectedButFingerprintCaptured() {
        val cert = selfSigned().certificate
        val fp = GkillServerTrust.certSha256Hex(cert)
        val trust = GkillServerTrust("00".repeat(32))

        try {
            trust.trustManager().checkServerTrusted(arrayOf(cert), "RSA")
            fail("expected CertificateException for a mismatched pin")
        } catch (_: CertificateException) {
            // expected
        }

        assertFalse(trust.lastServerTrusted)
        // Even a rejected cert has its fingerprint captured, for the learning UI.
        assertEquals(fp, trust.lastLeafSha256)
    }

    // ─── trust manager: no pin (background-equivalent) ───────────────────────

    @Test
    fun trustManager_noPin_selfSignedIsRejected() {
        val cert = selfSigned().certificate
        val trust = GkillServerTrust(null)

        try {
            trust.trustManager().checkServerTrusted(arrayOf(cert), "RSA")
            fail("expected CertificateException when no pin is stored")
        } catch (_: CertificateException) {
            // expected: self-signed cert is not in the platform trust store
        }

        assertFalse(trust.lastServerTrusted)
        assertEquals(GkillServerTrust.certSha256Hex(cert), trust.lastLeafSha256)
    }

    // ─── host key derivation ─────────────────────────────────────────────────

    @Test
    fun hostKeyOf_usesHostAndPort() {
        assertEquals("localhost:9999", GkillServerTrust.hostKeyOf("http://localhost:9999"))
        assertEquals("example.com:8443", GkillServerTrust.hostKeyOf("https://example.com:8443/api"))
    }

    @Test
    fun hostKeyOf_omitsImplicitPort() {
        assertEquals("example.com", GkillServerTrust.hostKeyOf("https://example.com/"))
    }

    @Test
    fun hostKeyOf_lowercasesHost() {
        assertEquals("example.com:9999", GkillServerTrust.hostKeyOf("https://EXAMPLE.com:9999"))
    }
}
