# APIエンドポイント一覧

## 概要

gkill サーバーは gorilla/mux ベースの HTTP API を提供する。全エンドポイントは **POST メソッド**（一部 GET あり）で、`/api/` プレフィックス配下に配置される。

- **エンドポイント定義:** `src/server/gkill/api/gkill_server_api_address.go`（パス・メソッド定義）
- **ハンドラ登録:** `src/server/gkill/api/gkill_server_api.go`（ルーティング・ハンドラ実装）
- **リクエスト/レスポンス型:** `src/server/gkill/api/req_res/`（172 型定義）

## 共通仕様

### リクエスト共通フィールド

多くのエンドポイントは以下のフィールドを含む：

| フィールド | 型 | 説明 |
|---|---|---|
| `session_id` | string | ログインセッションID（login 以外の全エンドポイントで必須） |
| `locale_name` | string | ロケール名（`ja`, `en`, `zh`, `ko`, `es`, `fr`, `de`） |

### レスポンス共通構造

```json
{
  "messages": [
    { "message_code": "string", "message": "string" }
  ],
  "errors": [
    { "error_code": "string", "error_message": "string" }
  ],
  // ... エンドポイント固有のデータフィールド
}
```

- **正常:** HTTP 200 + `errors` が空配列
- **業務エラー:** HTTP 200 + `errors` に詳細あり（エラーコードで判別）
- **アクセス拒否:** HTTP 403（ローカルアクセス制限時）
- **サーバーエラー:** HTTP 500

### 主要エラーコード

| エラーコード | 説明 |
|---|---|
| `AccountNotFoundError` | アカウントが存在しない |
| `AccountInvalidPasswordError` | パスワード不一致 |
| `AccountIsNotEnableError` | アカウントが無効化されている |
| `AccountPasswordResetTokenIsNotNilError` | パスワードリセット中 |
| `AccountInvalidLoginResponseDataError` | ログインレスポンスデータ不正 |
| `GetDeviceError` | デバイス情報取得失敗 |

---

## 認証・アカウント管理（6件）

| パス | 説明 |
|---|---|
| `/api/login` | ログイン（user_id + password_sha256 → session_id） |
| `/api/logout` | ログアウト（セッション無効化） |
| `/api/reset_password` | パスワードリセット要求 |
| `/api/set_new_password` | 新パスワード設定（リセットトークン使用） |
| `/api/add_user` | アカウント追加 |
| `/api/update_account_status` | アカウント状態更新（有効/無効） |

## Kyouデータ追加（11件）

各データ型の新規レコードを作成する。Append-Only DAOのため、既存データを変更せず常にINSERTされる。

| パス | 対象データ型 | 説明 |
|---|---|---|
| `/api/add_kmemo` | Kmemo | テキストメモ追加 |
| `/api/add_kc` | KC | 数値記録追加 |
| `/api/add_urlog` | URLog | ブックマーク追加 |
| `/api/add_nlog` | Nlog | 支出記録追加 |
| `/api/add_timeis` | TimeIs | 打刻（タイムスタンプ）追加 |
| `/api/add_mi` | Mi | タスク追加 |
| `/api/add_lantana` | Lantana | 気分値（0-10）追加 |
| `/api/add_rekyou` | ReKyou | リポスト追加 |
| `/api/add_tag` | Tag | タグ追加（対象KyouのIDを指定） |
| `/api/add_text` | Text | テキスト注釈追加（対象KyouのIDを指定） |
| `/api/add_gkill_notification` | Notification | 通知追加（対象KyouのIDを指定） |

## Kyouデータ更新（12件）

Append-Only DAOのため「更新」は同一IDで新しいレコードをINSERTする。最新レコードが有効データとなる。

| パス | 対象データ型 | 説明 |
|---|---|---|
| `/api/update_kmemo` | Kmemo | テキストメモ更新 |
| `/api/update_kc` | KC | 数値記録更新 |
| `/api/update_urlog` | URLog | ブックマーク更新 |
| `/api/update_nlog` | Nlog | 支出記録更新 |
| `/api/update_timeis` | TimeIs | 打刻更新 |
| `/api/update_mi` | Mi | タスク更新 |
| `/api/update_lantana` | Lantana | 気分値更新 |
| `/api/update_idf_kyou` | IDFKyou | ファイル参照更新 |
| `/api/update_rekyou` | ReKyou | リポスト更新 |
| `/api/update_tag` | Tag | タグ更新 |
| `/api/update_text` | Text | テキスト注釈更新 |
| `/api/update_gkill_notification` | Notification | 通知更新 |

## Kyouデータ取得（13件）

| パス | 説明 |
|---|---|
| `/api/get_kyous` | 複数レコード取得（フィルタリング・ページネーション対応） |
| `/api/get_kyou` | 単一レコード取得（ID指定） |
| `/api/get_kmemo` | Kmemo取得 |
| `/api/get_kc` | KC取得 |
| `/api/get_urlog` | URLog取得 |
| `/api/get_nlog` | Nlog取得 |
| `/api/get_timeis` | TimeIs取得 |
| `/api/get_mi` | Mi取得 |
| `/api/get_lantana` | Lantana取得 |
| `/api/get_rekyou` | ReKyou取得 |
| `/api/get_git_commit_log` | Gitコミットログ取得 |
| `/api/get_idf_kyou` | IDFKyou（ファイル参照）取得 |
| `/api/get_updated_datas_by_time` | 指定時刻以降に更新されたデータ取得 |

## タグ・テキスト・通知 メタデータ取得（8件）

| パス | 説明 |
|---|---|
| `/api/get_all_tag_names` | 全タグ名一覧取得 |
| `/api/get_all_rep_names` | 全リポジトリ名一覧取得 |
| `/api/get_tags_by_id` | 対象KyouのIDに紐づくタグ一覧取得 |
| `/api/get_tag_histories_by_tag_id` | タグIDの履歴取得（Append-Only全バージョン） |
| `/api/get_texts_by_id` | 対象KyouのIDに紐づくテキスト一覧取得 |
| `/api/get_text_histories_by_text_id` | テキストIDの履歴取得 |
| `/api/get_gkill_notifications_by_id` | 対象KyouのIDに紐づく通知一覧取得 |
| `/api/get_gkill_notification_histories_by_notification_id` | 通知IDの履歴取得 |

## タスク管理（1件）

| パス | 説明 |
|---|---|
| `/api/get_mi_board_list` | Miタスクボード名一覧取得 |

## 設定管理（7件）

| パス | 説明 |
|---|---|
| `/api/get_application_config` | アプリケーション設定取得（KFTLテンプレート含む） |
| `/api/update_application_config` | アプリケーション設定更新 |
| `/api/get_server_configs` | サーバー設定取得 |
| `/api/update_server_configs` | サーバー設定更新 |
| `/api/get_repositories` | ユーザーのリポジトリ一覧取得 |
| `/api/update_user_reps` | リポジトリパス更新 |
| `/api/reload_repositories` | リポジトリ再読み込み |

## ファイル操作（4件）

| パス | 説明 |
|---|---|
| `/api/upload_files` | ファイルアップロード |
| `/api/upload_gpslog_files` | GPSログファイルアップロード |
| `/api/open_directory` | ディレクトリを開く（OS コマンド実行） |
| `/api/open_file` | ファイルを開く（OS コマンド実行） |

## KFTL（3件）

| パス | 説明 |
|---|---|
| `/api/submit_kftl_text` | KFTLテキスト送信・パース・保存 |
| `/api/commit_tx` | KFTLトランザクションコミット |
| `/api/discard_tx` | KFTLトランザクション破棄 |

## 共有（5件）

| パス | 説明 |
|---|---|
| `/api/get_share_kyou_list_infos` | 共有設定一覧取得 |
| `/api/add_share_kyou_list_info` | 共有設定追加 |
| `/api/update_share_kyou_list_info` | 共有設定更新 |
| `/api/delete_share_kyou_list_infos` | 共有設定削除 |
| `/api/get_shared_kyous` | 共有レコード取得（認証不要） |

## 通知・プッシュ（2件）

| パス | 説明 |
|---|---|
| `/api/get_gkill_notification_public_key` | VAPID公開鍵取得（Web Push用） |
| `/api/register_gkill_notification` | プッシュ通知登録 |

## GPS（1件）

| パス | 説明 |
|---|---|
| `/api/get_gps_log` | GPSログ取得（日付範囲指定） |

## MCP連携（1件）

| パス | 説明 |
|---|---|
| `/api/get_kyous_mcp` | MCP経由でのKyouデータ取得 |

## TLS・セキュリティ（1件）

| パス | 説明 |
|---|---|
| `/api/generate_tls_file` | TLS証明書ファイル生成 |

## その他（3件）

| パス | 説明 |
|---|---|
| `/api/urlog_bookmarklet` | URLogブックマークレット用エンドポイント |
| `/api/update_cache` | キャッシュ更新トリガー |
| `/api/get_gkill_info` | アプリケーション情報取得（※アドレス定義のみ、ハンドラ未確認） |

## 非APIルート

| パス | メソッド | 説明 |
|---|---|---|
| `/files/*` | GET | アップロードファイル配信 |
| `/serviceWorker.js` | GET | PWA Service Worker 配信 |
| `/` | GET | Vue SPA（embed された index.html） |

---

## 補足

- **合計:** POST エンドポイント 79件（うち77件はハンドラ登録済み、2件はアドレス定義のみ）+ 非APIルート 3件
- **全エンドポイント定義:** `src/server/gkill/api/gkill_server_api_address.go`
- **ハンドラ実装:** `src/server/gkill/api/gkill_server_api.go` および `src/server/gkill/api/handle_*.go`
- **リクエスト/レスポンス型:** `src/server/gkill/api/req_res/` 配下に各エンドポイント対応の構造体
- `get_kftl_template` と `get_gkill_info` はアドレス定義が存在するが、`HandleFunc` 登録は `gkill_server_api.go` のメイン登録ブロックでは確認されていない（別箇所で登録されている可能性あり）
