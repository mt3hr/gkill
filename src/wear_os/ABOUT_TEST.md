# wear_os テスト仕様

## 概要

Wear OS (Pixel Watch) KFTL 入力アプリのテスト。スマホ側コンパニオンアプリ（4ファイル、62テスト）とウォッチ側アプリ（5ファイル、61テスト）の合計123テスト（9ファイル）で構成される。

## テストフレームワーク

JUnit 4 + MockK（Kotlin モッキングライブラリ）

## テストファイル一覧

### phone_companion（スマホ側コンパニオン）— 62テスト

| ファイル | テスト数 | テスト内容 |
|---------|---------|-----------|
| `phone_companion/src/test/java/.../GkillCredentialStoreTest.kt` | 18 | 認証情報ストアの保存・取得・削除。`GkillSecretCipher`（Android Keystore による暗号化）経由の保存も含む（MockK使用） |
| `phone_companion/src/test/java/.../MainActivityTest.kt` | 8 | コンパニオンアプリの Activity ライフサイクル |
| `phone_companion/src/test/java/.../GkillApiClientTest.kt` | 17 | HTTP API クライアント（MockWebServer 使用、ログイン・KFTL送信・テンプレート取得・plaing検索クエリの形状検証） |
| `phone_companion/src/test/java/.../GkillWearableListenerServiceTest.kt` | 19 | ウォッチ→スマホ間メッセージパスのハンドリング |

### watch_app（ウォッチ側アプリ）— 61テスト

| ファイル | テスト数 | テスト内容 |
|---------|---------|-----------|
| `watch_app/src/test/java/.../MainActivityTest.kt` | 18 | ウォッチアプリの Activity テスト |
| `watch_app/src/test/java/.../TemplateCacheManagerTest.kt` | 9 | ウォッチ上のテンプレートキャッシュ管理と、スマホへ取りに行くかの判定（`shouldFetchFromPhone`） |
| `watch_app/src/test/java/.../GkillWearClientTest.kt` | 11 | Wearable Data Layer クライアント |
| `watch_app/src/test/java/.../data/model/TemplateNodeTest.kt` | 10 | テンプレートツリー構造のデータモデル |
| `watch_app/src/test/java/.../data/model/PlaingTimeIsNodeTest.kt` | 13 | Plaing（計画）UIノードモデル |

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

> **注意**: Gradle ラッパー（`gradlew` / `gradlew.bat` / `gradle-wrapper.jar`）はコミット済みでコピー不要。壊れた場合は `npm run setup_wear_os_gradle` で入れ直せる。
