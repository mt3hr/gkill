# wear_os テスト仕様

## 概要

Wear OS (Pixel Watch) KFTL 入力アプリのテスト。スマホ側コンパニオンアプリ（4テスト）とウォッチ側アプリ（5テスト）の合計9テストで構成される。

## テストフレームワーク

JUnit 4 + MockK（Kotlin モッキングライブラリ）

## テストファイル一覧

### phone_companion（スマホ側コンパニオン）

| ファイル | テスト内容 |
|---------|-----------|
| `phone_companion/src/test/java/.../GkillCredentialStoreTest.kt` | SharedPreferences を使用した認証情報ストアの保存・取得・削除（MockK使用） |
| `phone_companion/src/test/java/.../MainActivityTest.kt` | コンパニオンアプリの Activity ライフサイクル |
| `phone_companion/src/test/java/.../GkillApiClientTest.kt` | HTTP API クライアント（MockWebServer 使用、ログイン・KFTL送信・テンプレート取得） |
| `phone_companion/src/test/java/.../GkillWearableListenerServiceTest.kt` | ウォッチ→スマホ間メッセージパスのハンドリング |

### watch_app（ウォッチ側アプリ）

| ファイル | テスト内容 |
|---------|-----------|
| `watch_app/src/test/java/.../MainActivityTest.kt` | ウォッチアプリの Activity テスト |
| `watch_app/src/test/java/.../TemplateCacheManagerTest.kt` | ウォッチ上のテンプレートキャッシュ管理 |
| `watch_app/src/test/java/.../GkillWearClientTest.kt` | Wearable Data Layer クライアント |
| `watch_app/src/test/java/.../data/model/TemplateNodeTest.kt` | テンプレートツリー構造のデータモデル |
| `watch_app/src/test/java/.../data/model/PlaingTimeIsNodeTest.kt` | Plaing（計画）UIノードモデル |

## テスト内容

- **認証情報管理**: SharedPreferences を通じた gkill サーバ接続情報の保存
- **API 通信**: MockWebServer によるログイン、KFTL テキスト送信、テンプレート取得のテスト
- **Watch-Phone 連携**: Wearable Data Layer メッセージパス（`/gkill/submit`, `/gkill/templates` 等）の送受信
- **データモデル**: テンプレートノードと PlaingTimeIs ノードの構造検証

## 実行方法

```bash
npm run test_wear_os
```

手動実行:
```bash
cd src/wear_os && ./gradlew test
```

> **注意**: `gradlew` / `gradlew.bat` / `gradle-wrapper.jar` は `src/android/` からコピーが必要。
