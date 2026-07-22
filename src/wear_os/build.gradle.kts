// AGP 9 は Kotlin サポートを内蔵し、KGP 2.2.10 を同梱している。
// 最新の Kotlin を使うため classpath で KGP のバージョンを引き上げる。
buildscript {
    repositories {
        google()
        mavenCentral()
    }
    dependencies {
        classpath("org.jetbrains.kotlin:kotlin-gradle-plugin:${libs.versions.kotlin.get()}")
    }
}

plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.compose) apply false
    alias(libs.plugins.kotlinx.serialization) apply false
}
