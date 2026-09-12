# wear_os - Wear OS アプリ

## 概要

Wear OS（Pixel Watch 等）用の gkill 記録アプリ（表示名 `gkill wear`）。Gradle マルチモジュール
プロジェクトとして構成され、スマートフォン側のコンパニオンサービスとウォッチ側のアプリで協調動作する。
ウォッチからテンプレートベースの KFTL テキストや気分値を送信し、gkill_server 経由でデータを記録する。
UI の文言は gkill 本体と同じ7言語（ja / en / zh / ko / es / fr / de）に対応する（後述「多言語対応」）。

## ディレクトリ構造

```
wear_os/
├── build.gradle.kts           # ルート Gradle 設定
├── settings.gradle.kts        # マルチモジュール設定
├── gradle.properties          # Gradle プロパティ
├── gradlew / gradlew.bat      # Gradle ラッパー（コミット済み）
├── gradle/
│   └── wrapper/               # Gradle Wrapper JAR（コミット済み）
├── phone_companion/           # スマホ側コンパニオンモジュール
│   ├── build.gradle.kts
│   └── src/main/
│       ├── AndroidManifest.xml
│       ├── java/.../wear/companion/    # Kotlin ソース（12ファイル）
│       └── res/
│           ├── resources.properties    # 既定言語 ja（generateLocaleConfig 用）
│           ├── values/strings.xml      # 既定（日本語）
│           └── values-{en,zh,ko,es,fr,de}/strings.xml
└── watch_app/                 # ウォッチ側アプリモジュール
    ├── build.gradle.kts
    └── src/main/
        ├── AndroidManifest.xml
        ├── java/.../wear/watch/        # Kotlin ソース（17ファイル）
        └── res/
            ├── values/strings.xml      # 既定（日本語）
            └── values-{en,zh,ko,es,fr,de}/strings.xml
```

## モジュール構成

### `phone_companion/` — スマホ側コンパニオンサービス（12ファイル）

ウォッチからのリクエストを受け取り、gkill_server API を呼び出すサービス。

| ファイル | 役割 |
|---------|------|
| `GkillWearableListenerService.kt` | Wearable Data Layer のメッセージリスナー。受け取ったリクエストは WorkManager へ委譲して Service 破棄を跨いで処理する |
| `WearRequestWorker.kt` | ウォッチからのリクエストを実際に処理する CoroutineWorker（資格情報ロード→API 呼び出し→結果送信。Service 破棄・プロセス死を跨ぐ） |
| `WearRequestHandler.kt` | テンプレート取得・KFTL 送信・実行中取得・終了の4処理の本体（Service から抽出し MockWebServer で単体テスト可能に）。時計へ返す `OK` / `DUPLICATE` / `ERROR:<code>` の語彙（`WIRE_*` 定数）もここに置く |
| `WearSubmitLedger.kt` | 直近成功した KFTL 送信の台帳。同一内容の再配送を重複登録せず確認へ回す（share-target-dedup 契約） |
| `GkillApiClient.kt` | gkill_server の HTTP API を呼び出すクライアント（login, get_application_config, submit_kftl_text, get_kyous, get_timeis, update_timeis）。全リクエストに `locale_name` を載せ、自前のエラーは ASCII コードで返す |
| `GkillServerUrlPolicy.kt` | サーバー URL の受け入れ判定（平文 HTTP はループバックのみ）。Android 非依存 |
| `GkillLocale.kt` | サーバーへ送る `locale_name` の決め方。`R.string.server_locale_name`（UI が解決したロケール）から引く |
| `GkillErrorText.kt` | エラーコード → 文言の照合。時計へ送る直前と設定画面の表示で使う唯一の翻訳箇所 |
| `GkillServerTrust.kt` | TLS 検証。プラットフォーム既定で検証し、失敗時のみ保存済み SHA-256 フィンガープリント一致（TOFU/ピン留め）で許可 |
| `GkillCredentialStore.kt` | ユーザ認証情報（user_id, password）と証明書ピンの安全な保存・読み取り |
| `GkillSecretCipher.kt` | 認証情報の暗号化・復号ユーティリティ |
| `MainActivity.kt` | コンパニオンアプリのメインアクティビティ（認証情報設定画面・証明書ピンの承認） |

### `watch_app/` — ウォッチ側アプリ（17ファイル）

Compose for Wear OS で構築されたウォッチアプリ。KFTL テンプレートの選択・送信と、
星5個による気分（Lantana）の記録を行う。

#### エントリポイント

| ファイル | 役割 |
|---------|------|
| `MainActivity.kt` | ウォッチアプリのエントリポイント。Compose UI のセットアップ。トップメニュー（タイルと同じ配置の3チップ）と重複確認画面（ScalingLazyColumn）もここにある |

#### `data/` — データ層（4ファイル）

| ファイル | 役割 |
|---------|------|
| `GkillWearClient.kt` | Wearable Data Layer 通信。スマホ側へのメッセージ送受信 |
| `LantanaKftl.kt` | 気分値と星5個の対応（Web 版と同じ 1-10 の刻み）、および送信する KFTL テキストの組み立て。Android API に触らないので JVM 単体テストできる |
| `model/PlayingTimeIsNode.kt` | 稼働中 TimeIs のデータモデル |
| `model/TemplateNode.kt` | KFTL テンプレートのデータモデル |

#### `presentation/` — UI 層（9ファイル）

Compose for Wear OS による画面構成。チップの見た目はタイル（`tile/GkillTileService.kt`）が基準で、
アプリ側は `components/MenuChip.kt` を使って揃える。

| ファイル | 画面 | 説明 |
|---------|------|------|
| `components/MenuChip.kt` | 共通部品 | タイルのチップと同じ見た目のチップ（140dp 固定幅・primary 色・文字中央揃え・末尾省略）。幅の定数 `MENU_CHIP_WIDTH_DP` をタイルと共有する |
| `screens/TemplateListScreen.kt` | テンプレート一覧 | KFTL テンプレートの選択画面 |
| `screens/ConfirmScreen.kt` | 送信確認 | 選択したテンプレートの送信確認 |
| `screens/LoadingScreen.kt` | ローディング | 通信中の待機画面 |
| `screens/ResultScreen.kt` | 結果表示 | 送信結果（成功/失敗）の表示 |
| `screens/PlayingTimeIsListScreen.kt` | 稼働中 TimeIs | 稼働中タイマーの一覧・終了操作 |
| `screens/PlayingEndConfirmScreen.kt` | TimeIs 終了確認 | タイマー終了の確認画面 |
| `screens/LantanaScreen.kt` | 気分記録 | 星5個の選択画面と送信確認画面。星の左半分/右半分で気分値 1-10 を選ぶ |
| `theme/Theme.kt` | テーマ | Compose テーマ定義 |

#### `tile/` — Wear OS タイル（3ファイル）

ウォッチフェイスから直接アクセスできるタイル。

| ファイル | 役割 |
|---------|------|
| `GkillTileService.kt` | タイルサービス。「📝 記録する」「▶ 実行中」「⭐️ 気分記録」の3つの導線を表示する（テンプレートは並べない）。チップの見た目はアプリ側の基準で、幅は `presentation/components/MenuChip.kt` の `MENU_CHIP_WIDTH_DP` を共有する |
| `TemplateCacheManager.kt` | テンプレートのローカルキャッシュ管理。読み書きするのは `MainActivity` のみ |
| `LocaleChangedReceiver.kt` | 端末の言語変更（`LOCALE_CHANGED`）でタイルの再描画を要求する。タイルのレイアウトは `onTileRequest` 時の文言で固定されるため |

## メッセージパス（Watch ↔ Phone）

Wearable Data Layer API を使用した通信:

| パス | 方向 | 説明 |
|-----|------|------|
| `/gkill/get_templates` | Watch → Phone | テンプレート一覧を要求 |
| `/gkill/templates` | Phone → Watch | テンプレート一覧を返却（JSON 配列） |
| `/gkill/submit` | Watch → Phone | KFTL テキストを送信 |
| `/gkill/submit_result` | Phone → Watch | 送信結果を返却（"OK" / "DUPLICATE" / "ERROR:message"） |
| `/gkill/get_playing_timeis` | Watch → Phone | 実行中の打刻一覧を要求 |
| `/gkill/playing_timeis` | Phone → Watch | 実行中の打刻一覧を返却（JSON 配列）または "ERROR:message" |
| `/gkill/end_timeis` | Watch → Phone | 打刻の終了を要求（"id\nrep_name"） |
| `/gkill/end_timeis_result` | Phone → Watch | 終了結果を返却（"OK" / "ERROR:message"） |

`ERROR:` の後ろは **スマホのロケールで訳された人間文**（サーバーの `error_message` か、スマホ側の
`GkillErrorText` が訳したもの）。ハンドラ内部（`WearRequestHandler` → `WearRequestWorker`）では
`login_failed` 等の ASCII コードで持ち、送信直前に訳す。時計側では訳さない。

## データフロー

```
[Watch App]
  ↓ /gkill/get_templates
[Phone Companion]
  ↓ POST /api/login → POST /api/get_application_config
[gkill_server]
  ↓ テンプレート一覧
[Phone Companion]
  ↓ /gkill/templates
[Watch App]
  → ユーザがテンプレート選択
  ↓ /gkill/submit (KFTL テキスト)
[Phone Companion]
  ↓ POST /api/submit_kftl_text
[gkill_server]
  ↓ データ保存
[Phone Companion]
  ↓ /gkill/submit_result ("OK")
[Watch App]
  → 結果表示
```

## ビルド方法

```bash
cd src/wear_os

# ウォッチアプリビルド
./gradlew :watch_app:assembleDebug

# コンパニオンアプリビルド
./gradlew :phone_companion:assembleDebug
```

**注意:**
- Gradle ラッパー（`gradlew`, `gradlew.bat`, `gradle/wrapper/gradle-wrapper.jar`）はコミット済みでコピー不要。壊れた場合は `npm run setup_wear_os_gradle` で `src/android/` から入れ直せる
- 両モジュールの applicationId は `com.mt3hr.gkill.wear`（Wearable Data Layer でスマホと時計を結びつけるため、2モジュールで一致していることが必須）

## 開発ガイドライン

### パッケージ名の統一

Phone Companion と Watch App は同一の applicationId（`com.mt3hr.gkill.wear`）を使用。
Wearable Data Layer 通信にはパッケージ名の一致が必要なため、変更時は2モジュール同時に更新すること。

### 多言語対応

- UI の文言は両モジュールの `res/values/strings.xml`（既定 = 日本語）と `values-{en,zh,ko,es,fr,de}/strings.xml` に置く。
  Kotlin に文言を直書きしない（Compose は `stringResource`、Activity / Service / Worker は `getString`）。
  既定が日本語なのは Web の `fallbackLocale: 'ja'`・サーバーの `GetLocalizer` フォールバックと揃えるため。
- キーを足したら7ファイル同時に足す。`StringsParityTest`（両モジュール）がキー集合・プレースホルダ・
  エスケープ（`\'` / `\"` / `&amp;` / `%%`）を検査する。aapt2 のエスケープ検査は `assemble` でしか走らないので、
  `test` だけ通して安心しないこと。
- サーバーへ送る `locale_name` は `GkillLocale.serverLocaleName(context)` = `R.string.server_locale_name`
  （各 `values-xx` に `xx`）。`Locale.getDefault()` から自前で判定しない（言語優先リストで UI と割れる）。
- 訳さないもの: KFTL テキスト（`?<日時>` / `/mood`）、ワイヤ上のコード `OK` / `DUPLICATE` / `ERROR:<code>`、
  機械識別子（`gkill_wear` / `gkill_wear_sync` / `gkill_wear_prefs`）、ブランド名 `gkill` / `gkill wear`（`translatable="false"`）。
- 依存ライブラリの80言語超のリソースは `androidResources.localeFilters` で7言語に絞る。companion は
  `generateLocaleConfig = true` + `res/resources.properties` で Android 13+ のアプリ別言語設定にも出る。
- 詳細と却下案は [ADR-1102](../../documents/adr/1102-wear-ui-strings-in-android-resources-with-ja-default.md)。

### テンプレート形式

テンプレートは `TemplateNode` として JSON でやり取り:
- gkill_server の `ApplicationConfig.kftl_template_struct` から取得
- Watch App でローカルキャッシュ（`TemplateCacheManager`）

キャッシュの更新はテンプレート一覧の一番下の「🔄 更新」から行う（`実行中` 画面と同じ位置・同じ見た目）。
これを押したときだけキャッシュを無視してスマホへ取りに行く（判定は `TemplateCacheManager.shouldFetchFromPhone`）。
サーバ側でテンプレートを直したら、ウォッチではこのボタンを押すまで反映されない。
