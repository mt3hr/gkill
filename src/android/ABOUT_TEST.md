# android テスト仕様

## 概要

Android APK ラッパーのテスト。JVM 上で動作するユニットテストと、Android デバイス/エミュレータが必要なインストルメンテーションテストの2種類がある。

## テストフレームワーク

JUnit 4 + Kotlin

## テストファイル一覧

| ファイル | テスト種別 | テスト内容 |
|---------|-----------|-----------|
| `app/src/test/java/.../MainActivityUnitTest.kt` | ユニットテスト（JVM） | 定数値（サーバURL、ポート9999、バイナリ名、ソケットタイムアウト）の検証 |
| `app/src/androidTest/java/.../MainActivityInstrumentedTest.kt` | インストルメンテーションテスト | Android フレームワーク統合テスト |

## テスト内容

- **ユニットテスト**: サーバ接続定数（`http://localhost:9999`）、gkill_server バイナリ名、ソケットタイムアウト値など純粋ロジックの検証
- **インストルメンテーションテスト**: Android デバイス上での WebView 起動、サーババイナリのコピー・起動確認

## 実行方法

```bash
npm run test_android
```

手動実行:
```bash
cd src/android && ./gradlew test          # ユニットテスト
cd src/android && ./gradlew connectedTest  # インストルメンテーションテスト
```
