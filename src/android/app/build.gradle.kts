import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    // AGP 9 は Kotlin サポートを内蔵しているため kotlin-android プラグインは適用しない
    alias(libs.plugins.android.application)
}

android {
    namespace = "com.gkill_android.mobile_app.src.gkill.mt3hr.gkill"
    // androidx 1.19.0 系が compileSdk 37 以上を要求する。
    // targetSdk は実行時挙動が変わるため 36 のまま据え置く。
    compileSdk = 37

    defaultConfig {
        applicationId = "com.mt3hr.gkill"
        minSdk = 26
        targetSdk = 36
        versionCode = (findProperty("versionCode") as? String)?.toIntOrNull() ?: 1
        versionName = (findProperty("versionName") as? String) ?: "1.0.0"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    // リリース署名。鍵はリポジトリ外で管理し、~/.gradle/gradle.properties または環境変数で渡す:
    //   GKILL_RELEASE_KEYSTORE (keystoreパス) / GKILL_RELEASE_KEYSTORE_PASSWORD /
    //   GKILL_RELEASE_KEY_ALIAS / GKILL_RELEASE_KEY_PASSWORD
    // 未設定の assembleRelease は未署名 APK (app-release-unsigned.apk) になり、配布名への
    // rename が見つからず release パイプラインが止まる (黙って debug 鍵の配布物を作らない。
    // 2026-08-30 監査 F-006)。
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
    packaging {
        jniLibs {
            // gkill_server を libgkill_server.so として同梱し、nativeLibraryDir から実行する。
            // 圧縮同梱にしないと APK 内に据え置かれ実体ファイルが作られず、exec できない
            useLegacyPackaging = true
        }
    }
}

dependencies {

    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.appcompat)
    implementation(libs.material)
    implementation(libs.androidx.activity)
    implementation(libs.androidx.constraintlayout)
    testImplementation(libs.junit)
    testImplementation(libs.mockk)
    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(libs.androidx.espresso.core)
}