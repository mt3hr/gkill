# android テスト仕様

## 概要

Android APK ラッパーのテスト。JVM 上で動作するユニットテストと、Android デバイス/エミュレータが必要なインストルメンテーションテストの2種類がある。

## テストフレームワーク

JUnit 4 + Kotlin

## テスト統計

合計12テスト（2ファイル）

## テストファイル一覧

| ファイル | テスト種別 | テスト数 | テスト内容 |
|---------|-----------|---------|-----------|
| `app/src/test/java/.../MainActivityUnitTest.kt` | ユニットテスト（JVM） | 7 | サーバURL、ポート9999、バイナリ名、ソケットタイムアウト、リトライ間隔、PID抽出正規表現、プロセス行フィルタの検証 |
| `app/src/androidTest/java/.../MainActivityInstrumentedTest.kt` | インストルメンテーションテスト | 5 | Android コンテキスト検証（パッケージ名、appContext非null、filesDir、assets、cacheDirの存在確認） |

## テスト内容

- **ユニットテスト**: サーバ接続定数（`http://localhost:9999`）、gkill_server バイナリ名、ソケットタイムアウト値、リトライ間隔、PID抽出正規表現、プロセス行フィルタなど純粋ロジックの検証
- **インストルメンテーションテスト**: Android デバイス上でのコンテキスト検証（パッケージ名の正確性、appContext の存在、filesDir・assets・cacheDir の利用可能性確認）

## 実行方法

```bash
npm run test_android
```

手動実行:
```bash
cd src/android && ./gradlew test          # ユニットテスト
cd src/android && ./gradlew connectedTest  # インストルメンテーションテスト
```
