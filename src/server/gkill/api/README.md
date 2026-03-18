# api - HTTP API 層

## 概要

gorilla/mux を使用した HTTP API ハンドラ層。全エンドポイントは POST メソッドで `/api/` パス配下に配置される。
コンパイル済み Vue 3 SPA を `//go:embed` ディレクティブでバイナリに埋め込み、ルート `/` で配信する。

## ディレクトリ構造

```
api/
├── gkill_server_api.go          # メインハンドラ（全エンドポイント登録、~557KB）
├── gkill_server_api_address.go  # エンドポイントアドレス定義
├── embed.go                     # //go:embed ディレクティブ（Vue SPA 配信）
├── find_filter.go               # 検索フィルタロジック
├── find_filter_helpers.go       # 検索フィルタヘルパー
├── find_kyou_context.go         # Kyou 検索コンテキスト
├── gkill_version_data.go        # バージョンデータ構造体
├── version.go                   # バージョン情報
├── handle_submit_kftl_text.go   # KFTL テキスト送信ハンドラ
├── handle_get_kyous_mcp.go      # MCP 用 Kyou 取得ハンドラ
├── handle_update_cache.go       # キャッシュ更新ハンドラ
├── find/                        # 検索クエリ型定義
├── gpslogs/                     # GPS ログパーサ
├── kftl/                        # KFTL パーサ → kftl/README.md 参照
├── message/                     # エラー/メッセージコード
├── req_res/                     # Request/Response 構造体 → req_res/README.md 参照
└── embed/                       # ビルド生成物（.gitignore 対象）
```

## ルートレベルファイル（11ファイル）

| ファイル | 役割 |
|---------|------|
| `gkill_server_api.go` | **メインハンドラファイル**（約557KB）。`GkillServerAPI` 構造体に全ハンドラメソッドを集約。gorilla/mux ルーター設定と全76エンドポイントの登録 |
| `gkill_server_api_address.go` | `GkillServerAPIAddress` 構造体 — 全エンドポイントの URL パス定義 |
| `embed.go` | `//go:embed embed` ディレクティブ。ビルド時にフロントエンドの dist/ をバイナリに埋め込む |
| `find_filter.go` | Kyou の検索フィルタロジック。FindQuery に基づいたデータ絞り込み |
| `find_filter_helpers.go` | 検索フィルタのヘルパー関数群 |
| `find_kyou_context.go` | Kyou 検索時のコンテキスト構造体 |
| `gkill_version_data.go` | バージョンデータ構造体定義 |
| `version.go` | ビルド時に埋め込まれるバージョン情報 |
| `handle_submit_kftl_text.go` | `/api/submit_kftl_text` エンドポイントの独立ハンドラ |
| `handle_get_kyous_mcp.go` | `/api/get_kyous_mcp` エンドポイントの独立ハンドラ |
| `handle_update_cache.go` | `/api/update_cache` エンドポイントの独立ハンドラ |

## サブディレクトリ

### `find/`（4ファイル）— 検索クエリ型定義

| ファイル | 説明 |
|---------|------|
| `find_query.go` | `FindQuery` 構造体 — 検索条件（キーワード、日付範囲、タグ、データ型等） |
| `mi_check_state.go` | Mi（タスク）のチェック状態 enum |
| `mi_sort_type.go` | Mi のソート順 enum |
| `week_of_days.go` | 曜日フィルタ enum |

### `gpslogs/`（1ファイル）— GPS ログパーサ

| ファイル | 説明 |
|---------|------|
| `google_location_history_data.go` | Google Location History の JSON/GPX データ構造体 |

### `message/`（4ファイル）— エラー/メッセージコード

| ファイル | 説明 |
|---------|------|
| `gkill_error.go` | `GkillError` 構造体 — API エラーレスポンス用 |
| `gkill_message.go` | `GkillMessage` 構造体 — API メッセージレスポンス用 |
| `error_codes.go` | エラーコード定数定義 |
| `message_codes.go` | メッセージコード定数定義 |

### `kftl/`（19ファイル）— KFTL パーサ

詳細は [kftl/README.md](kftl/README.md) を参照。

### `req_res/`（155ファイル）— Request/Response 構造体

詳細は [req_res/README.md](req_res/README.md) を参照。

## 全エンドポイント一覧（76エンドポイント）

全エンドポイントは `POST /api/` 配下に配置。`gkill_server_api.go` 内で gorilla/mux に登録。

### 認証系（5エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `Login` | ログイン（user_id + password_sha256 → session_id） |
| `Logout` | ログアウト |
| `ResetPassword` | パスワードリセット |
| `SetNewPassword` | 新パスワード設定 |
| `AddAccount` | アカウント追加 |

### データ追加系（11エンドポイント）

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

### データ更新系（13エンドポイント）

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
| `UpdateAccountStatus` | アカウントステータス |

### データ取得系（14エンドポイント）

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
| `GetGitCommitLog` | Git コミットログ取得 |
| `GetIDFKyou` | IDFKyou 取得 |
| `GetGPSLog` | GPS ログ取得 |
| `GetUpdatedDatasByTime` | 時刻指定で更新データ取得 |

### メタ情報取得系（9エンドポイント）

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

### ファイル操作系（4エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `UploadFiles` | ファイルアップロード |
| `UploadGPSLogFiles` | GPS ログファイルアップロード |
| `OpenDirectory` | ディレクトリを OS で開く |
| `OpenFile` | ファイルを OS で開く |

### 共有系（5エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetShareKyouListInfos` | 共有リスト情報取得 |
| `AddShareKyouListInfo` | 共有リスト情報追加 |
| `UpdateShareKyouListInfo` | 共有リスト情報更新 |
| `DeleteShareKyouListInfos` | 共有リスト情報削除 |
| `GetSharedKyous` | 共有 Kyou 取得 |

### 通知・TLS・トランザクション・その他（7エンドポイント）

| エンドポイント | 説明 |
|---------------|------|
| `GetGkillNotificationPublicKey` | Web Push 通知公開鍵取得 |
| `RegisterGkillNotification` | Web Push 通知登録 |
| `GenerateTLSFile` | TLS 証明書ファイル生成 |
| `CommitTX` | トランザクションコミット |
| `DiscardTX` | トランザクション破棄 |
| `URLogBookmarklet` | URLog ブックマークレットアドレス取得 |
| `SubmitKFTLText` | KFTL テキスト送信・実行 |
| `GetKyousMCP` | MCP 用 Kyou 取得 |

## 開発ガイドライン

### ハンドラの追加方法

1. `req_res/` に Request/Response 構造体を追加
2. `gkill_server_api_address.go` にアドレス定数を追加
3. `gkill_server_api.go` にハンドラメソッドを実装
4. `gkill_server_api.go` の `resetRouter()` 内で `router.HandleFunc()` を登録
5. 独立したハンドラの場合は `handle_xxx.go` として分離可能

### 命名規則

- ハンドラメソッド: `HandleXxx(w, r)` パターン
- アドレス定数: `XxxAddress` パターン
- 全ハンドラは `GkillServerAPI` 構造体のメソッドとして実装
