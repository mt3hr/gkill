# フォルダ構成説明

## 概要

本資料では、gkillプロジェクトのディレクトリ構成を説明します。開発者がソースコードの所在やビルド成果物の配置を把握するための参照資料です。

## プロジェクトルート

```
gkill/
├── src/                    # ソースコード（詳細は後述）
├── documents/              # ドキュメント類（詳細は後述）
├── dist/                   # フロントエンドビルド成果物（vite build出力先）
├── public/                 # 静的アセット（favicon.ico等）
├── release/                # リリースビルド成果物（クロスコンパイル時に生成）
├── resources/              # リソースファイル（サンプルデータ等）
├── node_modules/           # npm依存パッケージ（git管理外）
├── package.json            # npm設定・ビルドスクリプト定義
├── tsconfig*.json          # TypeScript設定
├── vite.config.ts          # Viteビルド設定
├── env.d.ts                # 環境変数型定義
├── CLAUDE.md               # Claude Code向けプロジェクトガイド
└── README.md               # プロジェクト説明（日本語）
```

## src/ — ソースコード

```
src/
├── client/       # フロントエンド（Vue 3 + TypeScript）
├── server/       # バックエンド（Go）
├── android/      # Android APKラッパー
├── wear_os/      # Wear OSアプリ（phone_companion + watch_app）
├── mcp/          # MCPサーバー（AI連携用）
├── locales/      # i18nリソース（7言語対応）
├── tools/        # ユーティリティスクリプト
└── README.md     # 実装資料への導線
```

### src/client/ — フロントエンド

Vue 3 + Vuetify 3 + TypeScript で構成されたPWAフロントエンドです。

```
src/client/
├── App.vue                 # ルートコンポーネント
├── main.ts                 # エントリーポイント（Vuetify, Router, i18n登録）
├── serviceWorker.ts        # PWAサービスワーカー（Workbox, Push通知, Web Share Target）
├── router/
│   └── index.ts            # ルーター定義（12ルート）
├── pages/                  # ルートページコンポーネント
│   ├── login-page.vue
│   ├── kftl-page.vue
│   ├── mi-page.vue
│   ├── rykv-page.vue
│   ├── kyou-page.vue
│   ├── mkfl-page.vue
│   ├── plaing-timeis-page.vue
│   ├── saihate-page.vue
│   ├── set-new-password-page.vue
│   ├── regist-first-account-page.vue
│   ├── shared-page.vue
│   ├── old-shared-mi-page.vue
│   ├── views/              # ビューコンポーネント（175ファイル）
│   └── dialogs/            # ダイアログコンポーネント（92ファイル）
├── classes/
│   ├── api/
│   │   └── gkill-api.ts    # GkillAPI シングルトン（~3,400行、全API呼び出しを集約）
│   ├── datas/              # TypeScriptデータモデル（Go構造体のミラー）
│   └── kftl/               # KFTLパーサー（44ステートメント型）
└── plugins/
    └── vuetify.ts          # Vuetify設定（ライト/ダークテーマ）
```

### src/server/ — バックエンド

Go言語で実装されたHTTPサーバーです。`go.mod`はこのディレクトリに配置されています。

```
src/server/
├── go.mod                          # Goモジュール定義（github.com/mt3hr/gkill/src/server）
├── go.sum                          # 依存パッケージのチェックサム
└── gkill/
    ├── api/                        # HTTPハンドラ層
    │   ├── gkill_server_api.go     # メインAPIハンドラ（~14,000行）
    │   ├── gkill_server_api_address.go  # ルーティング定義（79 POSTエンドポイント）
    │   ├── handle_*.go             # 個別ハンドラ関数群
    │   ├── embed.go                # go:embed によるSPA埋め込み
    │   ├── embed/                  # 埋め込みリソース（ビルド時生成）
    │   │   ├── html/              # フロントエンドビルド成果物
    │   │   ├── i18n/locales/      # i18nリソース
    │   │   └── version.json       # バージョン情報
    │   ├── find_filter.go          # 検索フィルタロジック
    │   ├── find/                   # 検索クエリ構造体
    │   ├── message/                # メッセージ/エラー構造体
    │   ├── req_res/                # リクエスト/レスポンス構造体（164ファイル）
    │   └── kftl/                   # KFTLパーサー（バックエンド側）
    │       ├── kftl_factory.go     # ファクトリ（ステートメント生成）
    │       └── *.go                # 各ステートメント型実装
    ├── dao/                        # データアクセス層
    │   ├── gkill_dao_manager.go    # DAOマネージャ（ConfigDAOs + GkillRepositories管理）
    │   ├── config_da_os.go          # ConfigDAOs構造体（8つの設定DAO）
    │   ├── reps/                   # リポジトリインターフェース・実装
    │   │   ├── *_repository.go             # インターフェース定義
    │   │   ├── *_repository_sqlite3_impl.go        # SQLite3実装
    │   │   ├── *_repository_cached_sqlite3_impl.go # キャッシュ付き実装
    │   │   ├── *_repository_temp_sqlite3_impl.go   # テンポラリ実装
    │   │   └── rep_cache_updater/  # キャッシュ更新ロジック
    │   ├── account/                # アカウントDAO
    │   ├── account_state/          # ログインセッション・アップロード履歴DAO
    │   ├── server_config/          # サーバー設定DAO
    │   ├── user_config/            # ユーザー設定・リポジトリ定義DAO
    │   ├── share_kyou_info/        # 共有設定DAO
    │   ├── gkill_notification/     # 通知ターゲットDAO
    │   └── hide_files/             # ファイル隠蔽ユーティリティ
    ├── dvnf/                       # DVNF（Data Versioning and Naming Framework）
    │   ├── dvnf.go                 # DVNFコア（タイムスタンプベース命名）
    │   └── cmd/                    # DVNFコマンド（get/copy/move）
    └── main/                       # エントリーポイント
        ├── common/                 # 共通初期化・CLIコマンド
        │   ├── common.go           # 初期化フロー、サブコマンド定義
        │   ├── gkill_options/      # CLIフラグ・ディレクトリ構造定義
        │   └── gkill_log/          # ログ設定
        ├── gkill_server/
        │   └── main.go             # gkill_server エントリーポイント（ヘッドレスHTTP）
        └── gkill/
            └── main.go             # gkill エントリーポイント（デスクトップアプリ）
```

### src/android/ — Android APKラッパー

Androidプロジェクト（Gradle）。WebViewでgkill_serverを内包して起動します。

```
src/android/
├── app/
│   └── src/main/
│       ├── java/.../MainActivity.java   # WebView + gkill_server起動
│       ├── assets/                      # gkill_serverバイナリ配置先
│       └── AndroidManifest.xml
├── build.gradle
├── gradlew / gradlew.bat               # Gradleラッパー（Wear OSからコピーされる）
└── gradle/wrapper/
    └── gradle-wrapper.jar
```

- compileSdk=35, targetSdk=28, minSdk=26

### src/wear_os/ — Wear OSアプリ

Gradle マルチモジュールプロジェクト。Wearable Data Layerで通信します。

```
src/wear_os/
├── phone_companion/        # スマートフォン側コンパニオンアプリ
│   └── src/main/java/...  # gkill_serverへのKFTL送信中継
├── watch_app/              # 腕時計側アプリ
│   └── src/main/java/...  # KFTL入力UI
├── settings.gradle
└── build.gradle
```

**注意:** ビルド前に`src/android/`から`gradlew`, `gradlew.bat`, `gradle-wrapper.jar`をコピーする必要があります（`npm run setup_wear_os_gradle`）。

### src/mcp/ — MCPサーバー

AI連携用のMCP（Model Context Protocol）サーバーです。

```
src/mcp/
└── gkill-read-server.mjs   # 読み取り専用MCPサーバー（7ツール提供）
```

トランスポート: stdio（デフォルト）またはHTTP。

### src/locales/ — i18nリソース

```
src/locales/
├── ja.json    # 日本語（プライマリ）
├── en.json    # 英語
├── zh.json    # 中国語
├── ko.json    # 韓国語
├── es.json    # スペイン語
├── fr.json    # フランス語
└── de.json    # ドイツ語
```

~765キー/言語。フラットなキーバリューJSON形式。フロントエンド（import）とバックエンド（go:embed）で共用されます。

### src/tools/ — ユーティリティスクリプト

```
src/tools/
└── license_getter.ps1    # ライセンス情報収集スクリプト
```

## documents/ — ドキュメント

```
documents/
├── reverse/                          # リバースエンジニアリング設計資料集
│   ├── README.md                     # 資料集の目次・推奨読み順
│   ├── glossary.md                   # 用語集（71項目）
│   ├── design-philosophy.md          # 設計思想
│   ├── usecase.md                    # ユースケース一覧（74件）
│   ├── er-diagram.md                 # ER図（Mermaid）
│   ├── class-diagrams.md             # クラス図
│   ├── sequence-diagrams.md          # シーケンス図（21本: 正常系16 + 異常系5）
│   ├── activity-diagrams.md          # アクティビティ図
│   ├── state-machines.md             # ステートマシン図
│   ├── screen-transition.md          # 画面遷移図
│   ├── screen-specs.md               # 画面仕様（項目定義）
│   ├── frontend-architecture.md      # フロントエンド設計ガイド
│   ├── api-endpoints.md              # APIエンドポイント一覧（79件）
│   ├── error-handling-and-security.md # エラー処理・セキュリティ
│   ├── operations-guide.md           # 運用ガイド
│   ├── dvnf-rep-type-spec.md         # DVNF/RepType仕様
│   ├── program-spec.md               # 主要プログラム仕様
│   ├── folder-structure.md           # フォルダ構成説明（本資料）
│   ├── dev-setup.md                  # 環境構築資料
│   ├── testing-guide.md              # テストガイド（実行・構成・トラブルシューティング）
│   └── user-guide.md                 # ユーザ向け導入資料
├── resources/                        # 画像リソース等
├── gkill_user_document.pdf           # ユーザ利用説明書
└── gkill_develop_document.xlsx       # 開発者向け起点要件書・設計書
```

## ランタイムディレクトリ

gkill実行時に`$HOME/gkill/`配下に以下の構造が作成されます。詳細は[operations-guide.md](operations-guide.md)を参照してください。

```
$HOME/gkill/
├── configs/                # 設定データベース群（SQLite3）
│   ├── server_config.db    # サーバー設定（アドレス、TLS、デバイス名）
│   ├── account.db          # アカウント情報
│   ├── account_state.db    # ログインセッション、アップロード履歴
│   ├── user_config.db      # アプリ設定、リポジトリ定義
│   ├── share_kyou_info.db  # 共有設定
│   └── gkill_notification_target.db  # プッシュ通知ターゲット
├── datas/                  # ユーザーデータ（デフォルトデータディレクトリ）
├── caches/                 # キャッシュファイル
├── logs/                   # ログファイル（JSON形式、レベル別分割）
│   ├── gkill_error.log
│   ├── gkill_warn.log
│   ├── gkill_info.log
│   ├── gkill_debug.log
│   ├── gkill_trace.log
│   ├── gkill_trace_sql.log
│   └── gkill.log           # 統合ログ
├── lib/base_directory/     # ライブラリファイル
└── tls/                    # TLS証明書（オプション）
    ├── cert.cer
    └── key.pem
```

## ビルド成果物

`npm run release` 実行時に `release/` 配下に以下が生成されます。

| ディレクトリ/ファイル | 内容 |
|---|---|
| `release/windows_amd64/` | Windows用バイナリ（gkill.exe, gkill_server.exe） |
| `release/linux_amd64/` | Linux x86_64用バイナリ |
| `release/linux_arm64/` | Linux ARM64用バイナリ |
| `release/linux_arm/` | Linux ARM用バイナリ |
| `release/android_arm/` | Android ARM用バイナリ |
| `release/android_arm64/` | Android ARM64用バイナリ |
| `release/android_apk/` | Android APKファイル |
| `release/wear_os/` | Wear OS APK（companion + watch） |
| `release/gkill_sample_data/` | サンプルデータ一式 |
