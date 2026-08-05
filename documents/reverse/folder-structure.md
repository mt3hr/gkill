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
├── resources/              # リソースファイル
│   ├── manual/             # 生成済みHTMLマニュアル（7言語×21ページ、go:embed対象）
│   ├── manual_src/         # マニュアルの原本（7言語 + _layout.html）。build_manuals.mjs の入力
│   └── gkill_sample_data/  # サンプルデータ
├── .github/                # GitHub Actions ワークフロー
├── node_modules/           # npm依存パッケージ（git管理外）
├── playwright-report/      # Playwright実行結果（git管理外）
├── test-results/           # テスト成果物（git管理外）
├── index.html              # Vite のエントリHTML
├── package.json            # npm設定・ビルドスクリプト定義
├── tsconfig*.json          # TypeScript設定
├── vite.config.ts          # Viteビルド設定
├── vitest.config.ts        # Vitest（クライアント）設定
├── vitest.config.mcp.ts    # Vitest（MCP）設定
├── playwright.config.ts    # Playwright E2E設定
├── eslint.config.js        # ESLint flat config
├── env.d.ts                # 環境変数型定義
├── LICENSE                 # MIT ライセンス
├── LICENSES_DEPENDENCE     # 依存パッケージのライセンス一覧
├── CLAUDE.md               # Claude Code向けプロジェクトガイド
└── README.md               # プロジェクト説明（日本語）
```

## src/ — ソースコード

```
src/
├── ABOUT_TEST.md # テスト仕様の目次（サブディレクトリの ABOUT_TEST.md へのリンク）
├── client/       # フロントエンド（Vue 3 + TypeScript）
├── server/       # バックエンド（Go）
├── android/      # Android APKラッパー
├── wear_os/      # Wear OSアプリ（phone_companion + watch_app）
├── mcp/          # MCPサーバー（AI連携用）
├── plugins/      # スタンドアロンプラグインバイナリ（examples/gkill_example, gkill_plugin_chatgpt, gkill_plugin_claudeai, gkill_plugin_claudecode）
├── locales/      # i18nリソース（7言語対応）
├── tools/        # ユーティリティスクリプト
└── README.md     # 実装資料への導線
```

### src/client/ — フロントエンド

Vue 3 + Vuetify 4 + TypeScript 6 で構成されたPWAフロントエンドです。

```
src/client/
├── App.vue                 # ルートコンポーネント
├── main.ts                 # エントリーポイント（Vuetify, Router, i18n登録）
├── serviceWorker.ts        # PWAサービスワーカー（Workbox, Push通知, Web Share Target）
├── router/
│   └── index.ts            # ルーター定義（13ルート）
├── pages/                  # ルートページコンポーネント
│   ├── login-page.vue
│   ├── kftl-page.vue
│   ├── mi-page.vue
│   ├── rykv-page.vue
│   ├── kyou-page.vue
│   ├── mkfl-page.vue
│   ├── plaing-time-is-page.vue
│   ├── dashboard-page.vue
│   ├── saihate-page.vue
│   ├── set-new-password-page.vue
│   ├── register-first-account-page.vue
│   ├── shared-page.vue
│   ├── old-shared-mi-page.vue
│   ├── shared-mi-page.vue
│   ├── shared-rykv-page.vue
│   ├── views/              # ビューコンポーネント（189ファイル）
│   └── dialogs/            # ダイアログコンポーネント（103ファイル、browse-zip-contents-dialog.vue 含む）
├── i18n.ts                 # i18n設定（ja のみ静的、他6言語は動的import）
├── assets/                 # 画像等の静的アセット
├── classes/
│   ├── api/
│   │   ├── gkill-api.ts    # GkillAPI シングルトン（~3,330行、全API呼び出しを集約）
│   │   ├── hydrate.ts      # JSON→クラスインスタンスの詰め替え（any を使わない共通ヘルパー）
│   │   ├── find_query/     # 検索クエリビルダー
│   │   └── req_res/        # リクエスト/レスポンス型（168ファイル）
│   ├── datas/              # TypeScriptデータモデル（Go構造体のミラー）
│   ├── dnote/              # Dnote集計ユーティリティ（dnote-trend-aggregator.ts, dnote-predicate/ 等）
│   ├── dto/                # データ転送オブジェクト
│   ├── lantana/            # 気分値関連クラス
│   ├── kftl/               # KFTLパーサー（41ステートメント型）
│   ├── component-ref.ts    # ComponentRef 型（any をここに封じ込める）
│   ├── kyou-content-text.ts # Kyou の内容/IDのクリップボードコピー
│   └── use-*.ts            # Composition関数群（231ファイル）
├── __tests__/              # テスト
│   ├── e2e/                # Playwright E2E（run-e2e.mjs, free-port.mjs, auth.setup.ts 等）
│   ├── helpers/            # テストヘルパー
│   └── unit/               # Vitest ユニットテスト（api / classes / composables / datas / dnote）
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
    ├── api/                        # API共通基盤
    │   ├── embed.go                # go:embed によるSPA埋め込み
    │   ├── embed/                  # 埋め込みリソース（ビルド時生成）
    │   │   ├── html/              # フロントエンドビルド成果物
    │   │   ├── i18n/locales/      # i18nリソース
    │   │   └── version.json       # バージョン情報
    │   ├── version.go              # バージョン情報取得
    │   ├── gkill_version_data.go   # バージョンデータ構造体
    │   ├── find_filter.go          # 検索フィルタロジック
    │   ├── find_filter_helpers.go  # 検索フィルタヘルパー
    │   ├── find_kyou_context.go    # 検索コンテキスト
    │   ├── find/                   # 検索クエリ構造体
    │   ├── message/                # メッセージ/エラー構造体
    │   ├── req_res/                # リクエスト/レスポンス構造体（182ファイル）
    │   ├── kftl/                   # KFTLパーサー（バックエンド側、39ステートメント型）
    │   │   ├── kftl_factory.go     # ファクトリ（ステートメント生成、日本語/ASCII両プレフィックス）
    │   │   └── *.go                # 各ステートメント型実装
    │   ├── gkill_plugin/           # プラグインプロトコル型
    │   │   ├── plugin_manifest.go  # PluginManifest（8フィールド）
    │   │   └── plugin_protocol.go  # PluginRequest / PluginResponse / PluginKyou
    │   └── gkill_server_api/       # HTTPハンドラ層（108ファイル）
    │       ├── serve.go            # HTTPサーバー起動・停止
    │       ├── close.go            # サーバー終了処理
    │       ├── gkill_server_api_address.go  # ルーティング定義（90エンドポイント定義・88登録：89 POST + 1 GET）
    │       ├── auth.go             # セッション認証ヘルパー
    │       ├── auth_context.go     # AuthContext構造体（認証済みコンテキスト）
    │       ├── auth_middleware.go  # authMiddleware / authWithReposMiddleware
    │       ├── filter_local_only.go # ローカルアクセス制限フィルタ
    │       ├── utils.go            # ユーティリティ関数
    │       ├── web_push.go         # WebPush通知
    │       ├── gkill_server_api_access_log.go  # アクセスログミドルウェア
    │       └── handle_*.go         # 個別ハンドラ（1ファイル1ハンドラ、94ファイル）
    ├── plugin/                     # プラグイン作者向けSDK
    │   └── sdk/                    # sdk.Run / sdk.Handler / sdk.EnsureConfig
    ├── dao/                        # データアクセス層
    │   ├── gkill_dao_manager.go    # DAOマネージャ（ConfigDAOs + GkillRepositories管理）
    │   ├── plugin_manager.go       # プラグインディレクトリを走査してPluginRepositoryを登録
    │   ├── config_da_os.go          # ConfigDAOs構造体（8つの設定DAO）
    │   ├── reps/                   # リポジトリインターフェース・実装
    │   │   ├── *_repository.go             # インターフェース定義
    │   │   ├── *_repository_sqlite3_impl.go        # SQLite3実装
    │   │   ├── *_repository_cached_sqlite3_impl.go # キャッシュ付き実装
    │   │   ├── *_repository_temp_sqlite3_impl.go   # テンポラリ実装
    │   │   ├── mi_re_kyou*.go              # MiReKyou（既存記録のタスク化、10ファイル）
    │   │   ├── plugin_repository.go        # プラグインリポジトリ インターフェース
    │   │   ├── plugin_repository_impl.go   # サブプロセス管理・stdio JSON通信
    │   │   ├── cache/              # LatestDataRepositoryAddress 等のキャッシュDAO
    │   │   └── rep_cache_updater/  # キャッシュ更新ロジック
    │   ├── account/                # アカウントDAO
    │   ├── account_state/          # ログインセッション・アップロード履歴DAO
    │   ├── server_config/          # サーバー設定DAO
    │   ├── user_config/            # ユーザー設定・リポジトリ定義DAO
    │   ├── share_kyou_info/        # 共有設定DAO
    │   ├── gkill_notification/     # 通知ターゲットDAO
    │   └── hide_files/             # ファイル隠蔽ユーティリティ
    ├── usecase/                    # ビジネスロジック層（17ファイル）
    │   └── *.go                    # HTTP非依存のユースケース関数群
    ├── dvnf/                       # DVNF（DeVice Name Folder Naming Framework）
    │   ├── dvnf.go                 # DVNFコア（タイムスタンプベース命名）
    │   └── cmd/                    # DVNFコマンド（get/copy/move）
    └── main/                       # エントリーポイント
        ├── common/                 # 共通初期化・CLIコマンド
        │   ├── common.go           # 初期化フロー、サブコマンド定義
        │   ├── password_admin.go   # reset_password サブコマンド、ローカル管理者セッションの発行
        │   ├── gkill_options/      # CLIフラグ・ディレクトリ構造定義
        │   ├── gkill_log/          # ログ設定
        │   └── threads/            # ゴルーチンプール（セマフォ）
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
│       ├── java/.../MainActivity.kt     # WebView + gkill_server起動
│       ├── jniLibs/arm64-v8a/          # gkill_serverバイナリ配置先
│       │   └── libgkill_server.so       #   （nativeLibraryDirから実行するため）
│       └── AndroidManifest.xml
├── build.gradle.kts
├── gradlew / gradlew.bat               # Gradleラッパー（Wear OSへコピーする元）
└── gradle/wrapper/
    └── gradle-wrapper.jar
```

- compileSdk=37, targetSdk=36, minSdk=26（compileSdk だけ 37 なのは androidx 1.19.0 系の要求。targetSdk は実行時挙動が変わるため 36 据え置き）

### src/wear_os/ — Wear OSアプリ

Gradle マルチモジュールプロジェクト。Wearable Data Layerで通信します。

```
src/wear_os/
├── phone_companion/        # スマートフォン側コンパニオンアプリ
│   └── src/main/java/...  # gkill_serverへのKFTL送信中継
├── watch_app/              # 腕時計側アプリ
│   └── src/main/java/...  # KFTL入力UI
├── settings.gradle.kts
└── build.gradle.kts
```

**補足:** Gradleラッパー（`gradlew` / `gradlew.bat` / `gradle-wrapper.jar` / `gradle-wrapper.properties`）は`src/wear_os/`にコミット済みなので、コピーは通常不要です。`src/android/`側と揃え直したいときだけ`npm run setup_wear_os_gradle`を実行してください。

### src/mcp/ — MCPサーバー

AI連携用のMCP（Model Context Protocol）サーバーです。

```
src/mcp/
├── gkill-read-server.mjs      # Read専用MCPサーバー（10ツール = 固有8 + プラグイン2、port 8808）
├── gkill-write-server.mjs     # Write専用MCPサーバー（25ツール = 固有23 + プラグイン2、port 8809）
├── gkill-readwrite-server.mjs # Read/Write統合MCPサーバー（30ツール = 固有28 + プラグイン2、port 8810）
└── lib/
    ├── access-log.mjs         # MCPアクセスログモジュール（MCP_LOG環境変数で制御）
    ├── plugin-tools.mjs       # 3サーバ共通のプラグインツール（gkill_get_plugin_list / gkill_get_plugin_content）
    ├── html-text.mjs          # プラグインコンテンツHTML→プレーンテキスト変換
    ├── file-link-store.mjs    # HTTPモード用ファイルリンクストア（期限付きトークンで /files/{token} 配信）
    ├── normalization.mjs      # Read入力正規化
    ├── write-normalization.mjs # Write入力正規化
    ├── validation.mjs         # 入力バリデーション
    ├── constants.mjs          # 共通定数
    ├── errors.mjs             # エラークラス
    ├── oauth-server.mjs       # OAuth 2.1サーバー
    ├── oauth-store.mjs        # トークン/コード永続化
    ├── oauth-html.mjs         # OAuth ログインページテンプレート
    └── pkce.mjs               # PKCE検証
```

トランスポート: stdio（デフォルト）またはHTTP（OAuth 2.1認証付き）。

### src/plugins/ — スタンドアロンプラグイン

gkill本体とは独立してビルドされるプラグインバイナリです。各プラグインは自身の `go.mod`・`manifest.json`・実行ファイルを持ち、stdio の改行区切りJSONで gkill サーバーと通信します（プロトコル型は `src/server/gkill/api/gkill_plugin/`、作者向けSDKは `src/server/gkill/plugin/sdk/`）。

```
src/plugins/
├── examples/
│   └── gkill_example/         # サンプルプラグイン（固定のKyouレスポンスを返す）
├── gkill_plugin_chatgpt/      # ChatGPT会話履歴プラグイン
├── gkill_plugin_claudeai/     # Claude.ai会話履歴プラグイン
├── gkill_plugin_claudecode/   # Claude Code チャットログプラグイン
├── ABOUT_TEST.md
└── README.md
```

各プラグインは独立した `go.mod` を持つ別モジュール。同梱プラグイン3本（chatgpt / claudeai / claudecode）は
`manifest.json` をバイナリに `//go:embed` しており、`--gkill-print-manifest` / `--gkill-print-config` で
内容を標準出力に書き出せる（`gkill_example` は埋め込みもフラグも `DefaultConfig` も持たない）。
この3本は `config.json` の `source_dirs` で取り込み元フォルダを指定し、
SQLite3 キャッシュを `$GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db` に置く
（各プラグインの `cache_path.go`）。

> `src/plugins/*` の Go テストは別モジュールのため、`npm run test_server`（`cd src/server && go test ./...`）
> では実行されない。`npm run test_plugins` が各モジュールを回し、`npm test` からも呼ばれる。
> 実行方法は `src/plugins/ABOUT_TEST.md` を参照。

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

856キー/言語。フラットなキーバリューJSON形式。フロントエンド（import）とバックエンド（go:embed）で共用されます。

### src/tools/ — ユーティリティスクリプト

```
src/tools/
├── dev.mjs                      # npm run dev のラッパー（--api / --api-target を GKILL_API_PROXY_TARGET に変換）
├── verify_docs.mjs              # docs CI。件数・リンク・参照パス・Mermaid・マニュアルを検証（npm run verify_docs）
├── build_manuals.mjs            # resources/manual_src/ → resources/manual/ を生成（npm run build_manuals）
├── manual_build.mjs             # マニュアル生成の実体（verify_docs からも import される）
├── manual_a11y.mjs              # マニュアルのアクセシビリティ検査
├── manual_ascii_fix.mjs         # マニュアルのASCII正規化
├── extract_manual_src.mjs       # 既存マニュアルから manual_src を抽出
├── verify_release_artifacts.mjs # リリース成果物の検証（npm run verify_release_artifacts）
├── license_getter.ps1           # ライセンス情報収集スクリプト
└── README.md
```

このうち `dev.mjs` / `verify_docs.mjs` / `build_manuals.mjs` / `verify_release_artifacts.mjs` は
`package.json` の npm スクリプトから直接呼ばれる。

## documents/ — ドキュメント

```
documents/
├── reverse/                          # リバースエンジニアリング設計資料集
│   ├── README.md                     # 資料集の目次・推奨読み順
│   ├── glossary.md                   # 用語集（90項目）
│   ├── design-philosophy.md          # 設計思想
│   ├── usecase.md                    # ユースケース一覧（84件）
│   ├── er-diagram.md                 # ER図（Mermaid）
│   ├── class-diagrams.md             # クラス図
│   ├── sequence-diagrams.md          # シーケンス図（29本: 正常系24 + 異常系5）
│   ├── activity-diagrams.md          # アクティビティ図
│   ├── state-machines.md             # ステートマシン図
│   ├── screen-transition.md          # 画面遷移図
│   ├── screen-specs.md               # 画面仕様（項目定義）
│   ├── frontend-architecture.md      # フロントエンド設計ガイド
│   ├── api-endpoints.md              # APIエンドポイント一覧（90件定義・88件登録）
│   ├── error-handling-and-security.md # エラー処理・セキュリティ
│   ├── operations-guide.md           # 運用ガイド
│   ├── dvnf-rep-type-spec.md         # DVNF/RepType仕様
│   ├── program-spec.md               # 主要プログラム仕様
│   ├── folder-structure.md           # フォルダ構成説明（本資料）
│   ├── dev-setup.md                  # 環境構築資料
│   ├── testing-guide.md              # テストガイド（実行・構成・トラブルシューティング）
│   ├── mcp-setup-guide.md            # MCPセットアップガイド
│   ├── plugin-system.md              # プラグインシステム仕様
│   ├── scenario.md                   # 利用シナリオ集（チャネル横断のend-to-endフロー）
│   └── user-guide.md                 # ユーザ向け導入資料
├── evidences/                        # 検証結果・エビデンス
├── gkill_model/                      # astah モデルのエクスポート
│   ├── 01_ユースケースモデル/
│   ├── 02_画面遷移モデル/
│   ├── 03_ERモデル/
│   └── 04_クラスモデル/
├── resources/                        # 画像リソース等
├── gkill_model.asta                  # astah モデル
├── gkill_model_latest.asta           # astah モデル（最新）
├── gkill_user_document.pdf           # ユーザ利用説明書
├── gkill_user_document.xlsx          # ユーザ利用説明書（原本）
└── gkill_develop_document.xlsx       # 開発者向け起点要件書・設計書
```

`documents/reverse/` は全24ファイル（README.md 含む）。

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
├── caches/                 # キャッシュファイル（thumb_cache/, video_cache/, zip_cache/, local_rep_cache/, git_commit_log_cache/, plugin_cache/ 等）
├── logs/                   # ログファイル（JSON形式、レベル別分割）
│   ├── gkill_error.log
│   ├── gkill_warn.log
│   ├── gkill_info.log
│   ├── gkill_access.log    # HTTPアクセスログ（--log access以上で出力）
│   ├── gkill_debug.log
│   ├── gkill_trace.log
│   ├── gkill_trace_sql.log
│   ├── gkill.log           # 統合ログ
│   ├── gkill_mcp_read_access.log      # Read MCPサーバアクセスログ（MCP_LOG環境変数で制御）
│   ├── gkill_mcp_write_access.log     # Write MCPサーバアクセスログ
│   └── gkill_mcp_readwrite_access.log # Read/Write MCPサーバアクセスログ
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
