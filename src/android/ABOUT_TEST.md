# android テスト仕様

## 概要

Android APK ラッパーのテスト。JVM 上で動作するユニットテストと、Android デバイス/エミュレータが必要なインストルメンテーションテストの2種類がある。

## テストフレームワーク

JUnit 4 + Kotlin

## テスト統計

合計24テスト（2ファイル）

## テストファイル一覧

| ファイル | テスト種別 | テスト数 | テスト内容 |
|---------|-----------|---------|-----------|
| `app/src/test/java/.../MainActivityUnitTest.kt` | ユニットテスト（JVM） | 19 | データ置き場 `/sdcard/gkill`、アプリ専用領域からの複製（置き場が無い・空なら複製して複製元を残す／中身があれば何もしない／複製元が無ければ何もしない／中断後の一時ディレクトリから完了する／同名ファイルは消さずに例外）、起動引数（`--address` と `--disable_tls` を含まないこと・ホームとログのフラグ）、サーバの起動行からの URL 取り出し（http / https・起動行でない行や壊れた URL は捨てる）、URL のポート（明示・スキーム既定）、同じオリジンの判定、ループバックのホスト判定、バイナリ名、ソケットタイムアウト、リトライ間隔、PID抽出正規表現、プロセス行フィルタの検証 |
| `app/src/androidTest/java/.../MainActivityInstrumentedTest.kt` | インストルメンテーションテスト | 5 | Android コンテキスト検証（パッケージ名、appContext非null、filesDir、assets、cacheDirの存在確認） |

## テスト内容

- **ユニットテスト**: データ置き場の定数（`/sdcard/gkill`）、`copyAppPrivateHomeIfNeeded` によるアプリ専用領域からの複製（一時ディレクトリ経由で改名するので、途中で止まっても中途半端な置き場が正として使われない）、`buildGkillServerArgs` が組み立てる起動引数（待受アドレスと TLS を上書きしない）、画面のアドレスを決める `parseServerUrlLine` / `serverPortOf` / `isSameServerOrigin` / `isLoopbackHost`、gkill_server バイナリ名、ソケットタイムアウト値、リトライ間隔、PID抽出正規表現、プロセス行フィルタなど純粋ロジックの検証
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
