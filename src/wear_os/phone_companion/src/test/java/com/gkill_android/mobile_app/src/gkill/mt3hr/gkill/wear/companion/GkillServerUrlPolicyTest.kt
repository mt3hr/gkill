package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion

import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * GkillServerUrlPolicy の境界テスト(指摘 F-007)。
 * 平文HTTPはループバックだけ許可し、LAN・公開ホストへの平文は保存前に拒否する。
 */
class GkillServerUrlPolicyTest {

    @Test
    fun httpLoopbackIsAllowed() {
        assertTrue(GkillServerUrlPolicy.isAllowed("http://localhost:9999"))
        assertTrue(GkillServerUrlPolicy.isAllowed("http://127.0.0.1:9999"))
        assertTrue(GkillServerUrlPolicy.isAllowed("http://127.0.0.1"))
        // 127.0.0.0/8 全域がループバック
        assertTrue(GkillServerUrlPolicy.isAllowed("http://127.5.6.7:8080"))
        assertTrue(GkillServerUrlPolicy.isAllowed("http://[::1]:9999"))
        assertTrue(GkillServerUrlPolicy.isAllowed("http://LOCALHOST:9999"))
    }

    @Test
    fun httpNonLoopbackIsRejected() {
        assertFalse(GkillServerUrlPolicy.isAllowed("http://192.168.1.10:9999"))
        assertFalse(GkillServerUrlPolicy.isAllowed("http://10.0.0.2:9999"))
        assertFalse(GkillServerUrlPolicy.isAllowed("http://example.com:9999"))
        assertFalse(GkillServerUrlPolicy.isAllowed("http://203.0.113.7"))
        // 127. を偽装したホスト名・別表記は許可しない
        assertFalse(GkillServerUrlPolicy.isAllowed("http://127.0.0.1.example.com"))
        assertFalse(GkillServerUrlPolicy.isAllowed("http://localhost.example.com"))
    }

    @Test
    fun httpsIsAllowedForAnyHost() {
        assertTrue(GkillServerUrlPolicy.isAllowed("https://example.com:9999"))
        assertTrue(GkillServerUrlPolicy.isAllowed("https://192.168.1.10:9999"))
        assertTrue(GkillServerUrlPolicy.isAllowed("https://localhost:9999"))
    }

    @Test
    fun malformedOrOtherSchemesAreRejected() {
        assertFalse(GkillServerUrlPolicy.isAllowed(""))
        assertFalse(GkillServerUrlPolicy.isAllowed("localhost:9999"))
        assertFalse(GkillServerUrlPolicy.isAllowed("ftp://localhost:9999"))
        assertFalse(GkillServerUrlPolicy.isAllowed("http://"))
        assertFalse(GkillServerUrlPolicy.isAllowed("http:// spaces .example"))
    }
}
