// Top-level build file where you can add configuration options common to all sub-projects/modules.

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
}
