import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    // AGP 9 は Kotlin サポートを内蔵しているため kotlin-android プラグインは適用しない
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlinx.serialization)
}

android {
    namespace = "com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.watch"
    // androidx 1.19.0 系が compileSdk 37 以上を要求する
    compileSdk = 37

    testOptions {
        // parseTemplates / parsePlayingTimeisList が android.util.Log を呼ぶため、
        // JVM単体テストでは Log をno-op化する（モックされていないandroid APIで落とさない）。
        // これが無いと Log.w/Log.e が throw し、失敗系のテストを @Ignore で落とすことになる
        // （実際に GkillWearClientTest の2本がそれで無効化されていた）。phone_companion と同じ設定。
        unitTests.isReturnDefaultValues = true
    }

    defaultConfig {
        // Must match phone_companion applicationId for Wearable MessageClient to work
        applicationId = "com.mt3hr.gkill.wear"
        minSdk = 30  // Wear OS 3+ (Pixel Watch 2 runs Wear OS 4)
        targetSdk = 36
        versionCode = (findProperty("versionCode") as? String)?.toIntOrNull() ?: 1
        versionName = (findProperty("versionName") as? String) ?: "1.0.0"
    }

    buildFeatures {
        compose = true
    }

    androidResources {
        // UI 文字列は res/values*/strings.xml の7言語（既定 ja）。依存ライブラリが持つ80言語超の
        // リソースを落とし、アプリ未対応の言語の端末でフレームワーク部品だけ別言語になる混在を防ぐ。
        // "ja" は依存側の values-ja を残すために要る。Wear OS にアプリ別言語設定は無いので
        // generateLocaleConfig は付けない（phone_companion と違う点）。
        localeFilters += listOf("ja", "en", "zh", "ko", "es", "fr", "de")
    }

    // リリース署名。鍵の受け渡しと未設定時の止まり方は src/android/app/build.gradle.kts の
    // 同名ブロックと同じ (指摘 F-006)。
    val gkillSigningProp = { name: String -> (findProperty(name) as? String) ?: System.getenv(name) }
    val gkillReleaseKeystore = gkillSigningProp("GKILL_RELEASE_KEYSTORE")
    if (gkillReleaseKeystore != null) {
        signingConfigs {
            create("release") {
                storeFile = file(gkillReleaseKeystore)
                storePassword = gkillSigningProp("GKILL_RELEASE_KEYSTORE_PASSWORD")
                keyAlias = gkillSigningProp("GKILL_RELEASE_KEY_ALIAS")
                keyPassword = gkillSigningProp("GKILL_RELEASE_KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
            signingConfig = signingConfigs.findByName("release")
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlin {
        compilerOptions {
            jvmTarget.set(JvmTarget.JVM_17)
        }
    }
}

dependencies {
    implementation(libs.play.services.wearable)
    implementation(libs.wear.compose.material)
    implementation(libs.wear.compose.foundation)
    implementation(libs.wear.compose.navigation)
    implementation(libs.androidx.activity.compose)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.kotlinx.coroutines.android)
    implementation(libs.kotlinx.coroutines.play.services)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.androidx.core.ktx)
    implementation(libs.wear.tiles)
    implementation(libs.protolayout)
    implementation(libs.protolayout.material)
    implementation(libs.concurrent.futures)
    testImplementation(libs.junit)
    testImplementation(libs.mockk)
    testImplementation(libs.kotlinx.coroutines.test)
}
