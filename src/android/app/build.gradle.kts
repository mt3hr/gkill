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