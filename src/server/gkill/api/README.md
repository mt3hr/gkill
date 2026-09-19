# api - HTTP API 層

## 概要

HTTP API の共通基盤とハンドラ層。全エンドポイントは POST メソッドで `/api/` パス配下に配置される。
コンパイル済み Vue 3 SPA を `//go:embed` ディレクティブでバイナリに埋め込み、ルート `/` で配信する。
ハンドラの実装は `gkill_server_api/` サブパッケージに集約されている。

## ディレクトリ構造

```
api/
├── embed.go                     # //go:embed ディレクティブ（Vue SPA 配信）
├── find_filter.go               # 検索フィルタロジック
├── find_filter_helpers.go       # 検索フィルタヘルパー
├── find_kyou_context.go         # Kyou 検索コンテキスト
├── gkill_version_data.go        # バージョンデータ構造体
├── version.go                   # バージョン情報
├── *_test.go                    # 検索フィルタ・rep名絞り込みなどのテスト（14ファイル）
├── gkill_server_api/            # HTTP ハンドラ（156ファイル）
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
│   └── handle_*.go              # 各エンドポイントのハンドラ（106ファイル。うちテスト15）
├── find/                        # 検索クエリ型定義
├── find_word/                   # ワード検索の Go 側判定（本体とプラグイン SDK が共用。標準ライブラリのみ）
├── gkill_plugin/                # プラグイン通信プロトコル型
├── gpslogs/                     # GPS ログパーサ
├── kftl/                        # KFTL パーサ → kftl/README.md 参照
├── message/                     # エラー/メッセージコード
├── req_res/                     # Request/Response 構造体 → req_res/README.md 参照
├── safefetch/                   # SSRF 対策付き HTTP 取得ヘルパ
└── embed/                       # ビルド生成物（.gitignore 対象）
```

## api/ ルートレベルファイル（20ファイル）

| ファイル | 役割 |
|---------|------|
| `embed.go` | `//go:embed embed` ディレクティブ。ビルド時にフロントエンドの dist/ をバイナリに埋め込む |
| `find_filter.go` | Kyou の検索フィルタロジック。FindQuery に基づいたデータ絞り込み |
| `find_filter_helpers.go` | 検索フィルタのヘルパー関数群 |
| `find_kyou_context.go` | Kyou 検索時のコンテキスト構造体 |
| `gkill_version_data.go` | バージョンデータ構造体定義 |
| `version.go` | ビルド時に埋め込まれるバージョン情報 |
| `*_test.go`（14ファイル） | 検索フィルタ（タグ・位置・Mi・TimeIs・ソート等）・rep名絞り込み・rep選択キャッシュ・rep種別網羅・サンプルデータのテスト |

## サブディレクトリ

### `gkill_server_api/`（141ファイル）— HTTP ハンドラ

詳細は [gkill_server_api/README.md](gkill_server_api/README.md) を参照。

`GkillServerAPI` 構造体に全ハンドラメソッドを集約。gorilla/mux で全91エンドポイントを登録する。
handle_*.go は112ファイル（実装92 + テスト20）で、1ハンドラ1ファイルとして分割されている。
ビジネスロジックは `usecase/` 層に委譲し、ハンドラは HTTP リクエスト/レスポンスの変換に専念する。

### `find/`（10ファイル）— 検索クエリ型定義

詳細は [find/README.md](find/README.md) を参照。

| ファイル | 説明 |
|---------|------|
| `find_query.go` | `FindQuery` 構造体 — 検索条件（40フィールド: キーワード、日付範囲、タグ、データ型等。値がnullなら未使用）と、検索語の正規化 `WithNormalizedWords` |
| `find_query_legacy_json.go` | 旧形式（`use_*` フラグ入り）JSON を新形式へ書き換える移行ウォーカー |
| `period_of_time.go` | 時間帯フィルタの秒値正規化ヘルパー |
| `rep_types.go` | `KyouRepTypes` — `RepTypes` が受理する正準値の一覧 |
| `mi_check_state.go` | Mi（タスク）のチェック状態 enum |
| `mi_sort_type.go` | Mi のソート順 enum |
| `week_of_days.go` | 曜日フィルタ enum |
| `find_query_test.go` | JSON シリアライズ・デシリアライズテスト |
| `find_query_legacy_json_test.go` | 旧形式 JSON 移行のテスト |
| `period_of_time_test.go` | 時間帯フィルタの秒値正規化のテスト |

### `find_word/`（2ファイル）— ワード検索の Go 側判定

| ファイル | 説明 |
|---------|------|
| `match_words.go` | `MatchLoweredWords`（肯定語は「対象テキストに含む OR ID が語で始まる」、除外語は対象テキストだけ）/ `LowerWords` / `NormalizeWords`。SQL 側の `sqlite3impl.GenerateFindSQLCommon` と同じ規則で、本体（IDF / git / プラグイン型別アダプタ）とプラグイン SDK（`sdk.Query.MatchText`）が共用する。SDK が引き込めるよう標準ライブラリ以外に依存しない |
| `match_words_test.go` | 判定規則（AND/OR・除外語・ID 前方一致・ID 照合なし）と正規化のテスト |

### `gkill_plugin/`（3ファイル）— プラグイン通信プロトコル型

| ファイル | 説明 |
|---------|------|
| `plugin_manifest.go` | `PluginManifest` 構造体 — manifest.json の型 |
| `plugin_manifest_test.go` | manifest 解釈のテスト（`emits_kyou` 未指定時の既定値など） |
| `plugin_protocol.go` | `PluginRequest` / `PluginResponse` / `PluginKyou` — stdio 改行区切り JSON のメッセージ型 |

### `gpslogs/`（2ファイル）— GPS ログパーサ

| ファイル | 説明 |
|---------|------|
| `google_location_history_data.go` | Google Location History の JSON/GPX データ構造体 |
| `gpslogs_test.go` | GPS データパーステスト |

### `message/`（8ファイル）— エラー/メッセージコード

詳細は [message/README.md](message/README.md) を参照。

| ファイル | 説明 |
|---------|------|
| `gkill_error.go` | `GkillError` 構造体 — API エラーレスポンス用。`EnsureNotEmpty`（エラー無し失敗の受け皿）もここ |
| `gkill_error_test.go` | `EnsureNotEmpty` のテスト |
| `gkill_message.go` | `GkillMessage` 構造体 — API メッセージレスポンス用 |
| `error_codes.go` | エラーコード定数（382定数、ERR000001〜ERR000423・欠番41。うち37は存在しないエンドポイントのコードを 2026-09-14 に削除したもの。ADR-0709） |
| `message_codes.go` | メッセージコード定数（83定数、MSG000001〜MSG000090・欠番7） |
| `http_status.go` | エラーコード → HTTP ステータス対応表（`HTTPStatusOf` / `HTTPStatusForErrors`） |
| `http_status_test.go` | 全エラーコードが対応表に載っていることのソース走査テスト |
| `message_test.go` | コード形式テスト |

### `kftl/`（32ファイル）— KFTL パーサ

詳細は [kftl/README.md](kftl/README.md) を参照。

### `req_res/`（189ファイル）— Request/Response 構造体

詳細は [req_res/README.md](req_res/README.md) を参照。

### `safefetch/`（2ファイル）— SSRF 対策付き HTTP 取得

| ファイル | 説明 |
|---------|------|
| `safefetch.go` | `GetCapped`（scheme 検査・接続先 IP 検証・サイズ上限）、`LooksLikeSupportedImage` / `CheckImageDimensions`。利用者入力由来の URL 取得はここを通す |
| `safefetch_test.go` | SSRF 防御・サイズ上限・画像判定のテスト |

## 全エンドポイント一覧（91エンドポイント）

全エンドポイントは `/api/` 配下に配置（POST 中心、一部 GET）。正本は `gkill_server_api/gkill_server_api_address.go` のルート表 `apiRoutes()` で、`serve.go` の `registerAPIRoutes` が gorilla/mux へそのまま登録する（テストハーネスも同じ表を使う。ADR-0709）。Service Worker（`/serviceWorker.js`、GET）だけは API ではなく、`router.PathPrefix` で別途配信している。

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

### ファイル操作系（6エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `UploadFiles` | ファイルアップロード |
| `UploadGPSLogFiles` | GPS ログファイルアップロード |
| `OpenDirectory` | ディレクトリを OS で開く |
| `OpenFile` | ファイルを OS で開く |
| `BrowseZipContents` | IDFKyou の ZIP ファイル内容閲覧（展開・キャッシュ・パストラバーサル防止） |
| `GetIDFKyouByRelativePath` | 基準 IDFKyou からの相対パスで同一 Rep 内のファイル記録を解決（Markdown 内相対リンク用） |

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

### 通知・TLS・トランザクション・その他（10エンドポイント）

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
| `GetRepInfosMCP` | MCP 用 rep 一覧取得（Kyou を供給する rep の rep_name と rep_type、rep_types の正準値一覧。ファイルパスは返さない） |

## 開発ガイドライン

### ハンドラの追加方法

1. `req_res/` に Request/Response 構造体を追加
2. `gkill_server_api/handle_xxx.go` にハンドラメソッドを実装（1ハンドラ1ファイル。doc コメントは「1行説明 / 空行 / `POST /api/xxx（wrapXxx）` / req_res 型」）
3. ビジネスロジックは `usecase/` 層に実装し、ハンドラから呼び出す
4. `gkill_server_api/gkill_server_api_address.go` のルート表 `apiRoutes()` に1行足す（パス・メソッド・認証区分・無認証ならボディ上限・ハンドラ）。`serve.go` に `HandleFunc` を直に書かない
5. `api_routes_test.go` の golden（認証区分の固定）にも1行足す。Web クライアントから叩くなら `gkill-api.ts` に `xxx_address` / `xxx_method` とメソッドを足す（`gkill-api.test.ts` が表と突き合わせる）

### 命名規則

- ハンドラメソッド: `HandleXxx(w, r)` パターン
- ルート表の行: `{Path: "/api/xxx", Method: "POST", Auth: authXxx, Body: bodyXxx, Handler: g.HandleXxx},`（1行で書く。verify_docs と TS のテストが正規表現で読む）
- 全ハンドラは `GkillServerAPI` 構造体のメソッドとして実装
