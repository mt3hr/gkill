package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * Unit tests for WearSubmitLedger (duplicate-submit ledger).
 *
 * Storage and clock are injectable, so persistence and TTL are tested on the JVM
 * without SharedPreferences.
 */
class WearSubmitLedgerTest {

    private class MemoryStorage : WearSubmitLedger.Storage {
        var value = ""
        override fun read(): String = value
        override fun write(value: String) {
            this.value = value
        }
    }

    @Test
    fun unknownText_isNotDuplicate() {
        val ledger = WearSubmitLedger(MemoryStorage())
        assertFalse(ledger.isDuplicate("/m memo"))
    }

    @Test
    fun recordedText_isDuplicate() {
        val ledger = WearSubmitLedger(MemoryStorage())
        ledger.recordSuccess("/m memo")
        assertTrue(ledger.isDuplicate("/m memo"))
        assertFalse(ledger.isDuplicate("/m other"))
    }

    @Test
    fun ledgerPersistsAcrossInstances_viaSharedStorage() {
        // Mirrors process death: a new ledger over the same storage still sees the record.
        val storage = MemoryStorage()
        WearSubmitLedger(storage).recordSuccess("/m memo")
        assertTrue(WearSubmitLedger(storage).isDuplicate("/m memo"))
    }

    @Test
    fun expiredEntry_isNotDuplicate() {
        var now = 1_000_000L
        val ledger = WearSubmitLedger(
            storage = MemoryStorage(),
            ttlMillis = 1000L,
            clock = { now },
        )
        ledger.recordSuccess("/m memo")
        assertTrue(ledger.isDuplicate("/m memo"))
        now += 1001L
        assertFalse(ledger.isDuplicate("/m memo"))
    }

    @Test
    fun maxEntries_evictsOldest() {
        val ledger = WearSubmitLedger(storage = MemoryStorage(), maxEntries = 2)
        ledger.recordSuccess("a")
        ledger.recordSuccess("b")
        ledger.recordSuccess("c")
        // "a" was evicted; "b"/"c" remain
        assertFalse(ledger.isDuplicate("a"))
        assertTrue(ledger.isDuplicate("b"))
        assertTrue(ledger.isDuplicate("c"))
    }

    @Test
    fun reRecordingSameText_keepsSingleEntry() {
        val ledger = WearSubmitLedger(storage = MemoryStorage(), maxEntries = 2)
        ledger.recordSuccess("a")
        ledger.recordSuccess("a")
        ledger.recordSuccess("b")
        // "a" re-recorded, not double-counted, so "b" is still present
        assertTrue(ledger.isDuplicate("a"))
        assertTrue(ledger.isDuplicate("b"))
    }

    @Test
    fun corruptStorage_isTreatedAsEmpty() {
        val storage = MemoryStorage().apply { value = "not-json" }
        val ledger = WearSubmitLedger(storage)
        assertFalse(ledger.isDuplicate("/m memo"))
    }
}
