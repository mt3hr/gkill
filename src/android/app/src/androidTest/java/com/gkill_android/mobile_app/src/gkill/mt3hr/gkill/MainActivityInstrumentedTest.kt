package com.gkill_android.mobile_app.src.gkill.mt3hr.gkill

import androidx.test.platform.app.InstrumentationRegistry
import androidx.test.ext.junit.runners.AndroidJUnit4

import org.junit.Test
import org.junit.runner.RunWith

import org.junit.Assert.*

/**
 * Instrumented tests for MainActivity.
 * These run on an Android device or emulator.
 */
@RunWith(AndroidJUnit4::class)
class MainActivityInstrumentedTest {

    @Test
    fun packageName_isCorrect() {
        val appContext = InstrumentationRegistry.getInstrumentation().targetContext
        assertEquals(
            "com.gkill_android.mobile_app.src.gkill.mt3hr.gkill",
            appContext.packageName
        )
    }

    @Test
    fun appContext_isNotNull() {
        val appContext = InstrumentationRegistry.getInstrumentation().targetContext
        assertNotNull(appContext)
    }

    @Test
    fun appContext_hasFilesDir() {
        val appContext = InstrumentationRegistry.getInstrumentation().targetContext
        assertNotNull(appContext.filesDir)
        assertTrue(appContext.filesDir.exists())
    }

    @Test
    fun appContext_hasAssets() {
        val appContext = InstrumentationRegistry.getInstrumentation().targetContext
        assertNotNull(appContext.assets)
    }

    @Test
    fun appContext_hasCacheDir() {
        val appContext = InstrumentationRegistry.getInstrumentation().targetContext
        assertNotNull(appContext.cacheDir)
    }
}
