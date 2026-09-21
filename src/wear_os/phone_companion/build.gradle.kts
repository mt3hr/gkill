import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    // AGP 9 は Kotlin サポートを内蔵しているため kotlin-android プラグインは適用しない
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlinx.serialization)
}

android {
    namespace = "com.gkill_android.mobile_app.src.gkill.mt3hr.gkill.wear.companion"
    // androidx 1.19.0 系が compileSdk 37 以上を要求する
    compileSdk = 37

    testOptions {
        // getPlayingTimeis 等が android.util.Log を呼ぶため、JVM単体テストでは
        // Log をno-op化する（モックされていないandroid APIで落とさない）
        unitTests.isReturnDefaultValues = true
    }

    defaultConfig {
        // Must match the watch_app applicationId for Wearable MessageClient to work
        applicationId = "com.mt3hr.gkill.wear"
        minSdk = 26
        targetSdk = 36
        versionCode = (findProperty("versionCode") as? String)?.toIntOrNull() ?: 1
        versionName = (findProperty("versionName") as? String) ?: "1.0.0"
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

    androidResources {
        // UI 文字列は res/values*/strings.xml の7言語（既定 ja）。依存ライブラリ（appcompat / material /
        // play-services）が持つ80言語超のリソースを落とし、アプリ未対応の言語の端末で
        // フレームワーク部品だけ別言語になる混在を防ぐ。"ja" は appcompat 側の values-ja を残すために要る。
        localeFilters += listOf("ja", "en", "zh", "ko", "es", "fr", "de")
        // Android 13+ のアプリ別言語設定。res/resources.properties の unqualifiedResLocale=ja と対で、
        // manifest の android:localeConfig を自動生成する（手書きの locales_config.xml と併用不可）。
        generateLocaleConfig = true
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
    // AGP 9 で wearApp 設定 (Wear 1.x 時代の埋め込み配布) は削除された。
    // watch_app は :watch_app:assembleDebug で個別にビルドし、個別に adb install する
    implementation(libs.play.services.wearable)
    implementation(libs.okhttp)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.kotlinx.coroutines.android)
    implementation(libs.kotlinx.coroutines.play.services)
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.appcompat)
    implementation(libs.androidx.work.runtime.ktx)
    implementation(libs.material)
    testImplementation(libs.junit)
    testImplementation(libs.mockk)
    testImplementation(libs.kotlinx.coroutines.test)
    testImplementation(libs.okhttp.mockwebserver)
    testImplementation(libs.okhttp.tls)
}
