# api - HTTP API 層

## 概要

HTTP API の共通基盤とハンドラ層。全エンドポイントは POST メソッドで `/api/` パス配下に配置される。
コンパイル済み Vue 3 SPA を `//go:embed` ディレクティブでバイナリに埋め込み、ルート `/` で配信する。
ハンドラの実装は `gkill_server_api/` サブパッケージに集約されている。

## ディレクトリ構造

```
api/
├── embed.go                     # //go:embed ディレクティブ（Vue SPA 配信）
├── filter_tags_kyous_test.go    # タグ絞り込みテスト
├── find_filter.go               # 検索フィルタロジック
├── find_filter_helpers.go       # 検索フィルタヘルパー
├── find_filter_test.go          # 検索フィルタテスト
├── find_kyou_context.go         # Kyou 検索コンテキスト
├── gkill_version_data.go        # バージョンデータ構造体
├── select_match_reps_cache_test.go # 対象リポジトリ選択キャッシュテスト
├── version.go                   # バージョン情報
├── gkill_server_api/            # HTTP ハンドラ（122ファイル）
│   ├── gkill_server_api.go      # GkillServerAPI 構造体定義
│   ├── gkill_server_api_address.go # エンドポイントアドレス定義
│   ├── serve.go                 # gorilla/mux ルーター設定・全90ルート登録
│   ├── close.go                 # サーバ終了処理
│   ├── auth.go                  # 認証処理
│   ├── auth_context.go          # 認証コンテキスト
│   ├── auth_middleware.go       # 認証ミドルウェア
│   ├── filter_local_only.go     # ローカル限定フィルタ
│   ├── utils.go                 # ユーティリティ関数
│   ├── web_push.go              # Web Push 通知
│   ├── gkill_server_api_access_log.go # アクセスログ
│   ├── gkill_server_api_rate_limit.go # レートリミット
│   └── handle_*.go              # 各エンドポイントのハンドラ（96ファイル。うちテスト5）
├── find/                        # 検索クエリ型定義
├── gkill_plugin/                # プラグイン通信プロトコル型
├── gpslogs/                     # GPS ログパーサ
├── kftl/                        # KFTL パーサ → kftl/README.md 参照
├── message/                     # エラー/メッセージコード
├── req_res/                     # Request/Response 構造体 → req_res/README.md 参照
└── embed/                       # ビルド生成物（.gitignore 対象）
```

## api/ ルートレベルファイル（9ファイル）

| ファイル | 役割 |
|---------|------|
| `embed.go` | `//go:embed embed` ディレクティブ。ビルド時にフロントエンドの dist/ をバイナリに埋め込む |
| `filter_tags_kyous_test.go` | タグ絞り込みのテスト |
| `find_filter.go` | Kyou の検索フィルタロジック。FindQuery に基づいたデータ絞り込み |
| `find_filter_helpers.go` | 検索フィルタのヘルパー関数群 |
| `find_filter_test.go` | 検索フィルタのテスト |
| `find_kyou_context.go` | Kyou 検索時のコンテキスト構造体 |
| `gkill_version_data.go` | バージョンデータ構造体定義 |
| `select_match_reps_cache_test.go` | 対象リポジトリ選択キャッシュのテスト |
| `version.go` | ビルド時に埋め込まれるバージョン情報 |

## サブディレクトリ

### `gkill_server_api/`（115ファイル）— HTTP ハンドラ

詳細は [gkill_server_api/README.md](gkill_server_api/README.md) を参照。

`GkillServerAPI` 構造体に全ハンドラメソッドを集約。gorilla/mux で全90エンドポイントを登録する。
handle_*.go は99ファイル（実装91 + テスト8）で、1ハンドラ1ファイルとして分割されている。
ビジネスロジックは `usecase/` 層に委譲し、ハンドラは HTTP リクエスト/レスポンスの変換に専念する。

### `find/`（5ファイル）— 検索クエリ型定義

詳細は [find/README.md](find/README.md) を参照。

| ファイル | 説明 |
|---------|------|
| `find_query.go` | `FindQuery` 構造体 — 検索条件（41フィールド: キーワード、日付範囲、タグ、データ型等。値がnullなら未使用） |
| `mi_check_state.go` | Mi（タスク）のチェック状態 enum |
| `mi_sort_type.go` | Mi のソート順 enum |
| `week_of_days.go` | 曜日フィルタ enum |
| `find_query_test.go` | JSON シリアライズ・デシリアライズテスト |

### `gkill_plugin/`（2ファイル）— プラグイン通信プロトコル型

| ファイル | 説明 |
|---------|------|
| `plugin_manifest.go` | `PluginManifest` 構造体 — manifest.json の型 |
| `plugin_protocol.go` | `PluginRequest` / `PluginResponse` / `PluginKyou` — stdio 改行区切り JSON のメッセージ型 |

### `gpslogs/`（2ファイル）— GPS ログパーサ

| ファイル | 説明 |
|---------|------|
| `google_location_history_data.go` | Google Location History の JSON/GPX データ構造体 |
| `gpslogs_test.go` | GPS データパーステスト |

### `message/`（5ファイル）— エラー/メッセージコード

詳細は [message/README.md](message/README.md) を参照。

| ファイル | 説明 |
|---------|------|
| `gkill_error.go` | `GkillError` 構造体 — API エラーレスポンス用 |
| `gkill_message.go` | `GkillMessage` 構造体 — API メッセージレスポンス用 |
| `error_codes.go` | エラーコード定数（409定数、ERR000001〜ERR000410・ERR000243欠番） |
| `message_codes.go` | メッセージコード定数（88定数） |
| `message_test.go` | コード形式テスト |

### `kftl/`（25ファイル）— KFTL パーサ

詳細は [kftl/README.md](kftl/README.md) を参照。

### `req_res/`（186ファイル）— Request/Response 構造体

詳細は [req_res/README.md](req_res/README.md) を参照。

## 全エンドポイント一覧（92エンドポイント定義・90登録）

全エンドポイントは `/api/` 配下に配置（POST 中心、一部 GET）。`gkill_server_api/serve.go` 内で gorilla/mux に登録。`GetKFTLTemplate` と `GetGkillInfo` の2件はアドレス定義のみで未登録。`GkillWebpushServiceWorkerJs`（`/serviceWorker.js`、GET）だけは `/api/` 配下ではなく、`router.PathPrefix` で別途登録している。

### 認証系（5エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `Login` | ログイン（user_id + password_sha256 → session_id） |
| `Logout` | ログアウト |
| `ResetPassword` | パスワードリセット |
| `SetNewPassword` | 新パスワード設定 |
| `AddAccount` | アカウント追加 |

### データ追加系（12エンドポイント）

| エンドポイント | データ型 |
|---------------|---------|
| `AddTag` | Tag |
| `AddText` | Text |
| `AddNotification` | Notification |
| `AddKmemo` | Kmemo（テキストメモ） |
| `AddKC` | KC（数値記録） |
| `AddURLog` | URLog（ブックマーク） |
| `AddNlog` | Nlog（支出記録） |
| `AddTimeis` | TimeIs（タイムスタンプ） |
| `AddMi` | Mi（タスク） |
| `AddLantana` | Lantana（気分値） |
| `AddRekyou` | ReKyou（リポスト） |
| `AddMiReKyou` | MiReKyou（既存記録のタスク化。タイトルを持たず target_id で元の Kyou を指す） |

### データ更新系（14エンドポイント）

| エンドポイント | データ型 |
|---------------|---------|
| `UpdateTag` | Tag |
| `UpdateText` | Text |
| `UpdateNotification` | Notification |
| `UpdateKmemo` | Kmemo |
| `UpdateKC` | KC |
| `UpdateURLog` | URLog |
| `UpdateNlog` | Nlog |
| `UpdateTimeis` | TimeIs |
| `UpdateLantana` | Lantana |
| `UpdateIDFKyou` | IDFKyou（ファイル） |
| `UpdateMi` | Mi |
| `UpdateRekyou` | ReKyou |
| `UpdateMiReKyou` | MiReKyou |
| `UpdateAccountStatus` | アカウントステータス |

### データ取得系（15エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetKyous` | Kyou 一覧取得（検索クエリ付き） |
| `GetKyou` | 単一 Kyou 取得 |
| `GetKmemo` | Kmemo 取得 |
| `GetKC` | KC 取得 |
| `GetURLog` | URLog 取得 |
| `GetNlog` | Nlog 取得 |
| `GetTimeis` | TimeIs 取得 |
| `GetMi` | Mi 取得 |
| `GetLantana` | Lantana 取得 |
| `GetRekyou` | ReKyou 取得 |
| `GetMiReKyou` | MiReKyou 取得 |
| `GetGitCommitLog` | Git コミットログ取得 |
| `GetIDFKyou` | IDFKyou 取得 |
| `GetGPSLog` | GPS ログ取得 |
| `GetUpdatedDatasByTime` | 時刻指定で更新データ取得 |

### メタ情報取得系（11エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetMiBoardList` | Mi ボード一覧 |
| `GetAllTagNames` | 全タグ名一覧 |
| `GetAllRepNames` | 全リポジトリ名一覧 |
| `GetTagsByTargetID` | 対象 ID に紐づくタグ一覧 |
| `GetTagHistoriesByTagID` | タグの変更履歴 |
| `GetTextsByTargetID` | 対象 ID に紐づくテキスト一覧 |
| `GetTextHistoriesByTextID` | テキストの変更履歴 |
| `GetNotificationsByTargetID` | 対象 ID に紐づく通知一覧 |
| `GetNotificationHistoriesByNotificationID` | 通知の変更履歴 |
| `GetReKyousByTargetID` | 対象 ID を参照している ReKyou 一覧（Kyou 連鎖削除の逆引きに使う） |
| `GetMiReKyousByTargetID` | 対象 ID を参照している MiReKyou 一覧（同上） |

### 設定系（8エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetApplicationConfig` | アプリケーション設定取得 |
| `UpdateApplicationConfig` | アプリケーション設定更新 |
| `GetServerConfigs` | サーバ設定取得 |
| `UpdateServerConfigs` | サーバ設定更新 |
| `UpdateUserReps` | ユーザリポジトリ更新 |
| `GetRepositories` | リポジトリ一覧取得 |
| `ReloadRepositories` | リポジトリ再読み込み |
| `UpdateCache` | キャッシュ更新 |

### ファイル操作系（7エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `UploadFiles` | ファイルアップロード |
| `UploadGPSLogFiles` | GPS ログファイルアップロード |
| `OpenDirectory` | ディレクトリを OS で開く |
| `OpenFile` | ファイルを OS で開く |
| `BrowseZipContents` | IDFKyou の ZIP ファイル内容閲覧（展開・キャッシュ・パストラバーサル防止） |
| `GetIDFKyouByRelativePath` | 基準 IDFKyou からの相対パスで同一 Rep 内のファイル記録を解決（Markdown 内相対リンク用） |
| `GetIDFFilePath` | IDF ファイルの絶対パス解決（localhost からのリクエストのみ応答。MCP stdio クライアント用） |

### 共有系（5エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetShareKyouListInfos` | 共有リスト情報取得 |
| `AddShareKyouListInfo` | 共有リスト情報追加 |
| `UpdateShareKyouListInfo` | 共有リスト情報更新 |
| `DeleteShareKyouListInfos` | 共有リスト情報削除 |
| `GetSharedKyous` | 共有 Kyou 取得 |

### プラグイン系（4エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetPluginList` | インストール済みプラグイン一覧取得（名前・バージョン・説明・rep_name・is_alive） |
| `GetPluginContentHTML` | プラグイン Kyou のコンテンツ HTML 取得 |
| `GetPluginConfigHTML` | プラグイン設定画面 HTML 取得 |
| `PostPluginConfig` | プラグイン設定フォームのデータ保存 |

### 通知・TLS・トランザクション・その他（11エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetGkillNotificationPublicKey` | Web Push 通知公開鍵取得 |
| `RegisterGkillNotification` | Web Push 通知登録 |
| `GenerateTLSFile` | TLS 証明書ファイル生成 |
| `CommitTX` | トランザクションコミット |
| `DiscardTX` | トランザクション破棄 |
| `URLogBookmarklet` | URLog ブックマークレットアドレス取得 |
| `URLogBookmarkletPage` | URLog ブックマークレット導入ページ配信（GET） |
| `SubmitKFTLText` | KFTL テキスト送信・実行 |
| `GetKyousMCP` | MCP 用 Kyou 取得 |
| `GetKFTLTemplate` | KFTL テンプレート取得（※アドレス定義のみ、未登録） |
| `GetGkillInfo` | アプリケーション情報取得（※アドレス定義のみ、未登録） |

## 開発ガイドライン

### ハンドラの追加方法

1. `req_res/` に Request/Response 構造体を追加
2. `gkill_server_api/gkill_server_api_address.go` にアドレス定数を追加
3. `gkill_server_api/handle_xxx.go` にハンドラメソッドを実装（1ハンドラ1ファイル）
4. ビジネスロジックは `usecase/` 層に実装し、ハンドラから呼び出す
5. `gkill_server_api/serve.go` の `resetRouter()` 内で `router.HandleFunc()` を登録

### 命名規則

- ハンドラメソッド: `HandleXxx(w, r)` パターン
- アドレス定数: `XxxAddress` パターン
- 全ハンドラは `GkillServerAPI` 構造体のメソッドとして実装
