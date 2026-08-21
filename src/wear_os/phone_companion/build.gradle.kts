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
        // getPlaingTimeis 等が android.util.Log を呼ぶため、JVM単体テストでは
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

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
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
