# wear_os - Wear OS アプリ

## 概要

Wear OS（Pixel Watch 等）用の KFTL 記録アプリ。Gradle マルチモジュールプロジェクトとして構成され、
スマートフォン側のコンパニオンサービスとウォッチ側のアプリで協調動作する。
ウォッチからテンプレートベースの KFTL テキストを送信し、gkill_server 経由でデータを記録する。

## ディレクトリ構造

```
wear_os/
├── build.gradle.kts           # ルート Gradle 設定
├── settings.gradle.kts        # マルチモジュール設定
├── gradle.properties          # Gradle プロパティ
├── gradlew / gradlew.bat      # Gradle ラッパー（android/ からコピー）
├── gradle/
│   └── wrapper/               # Gradle Wrapper JAR（android/ からコピー）
├── phone_companion/           # スマホ側コンパニオンモジュール
│   ├── build.gradle.kts
│   └── src/main/
│       ├── AndroidManifest.xml
│       ├── java/.../wear/companion/    # Kotlin ソース（4ファイル）
│       └── res/
│           ├── raw/wearable_app.apk    # ウォッチアプリ APK
│           └── values/strings.xml
└── watch_app/                 # ウォッチ側アプリモジュール
    ├── build.gradle.kts
    └── src/main/
        ├── AndroidManifest.xml
        ├── java/.../wear/watch/        # Kotlin ソース（15ファイル）
        └── res/
            └── values/strings.xml
```

## モジュール構成

### `phone_companion/` — スマホ側コンパニオンサービス（4ファイル）

ウォッチからのリクエストを受け取り、gkill_server API を呼び出すサービス。

| ファイル | 役割 |
|---------|------|
| `GkillWearableListenerService.kt` | Wearable Data Layer のメッセージリスナー。ウォッチからのテンプレート要求・KFTL 送信を処理 |
| `GkillApiClient.kt` | gkill_server の HTTP API を呼び出すクライアント（login, get_application_config, submit_kftl_text） |
| `GkillCredentialStore.kt` | ユーザ認証情報（user_id, password）の安全な保存・読み取り |
| `MainActivity.kt` | コンパニオンアプリのメインアクティビティ（認証情報設定画面） |

### `watch_app/` — ウォッチ側アプリ（15ファイル）

Compose for Wear OS で構築されたウォッチアプリ。KFTL テンプレートの選択・送信を行う。

#### エントリポイント

| ファイル | 役割 |
|---------|------|
| `MainActivity.kt` | ウォッチアプリのエントリポイント。Compose UI のセットアップ |

#### `data/` — データ層（3ファイル）

| ファイル | 役割 |
|---------|------|
| `GkillWearClient.kt` | Wearable Data Layer 通信。スマホ側へのメッセージ送受信 |
| `model/PlaingTimeIsNode.kt` | 稼働中 TimeIs のデータモデル |
| `model/TemplateNode.kt` | KFTL テンプレートのデータモデル |

#### `presentation/` — UI 層（7ファイル）

Compose for Wear OS による画面構成。

| ファイル | 画面 | 説明 |
|---------|------|------|
| `screens/TemplateListScreen.kt` | テンプレート一覧 | KFTL テンプレートの選択画面 |
| `screens/ConfirmScreen.kt` | 送信確認 | 選択したテンプレートの送信確認 |
| `screens/LoadingScreen.kt` | ローディング | 通信中の待機画面 |
| `screens/ResultScreen.kt` | 結果表示 | 送信結果（成功/失敗）の表示 |
| `screens/PlaingTimeIsListScreen.kt` | 稼働中 TimeIs | 稼働中タイマーの一覧・終了操作 |
| `screens/PlaingEndConfirmScreen.kt` | TimeIs 終了確認 | タイマー終了の確認画面 |
| `theme/Theme.kt` | テーマ | Compose テーマ定義 |

#### `tile/` — Wear OS タイル（3ファイル）

ウォッチフェイスから直接アクセスできるタイル。

| ファイル | 役割 |
|---------|------|
| `GkillTileService.kt` | タイルサービス。テンプレート一覧をタイルとして表示 |
| `TemplateCacheManager.kt` | テンプレートのローカルキャッシュ管理 |
| `TileRefreshActivity.kt` | タイルの手動リフレッシュ |

## メッセージパス（Watch ↔ Phone）

Wearable Data Layer API を使用した通信:

| パス | 方向 | 説明 |
|-----|------|------|
| `/gkill/get_templates` | Watch → Phone | テンプレート一覧を要求 |
| `/gkill/templates` | Phone → Watch | テンプレート一覧を返却（JSON 配列） |
| `/gkill/submit` | Watch → Phone | KFTL テキストを送信 |
| `/gkill/submit_result` | Phone → Watch | 送信結果を返却（"OK" or "ERROR:message"） |

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
# gradlew を android/ からコピー（初回のみ）
cp src/android/gradlew src/wear_os/
cp src/android/gradlew.bat src/wear_os/
cp -r src/android/gradle/wrapper src/wear_os/gradle/

cd src/wear_os

# ウォッチアプリビルド
./gradlew :watch_app:assembleDebug

# コンパニオンアプリビルド
./gradlew :phone_companion:assembleDebug
```

**注意:**
- `gradlew`, `gradlew.bat`, `gradle/wrapper/gradle-wrapper.jar` は `src/android/` からコピーが必要
- 両モジュールの applicationId は `com.gkill_android.mobile_app.src.gkill.mt3hr.gkill`（android/ と統一）

## 開発ガイドライン

### パッケージ名の統一

Android アプリ・Phone Companion・Watch App はすべて同一の applicationId を使用。
Wearable Data Layer 通信にはパッケージ名の一致が必要なため、変更時は3箇所同時に更新すること。

### テンプレート形式

テンプレートは `TemplateNode` として JSON でやり取り:
- gkill_server の `ApplicationConfig.kftl_template_struct` から取得
- Watch App でローカルキャッシュ（`TemplateCacheManager`）
