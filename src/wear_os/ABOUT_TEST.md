# wear_os テスト仕様

## 概要

Wear OS (Pixel Watch) KFTL 入力アプリのテスト。スマホ側コンパニオンアプリ（7ファイル、110テスト）とウォッチ側アプリ（5ファイル、61テスト）の合計171テスト（12ファイル）で構成される。

## テストフレームワーク

JUnit 4 + MockK（Kotlin モッキングライブラリ）

## テストファイル一覧

### phone_companion（スマホ側コンパニオン）— 110テスト

| ファイル | テスト数 | テスト内容 |
|---------|---------|-----------|
| `phone_companion/src/test/java/.../GkillCredentialStoreTest.kt` | 24 | 認証情報ストアの保存・取得・削除。`GkillSecretCipher`（Android Keystore による暗号化）経由の保存と、ホスト別のピン留め証明書フィンガープリント保存も含む（MockK使用） |
| `phone_companion/src/test/java/.../MainActivityTest.kt` | 8 | コンパニオンアプリの Activity ライフサイクル |
| `phone_companion/src/test/java/.../GkillApiClientTest.kt` | 22 | HTTP API クライアント（MockWebServer 使用、ログイン・KFTL送信・テンプレート取得・plaing検索クエリの形状検証。okhttp-tls の自己署名証明書での TLS ピン留め一致/不一致/未保存/既定モード/SAN欠落フォールバック検証を含む） |
| `phone_companion/src/test/java/.../GkillServerTrustTest.kt` | 13 | TOFU+ピン留めの TrustManager。フィンガープリント計算・整形・照合、ピン一致/不一致/未保存の可否、ホストキー導出（okhttp-tls の HeldCertificate 使用） |
| `phone_companion/src/test/java/.../GkillWearableListenerServiceTest.kt` | 19 | ウォッチ→スマホ間メッセージパスのハンドリング |
| `phone_companion/src/test/java/.../WearRequestHandlerTest.kt` | 15 | 時計要求ハンドラ（MockWebServer 使用、4ハンドラの成功/失敗/`ERROR:`プレフィックス契約と重複送信の `DUPLICATE`/force 上書き） |
| `phone_companion/src/test/java/.../WearSubmitLedgerTest.kt` | 7 | KFTL 送信の重複台帳（成功時のみ記録・TTL・上限・永続化・破損時の空扱い） |

### watch_app（ウォッチ側アプリ）— 61テスト

| ファイル | テスト数 | テスト内容 |
|---------|---------|-----------|
| `watch_app/src/test/java/.../MainActivityTest.kt` | 18 | ウォッチアプリの Activity テスト |
| `watch_app/src/test/java/.../TemplateCacheManagerTest.kt` | 9 | ウォッチ上のテンプレートキャッシュ管理と、スマホへ取りに行くかの判定（`shouldFetchFromPhone`） |
| `watch_app/src/test/java/.../GkillWearClientTest.kt` | 11 | Wearable Data Layer クライアント |
| `watch_app/src/test/java/.../data/model/TemplateNodeTest.kt` | 10 | テンプレートツリー構造のデータモデル |
| `watch_app/src/test/java/.../data/model/PlaingTimeIsNodeTest.kt` | 13 | Plaing（計画）UIノードモデル |

## テスト内容

- **認証情報管理**: SharedPreferences を通じた gkill サーバ接続情報の保存と、ホスト別のピン留め証明書フィンガープリント保存
- **TLS ピン留め**: 自己署名サーバー向けの TOFU（Trust On First Use）+ ピン留め。プラットフォーム既定で検証し、失敗時のみ保存済みフィンガープリントと一致すれば許可（okhttp-tls）
- **API 通信**: MockWebServer によるログイン、KFTL テキスト送信、テンプレート取得のテスト
- **Watch-Phone 連携**: Wearable Data Layer メッセージパス（`/gkill/submit`, `/gkill/templates` 等）の送受信。受信は WorkManager ワーカーへ委譲し、処理本体は Android 非依存の `WearRequestHandler` に抽出
- **重複送信対策**: 直近成功した KFTL テキストの完全一致台帳。一致時は `DUPLICATE` を返して時計側に確認（それでも送信）を出させる
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
