# wear_os テスト仕様

## 概要

Wear OS (Pixel Watch) 記録アプリのテスト。スマホ側コンパニオンアプリ（10ファイル、142テスト）とウォッチ側アプリ（8ファイル、88テスト）の合計230テスト（18ファイル）で構成される。

## テストフレームワーク

JUnit 4 + MockK（Kotlin モッキングライブラリ）

## テストファイル一覧

### phone_companion（スマホ側コンパニオン）— 142テスト

| ファイル | テスト数 | テスト内容 |
|---------|---------|-----------|
| `phone_companion/src/test/java/.../GkillCredentialStoreTest.kt` | 24 | 認証情報ストアの保存・取得・削除。`GkillSecretCipher`（Android Keystore による暗号化）経由の保存と、ホスト別のピン留め証明書フィンガープリント保存も含む（MockK使用） |
| `phone_companion/src/test/java/.../MainActivityTest.kt` | 8 | コンパニオンアプリの Activity ライフサイクル |
| `phone_companion/src/test/java/.../GkillApiClientTest.kt` | 29 | HTTP API クライアント（MockWebServer 使用、ログイン・KFTL送信・テンプレート取得・playing検索クエリの形状検証。okhttp-tls の自己署名証明書での TLS ピン留め一致/不一致/未保存/既定モード/SAN欠落フォールバック検証を含む。`locale_name` が login を含む6種類の API 本文すべてに載ること、自前エラーが ASCII コードで返ること） |
| `phone_companion/src/test/java/.../GkillServerTrustTest.kt` | 13 | TOFU+ピン留めの TrustManager。フィンガープリント計算・整形・照合、ピン一致/不一致/未保存の可否、ホストキー導出（okhttp-tls の HeldCertificate 使用） |
| `phone_companion/src/test/java/.../GkillServerUrlPolicyTest.kt` | 4 | サーバーURLの受け入れ境界（平文HTTPはループバックのみ。LAN/公開ホスト・偽装ホスト名・不正形式の拒否、HTTPSの許可） |
| `phone_companion/src/test/java/.../GkillWearableListenerServiceTest.kt` | 19 | ウォッチ→スマホ間メッセージパスのハンドリング |
| `phone_companion/src/test/java/.../WearRequestHandlerTest.kt` | 17 | 時計要求ハンドラ（MockWebServer 使用、4ハンドラの成功/失敗/`ERROR:`プレフィックス契約と重複送信の `DUPLICATE`/force 上書き） |
| `phone_companion/src/test/java/.../WearSubmitLedgerTest.kt` | 9 | KFTL 送信の重複台帳（成功時のみ記録・TTL・既定の窓が30分であること・「それでも送信」で窓が最後の保存時刻から数え直されること・上限・永続化・破損時の空扱い） |
| `phone_companion/src/test/java/.../GkillErrorTextTest.kt` | 12 | エラーコード→文言の照合。`WIRE_ERROR_CODES` 全件に訳があること（コードを足して訳を忘れると落ちる）、未知文字列・`HTTP 500` の素通し、`ERROR:` 付き応答だけ訳して `OK`/`DUPLICATE`/JSON はバイト列のまま |
| `phone_companion/src/test/java/.../StringsParityTest.kt` | 7 | `res/values` と `values-{en,zh,ko,es,fr,de}` の `strings.xml` の整合（ロケール集合・キー集合の一致・`translatable="false"` の非重複・空値なし・`%1$s` プレースホルダの一致・未エスケープの `'` `"` と孤立 `%` の検出・`server_locale_name` がディレクトリ名と一致） |

### watch_app（ウォッチ側アプリ）— 88テスト

| ファイル | テスト数 | テスト内容 |
|---------|---------|-----------|
| `watch_app/src/test/java/.../MainActivityTest.kt` | 21 | ウォッチアプリの Activity テスト（画面状態の件数、タイルが渡す `EXTRA_MODE` の値） |
| `watch_app/src/test/java/.../data/LantanaKftlTest.kt` | 16 | 気分記録。星5個と気分値 1-10 の対応（Web 版の式との一致）、送信する KFTL テキストの完全一致、範囲外・未選択(0)の拒否 |
| `watch_app/src/test/java/.../TemplateCacheManagerTest.kt` | 9 | ウォッチ上のテンプレートキャッシュ管理と、スマホへ取りに行くかの判定（`shouldFetchFromPhone`） |
| `watch_app/src/test/java/.../tile/LocaleChangedReceiverTest.kt` | 2 | 端末の言語変更でタイルの再描画を要求すること（`TileService.getUpdater` を MockK で差し替え）と、`LOCALE_CHANGED` 以外のブロードキャストを無視すること。タイルのレイアウトは `onTileRequest` 時の文言で固定されるので、ここが無いと言語を切り替えてもタイルだけ古い言語のまま残る |
| `watch_app/src/test/java/.../GkillWearClientTest.kt` | 11 | Wearable Data Layer クライアント |
| `watch_app/src/test/java/.../data/model/TemplateNodeTest.kt` | 10 | テンプレートツリー構造のデータモデル |
| `watch_app/src/test/java/.../data/model/PlayingTimeIsNodeTest.kt` | 13 | Playing（計画）UIノードモデル |
| `watch_app/src/test/java/.../StringsParityTest.kt` | 6 | `res/values` と `values-{en,zh,ko,es,fr,de}` の `strings.xml` の整合（companion 側と同型。`server_locale_name` の検査だけ無い） |

## テスト内容

- **認証情報管理**: SharedPreferences を通じた gkill サーバ接続情報の保存と、ホスト別のピン留め証明書フィンガープリント保存
- **TLS ピン留め**: 自己署名サーバー向けの TOFU（Trust On First Use）+ ピン留め。プラットフォーム既定で検証し、失敗時のみ保存済みフィンガープリントと一致すれば許可（okhttp-tls）
- **API 通信**: MockWebServer によるログイン、KFTL テキスト送信、テンプレート取得のテスト
- **Watch-Phone 連携**: Wearable Data Layer メッセージパス（`/gkill/submit`, `/gkill/templates` 等）の送受信。受信は WorkManager ワーカーへ委譲し、処理本体は Android 非依存の `WearRequestHandler` に抽出
- **重複送信対策**: 直近成功した KFTL テキストの完全一致台帳。一致時は `DUPLICATE` を返して時計側に確認（それでも送信）を出させる
- **データモデル**: テンプレートノードと PlayingTimeIs ノードの構造検証
- **気分記録**: 星の塗り分けと気分値の対応、送信する KFTL テキスト（`?<日時>` / `/mood` / 値の3行）の完全一致。サーバ側の対の検査は `src/server/gkill/api/kftl/kftl_statement_test.go` の `TestStatement_LantanaFromWearOS`
- **多言語化**: 両モジュールの `strings.xml` 7言語セットの整合（Android のリソース解決はキーが欠けても例外を出さず既定言語へ静かに落ちるので、ここでしか守れない）。スマホ側のエラーコード→文言の照合と、サーバへ送る `locale_name` が UI と同じ言語になること

## 実行方法

```bash
npm run test_wear_os
```

手動実行:
```bash
cd src/wear_os && ./gradlew test
```

> **注意**: Gradle ラッパー（`gradlew` / `gradlew.bat` / `gradle-wrapper.jar`）はコミット済みでコピー不要。壊れた場合は `npm run setup_wear_os_gradle` で入れ直せる。
