# req_res - API リクエスト/レスポンス構造体

## 概要

全 HTTP API エンドポイントの Request / Response 構造体を定義するパッケージ。
各エンドポイントに対して `*_request.go` と `*_response.go` のペアが存在する。
フロントエンド側の対応実装: `src/client/classes/api/req_res/`

## 設計パターン

### ファイル命名規則

```
<操作>_<対象>_request.go   — リクエスト構造体
<操作>_<対象>_response.go  — レスポンス構造体
```

操作: `add_`, `update_`, `get_`, `delete_`, `login_`, `logout_`, `upload_`, `commit_`, `discard_`, `submit_`, `open_`, `generate_`, `register_`, `reload_`, `reset_`, `set_new_`

### 共通フィールド

**Request 共通:**
- `SessionID string` — 認証セッション（login 以外）

**Response 共通:**
- `Errors []message.GkillError` — エラー情報
- `Messages []message.GkillMessage` — メッセージ情報

## 全ファイル一覧（155ファイル）

### 認証系（10ファイル）

| ファイル | 説明 |
|---------|------|
| `login_request.go` / `login_response.go` | ログイン |
| `logout_request.go` / `logout_response.go` | ログアウト |
| `reset_password_request.go` / `reset_password_response.go` | パスワードリセット |
| `set_new_password_request.go` / `set_new_password_response.go` | 新パスワード設定 |
| `add_account_request.go` / `add_account_response.go` | アカウント追加 |

### データ追加系（28ファイル）

| ファイル | データ型 |
|---------|---------|
| `add_kc_request.go` / `add_kc_response.go` | KC（数値記録） |
| `add_kmemo_request.go` / `add_kmemo_response.go` | Kmemo（テキストメモ） |
| `add_kyou_info_request.go` / `add_kyou_info_response.go` | Kyou 情報 |
| `add_lantana_request.go` / `add_lantana_response.go` | Lantana（気分値） |
| `add_mi_request.go` / `add_mi_response.go` | Mi（タスク） |
| `add_nlog_request.go` / `add_nlog_response.go` | Nlog（支出記録） |
| `add_notification_request.go` / `add_notification_response.go` | Notification |
| `add_re_kyou_request.go` / `add_re_kyou_response.go` | ReKyou（リポスト） |
| `add_share_kyou_list_info_request.go` / `add_share_kyou_list_info_response.go` | 共有リスト情報 |
| `add_tag_request.go` / `add_tag_response.go` | Tag |
| `add_text_request.go` / `add_text_response.go` | Text |
| `add_time_is_request.go` / `add_time_is_response.go` | TimeIs（タイムスタンプ） |
| `add_ur_log_request.go` / `add_ur_log_response.go` | URLog（ブックマーク） |

### データ更新系（30ファイル）

| ファイル | データ型 |
|---------|---------|
| `update_idf_kyou_request.go` / `update_idf_kyou_response.go` | IDFKyou（ファイル） |
| `update_kc_request.go` / `update_kc_response.go` | KC |
| `update_kmemo_request.go` / `update_kmemo_response.go` | Kmemo |
| `update_lantana_request.go` / `update_lantana_response.go` | Lantana |
| `update_mi_request.go` / `update_mi_response.go` | Mi |
| `update_nlog_request.go` / `update_nlog_response.go` | Nlog |
| `update_notification_request.go` / `update_notification_response.go` | Notification |
| `update_re_kyou_request.go` / `update_re_kyou_response.go` | ReKyou |
| `update_share_kyou_list_info_request.go` / `update_share_kyou_list_info_response.go` | 共有リスト情報 |
| `update_tag_request.go` / `update_tag_response.go` | Tag |
| `update_text_request.go` / `update_text_response.go` | Text |
| `update_timeis_request.go` / `update_timeis_response.go` | TimeIs |
| `update_ur_log_request.go` / `update_ur_log_response.go` | URLog |
| `update_account_status_request.go` / `update_account_status_response.go` | アカウントステータス |
| `update_share_kyou_list_infos_response.go` | 共有リスト情報（複数） |

### データ取得系（56ファイル）

#### 個別データ取得

| ファイル | データ型 |
|---------|---------|
| `get_kyou_request.go` / `get_kyou_response.go` | 単一 Kyou |
| `get_kyous_request.go` / `get_kyous_response.go` | Kyou 一覧 |
| `get_kc_request.go` / `get_kc_response.go` | KC |
| `get_kmemo_request.go` / `get_kmemo_response.go` | Kmemo |
| `get_lantana_request.go` / `get_lantana_response.go` | Lantana |
| `get_mi_request.go` / `get_mi_response.go` | Mi |
| `get_nlog_request.go` / `get_nlog_response.go` | Nlog |
| `get_re_kyou_request.go` / `get_re_kyou_response.go` | ReKyou |
| `get_timeis_request.go` / `get_timeis_response.go` | TimeIs |
| `get_ur_log_request.go` / `get_ur_log_response.go` | URLog |
| `get_idf_kyou_request.go` / `get_idf_kyou_response.go` | IDFKyou |
| `get_git_commit_log_request.go` / `get_git_commit_log_response.go` | Git コミットログ |
| `get_gps_log_request.go` / `get_gps_log_response.go` | GPS ログ |
| `get_mi_board_request.go` / `get_mi_board_response.go` | Mi ボード |

#### メタ情報取得

| ファイル | 説明 |
|---------|------|
| `get_tags_by_target_id_request.go` / `get_tags_by_target_id_response.go` | 対象 ID に紐づくタグ一覧 |
| `get_tag_history_by_tag_id_request.go` / `get_tag_history_by_tag_id_response.go` | タグの履歴 |
| `get_texts_by_target_id_request.go` / `get_texts_by_target_id_response.go` | 対象 ID に紐づくテキスト一覧 |
| `get_text_history_by_text_id_request.go` / `get_text_history_by_text_id_response.go` | テキストの履歴 |
| `get_notifications_by_target_id_request.go` / `get_notifications_by_target_id_response.go` | 対象 ID に紐づく通知一覧 |
| `get_notifications_history_by_text_id_request.go` / `get_notification_history_by_text_id_response.go` | 通知の履歴 |
| `get_all_tag_names_request.go` / `get_all_tag_names_response.go` | 全タグ名一覧 |
| `get_all_rep_names_request.go` / `get_all_rep_names_response.go` | 全リポジトリ名一覧 |

#### 設定・構成取得

| ファイル | 説明 |
|---------|------|
| `get_application_config_request.go` / `get_application_config_response.go` | アプリケーション設定 |
| `get_server_configs_request.go` / `get_server_configs_response.go` | サーバ設定 |
| `get_repositories_request.go` / `get_repositories_response.go` | リポジトリ一覧 |

#### 共有・通知・その他取得

| ファイル | 説明 |
|---------|------|
| `get_shared_kyous_request.go` / `get_shared_kyous_response.go` | 共有 Kyou |
| `get_share_kyou_list_infos_request.go` / `get_share_kyou_list_infos_response.go` | 共有リスト情報 |
| `get_gkill_notification_public_key_request.go` / `get_gkill_notification_public_key_response.go` | Web Push 公開鍵 |
| `get_updated_datas_by_time_request.go` / `get_updated_datas_by_time_response.go` | 時刻指定更新データ |

### 設定更新系（8ファイル）

| ファイル | 説明 |
|---------|------|
| `update_application_config_request.go` / `update_application_config_response.go` | アプリケーション設定更新 |
| `update_server_config_request.go` / `update_server_config_response.go` | サーバ設定更新 |
| `update_user_reps_request.go` / `update_user_reps_response.go` | ユーザリポジトリ更新 |
| `update_cache_request.go` / `update_cache_response.go` | キャッシュ更新 |

### ファイル操作系（8ファイル）

| ファイル | 説明 |
|---------|------|
| `upload_files_request.go` / `upload_files_response.go` | ファイルアップロード |
| `upload_gps_log_files_request.go` / `upload_gps_log_files_response.go` | GPS ログファイルアップロード |
| `open_directory_request.go` / `open_directory_response.go` | ディレクトリを開く |
| `open_file_request.go` / `open_file_response.go` | ファイルを開く |

### トランザクション系（4ファイル）

| ファイル | 説明 |
|---------|------|
| `commit_tx_request.go` / `commit_tx_response.go` | トランザクションコミット |
| `discard_tx_request.go` / `discard_tx_response.go` | トランザクション破棄 |

### 共有削除系（2ファイル）

| ファイル | 説明 |
|---------|------|
| `delete_share_kyou_list_info_request.go` / `delete_share_kyou_list_infos_response.go` | 共有リスト情報削除 |

### その他操作系（6ファイル）

| ファイル | 説明 |
|---------|------|
| `generate_tls_file_request.go` / `generate_tls_file_response.go` | TLS 証明書生成 |
| `register_gkill_notification_request.go` / `register_gkill_notification_response.go` | Web Push 通知登録 |
| `reload_repositoriers_request.go` / `reload_repositories_response.go` | リポジトリ再読み込み |
| `submit_kftl_text_request.go` / `submit_kftl_text_response.go` | KFTL テキスト送信 |
| `urlog_bookmarklet_request.go` | URLog ブックマークレットアドレス |
| `get_kyous_mcp_request.go` / `get_kyous_mcp_response.go` | MCP 用 Kyou 取得 |

### ユーティリティ型（5ファイル）

| ファイル | 説明 |
|---------|------|
| `account.go` | Account 構造体定義 |
| `file_data.go` | FileData 構造体（アップロードファイル情報） |
| `file_upload_conflict_behavior.go` | ファイルアップロード競合時の振る舞い定義 |
| `share_kyou_list_info.go` | ShareKyouListInfo 構造体 |
| `kyou_mcp_dto.go` | MCP 向け Kyou DTO 定義 |

## 開発ガイドライン

### 新しいエンドポイントを追加する場合

1. `<操作>_<対象>_request.go` を作成し、Request 構造体を定義
2. `<操作>_<対象>_response.go` を作成し、Response 構造体を定義
3. Request に `SessionID` フィールドを含める（認証不要の場合を除く）
4. Response に `Errors` と `Messages` フィールドを含める
5. `gkill_server_api.go` にハンドラを追加
6. TypeScript 側 `src/client/classes/api/req_res/` にも対応する型を追加
