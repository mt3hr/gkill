# APIエンドポイント一覧

## 概要

gkill サーバーは gorilla/mux ベースの HTTP API を提供する。全エンドポイントは **POST メソッド**（一部 GET あり）で、`/api/` プレフィックス配下に配置される。

- **エンドポイント定義:** `src/server/gkill/api/gkill_server_api/gkill_server_api_address.go`（パス・メソッド定義）
- **ハンドラ実装:** `src/server/gkill/api/gkill_server_api/handle_*.go`（1ハンドラ1ファイル、93ファイル）
- **認証ミドルウェア:** `src/server/gkill/api/gkill_server_api/auth_middleware.go`（`wrapNoAuth`/`wrapAuth`/`wrapAuthRepos`でハンドラ登録）
- **リクエスト/レスポンス型:** `src/server/gkill/api/req_res/`（182ファイル）
- **ビジネスロジック:** `src/server/gkill/usecase/`（HTTP非依存のユースケース関数、17ファイル）

## 共通仕様

### 認証ラッパーの分類

各エンドポイントは `serve.go` で3つのラッパー関数のいずれかに包まれて登録される。

| ラッパー | 件数 | ミドルウェアが行うこと | 対象 |
|---|---|---|---|
| `wrapNoAuth` | 13 | `filterLocalOnly` のみ | `login`, `logout`, `reset_password`, `set_new_password`, `get_shared_kyous`, `urlog_bookmarklet`, `urlog_bookmarklet_page`, `get_kyous_mcp`, `upload_files`, `upload_gpslog_files`, `browse_zip_contents`, `get_idf_kyou_by_relative_path`, `get_idf_file_path` |
| `wrapAuth` | 19 | セッション検証 → Account / UserID / Device を `AuthContext` に設定 | `get_application_config`, `update_server_configs`, `add_user`, `generate_tls_file`, `update_cache`, プラグイン4本 等 |
| `wrapAuthRepos` | 56 | 上記に加えて `GkillRepositories` を解決 | データCRUD系（追加12 + 更新13 + 取得23 + 共有4 + リポジトリ/TX 4） |

> **`wrapNoAuth` = 認証なし、ではない。** 上表の `wrapNoAuth` のうち
> `upload_files` / `upload_gpslog_files` / `browse_zip_contents` /
> `get_idf_kyou_by_relative_path` / `get_idf_file_path` / `get_kyous_mcp` の6本は、
> **ハンドラ内部で `getAccountFromSessionID` を呼んでセッションを検証**する。
> ミドルウェアを通さないのは、これらがマルチパート相当の大きなボディや
> 独自のリクエスト形式を扱うため。
>
> `logout` / `reset_password` / `set_new_password` も `wrapNoAuth` に含まれる。
> `update_cache` は `wrapAuth` + ハンドラ内 `IsAdmin` チェックの組み合わせ。

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

### 代表的なリクエスト/レスポンス例

#### `/api/get_kyous` — 記録一覧取得

リクエストは `GetKyousRequest`（`session_id` + `query` + `locale_name`）。**検索条件は `query` オブジェクト（`FindQuery`）で包む**。出典: `src/server/gkill/api/req_res/get_kyous_request.go`, `src/server/gkill/api/find/find_query.go`。

`FindQuery` の主要フィールド：

| フィールド | 型 | 説明 |
|---|---|---|
| `use_words` | bool | テキスト検索を使用するか |
| `words` | []string | 検索キーワード |
| `words_and` | bool | `words` をAND条件にするか（false=OR） |
| `not_words` | []string | 除外キーワード |
| `use_tags` | bool | タグフィルタを使用するか |
| `tags` | []string | タグ名一覧 |
| `tags_and` | bool | `tags` をAND条件にするか |
| `use_reps` | bool | リポジトリフィルタを使用するか |
| `reps` | []string | リポジトリ名一覧 |
| `use_timeis` | bool | TimeIsフィルタを使用するか |
| `include_end_timeis` | bool | 終了済みTimeIsを含めるか |
| `use_calendar` | bool | 日付範囲フィルタを使用するか |
| `calendar_start_date` | string | 検索開始日時（RFC3339） |
| `calendar_end_date` | string | 検索終了日時（RFC3339） |
| `only_latest_data` | bool | 最新データのみ取得するか |
| `include_deleted_data` | bool | 削除済みデータを含めるか |

> ページング用の `page` / `page_size` フィールドは存在しない。`FindQuery` にページング機構はない。

```json
// リクエスト例
{
  "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "query": {
    "use_words": true,
    "words": ["今日", "作業"],
    "words_and": true,
    "use_tags": false,
    "tags": [],
    "use_reps": false,
    "reps": [],
    "use_calendar": true,
    "calendar_start_date": "2026-06-01T00:00:00+09:00",
    "calendar_end_date":   "2026-06-30T23:59:59+09:00"
  },
  "locale_name": "ja"
}

// レスポンス例（GetKyousResponse。messages / errors / kyous のみ）
{
  "kyous": [
    {
      "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "create_time": "2026-06-21T10:00:00+09:00",
      "create_app": "gkill",
      "create_device": "desktop",
      "create_user": "admin",
      "update_time": "2026-06-21T10:00:00+09:00",
      "is_deleted": false,
      "rep_name": "kmemo_desktop",
      "data_type": "kmemo"
    }
  ],
  "messages": [],
  "errors": []
}
```

> `Kyou` の実体（`typed_kmemo` など `typed_*` フィールド）はサーバーの `get_kyous` レスポンスには含まれない。クライアントが必要に応じて後から個別ロードする値。

#### `/api/submit_kftl_text` — KFTLテキスト送信

プレフィックスは**日本語と ASCII の2系統**があり、どちらも常に受理される（`kftl_factory.go:8-45`）。

| 用途 | 日本語 | ASCII |
|---|---|---|
| タグ | `。` | `#` |
| テキスト開始/終了 | `ーー` | `--` |
| 関連時刻 | `？` | `?` |
| 区切り | `、` | `,` |
| 区切り（+1秒） | `、、` | `,,` |
| 数値（KC） | `ーか` | `/num` |
| タスク（Mi） | `ーみ` | `/mi` |
| 気分（Lantana） | `ーら` | `/mood` |
| 支出（Nlog） | `ーん` | `/expense` |
| ブックマーク（URLog） | `ーう` | `/url` |
| 打刻開始 | `ーた` | `/start` |
| 打刻終了 | `ーえ` | `/end` |
| 打刻終了（存在時のみ） | `ーいえ` | `/end?` |
| 打刻終了（タグ指定） | `ーたえ` | `/endt` |
| 打刻終了（タグ指定・存在時のみ） | `ーいたえ` | `/endt?` |
| 打刻（開始+終了） | `ーち` | `/timeis` |
| 保存 | `！` | `!` |

`/`系プレフィックスは**行全体が完全一致**する必要があり、値は**次の行**に書く。`/mood 8` のようなインライン記法は無効。値付きの例: `/mood`＋改行＋`8`、`/num`＋改行＋`体重`＋改行＋`65.5`。例外として `#`(タグ) と `?`(関連時刻) のみ行内に内容を続けられる。出典: `src/server/gkill/api/kftl/kftl_factory.go`。

```json
// リクエスト例（\n は改行。/mood と /num の値は次行に置く）
{
  "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "kftl_text": "今日の朝食\n#食事\n/mood\n8\n/num\n体重\n65.5",
  "locale_name": "ja"
}

// レスポンス例（成功時）
{
  "messages": [
    { "message_code": "KFTLSubmitSuccess", "message": "KFTLテキストを記録しました" }
  ],
  "errors": []
}
```

#### `/api/upload_files` — ファイルアップロード

**JSON で送信する（multipart/form-data ではない）**。ファイル本体は base64（データURI）文字列として JSON に含める。出典: `handle_upload_files.go`, `upload_files_request.go`, `file_data.go`, `file_upload_conflict_behavior.go`, `upload_files_response.go`。

| フィールド | 型 | 説明 |
|---|---|---|
| `session_id` | string | セッションID |
| `target_rep_name` | string | アップロード先リポジトリ名 |
| `conflict_behavior` | string | 衝突時の挙動: `"override"` / `"rename"` / `"merge"` |
| `files` | FileData[] | アップロードファイル（複数可） |
| `locale_name` | string | ロケール名 |

`FileData`: `file_name` / `data_base64` / `last_modified`(RFC3339)。

```json
// リクエスト例
{
  "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "target_rep_name": "files_desktop",
  "conflict_behavior": "rename",
  "files": [
    {
      "file_name": "photo.jpg",
      "data_base64": "data:image/jpeg;base64,/9j/4AAQ...",
      "last_modified": "2026-06-21T10:00:00+09:00"
    }
  ],
  "locale_name": "ja"
}

// レスポンス例（UploadFilesResponse。uploaded_kyous を返す）
{
  "uploaded_kyous": [
    {
      "id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
      "data_type": "idf_kyou",
      "rep_name": "files_desktop"
    }
  ],
  "messages": [],
  "errors": []
}
```

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
| `/api/login` | ログイン（user_id + password_sha256 → session_id）。サーバは受け取った64桁hexを資格情報としてArgon2idで照合する。パスワード未設定のアカウントは常に不一致（fail-closed） |
| `/api/logout` | ログアウト。**セッションを解決できたときだけ削除**する（未認証で任意の session_id を投げても他人のセッションは消えない）。解決できなくても応答は成功のまま返す（べき等性を保ち、session_id が有効だったかを応答から読み取れないようにするため） |
| `/api/reset_password` | パスワードリセット要求（管理者のみ）。対象アカウントのパスワードを無効化し、有効期限72時間のリセットトークンを発行する |
| `/api/set_new_password` | 新パスワード設定（リセットトークン使用）。①IP単位のレート制限（ログインとは**別カウンタ**で15分10回）②`new_password_sha256` は64桁小文字hexでなければ拒否 ③トークンは constant-time 照合し期限も検証 ④成功時に**当該ユーザーの全ログインセッションを削除**する |
| `/api/add_user` | アカウント追加。`user_id` は `^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$` かつ `..` を含まないことを検証する（利用者IDがキャッシュディレクトリ名になるため）。応答の Account にパスワードは含まれない |
| `/api/update_account_status` | アカウント状態更新（有効/無効） |

## Kyouデータ追加（12件）

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
| `/api/add_mirekyou` | MiReKyou | リポストタスク追加 |
| `/api/add_tag` | Tag | タグ追加（対象KyouのIDを指定） |
| `/api/add_text` | Text | テキスト注釈追加（対象KyouのIDを指定） |
| `/api/add_gkill_notification` | Notification | 通知追加（対象KyouのIDを指定） |

## Kyouデータ更新（13件）

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
| `/api/update_mirekyou` | MiReKyou | リポストタスク更新 |
| `/api/update_tag` | Tag | タグ更新 |
| `/api/update_text` | Text | テキスト注釈更新 |
| `/api/update_gkill_notification` | Notification | 通知更新 |

## Kyouデータ取得（14件）

| パス | 説明 |
|---|---|
| `/api/get_kyous` | 複数レコード取得（フィルタリング。ページングは無く、条件に合う全件を返す） |
| `/api/get_kyou` | 単一レコード取得（ID指定） |
| `/api/get_kmemo` | Kmemo取得 |
| `/api/get_kc` | KC取得 |
| `/api/get_urlog` | URLog取得 |
| `/api/get_nlog` | Nlog取得 |
| `/api/get_timeis` | TimeIs取得 |
| `/api/get_mi` | Mi取得 |
| `/api/get_lantana` | Lantana取得 |
| `/api/get_rekyou` | ReKyou取得 |
| `/api/get_mirekyou` | MiReKyou取得 |
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
| `/api/get_server_configs` | サーバー設定取得（管理者のみ）。応答に含まれる Account から**パスワードハッシュは除かれる**（`Account.PasswordHash` が `json:"-"` のため型レベルで載らない）。`password_reset_token` は管理画面がリセットリンクの生成に使うので残る |
| `/api/update_server_configs` | サーバー設定更新 |
| `/api/get_repositories` | ユーザーのリポジトリ一覧取得 |
| `/api/update_user_reps` | リポジトリパス更新 |
| `/api/reload_repositories` | リポジトリ再読み込み |

## ファイル操作（7件）

| パス | 説明 |
|---|---|
| `/api/upload_files` | ファイルアップロード |
| `/api/upload_gpslog_files` | GPSログファイルアップロード |
| `/api/open_directory` | ディレクトリを開く（OS コマンド実行） |
| `/api/open_file` | ファイルを開く（OS コマンド実行） |
| `/api/browse_zip_contents` | ZIPファイル内容閲覧。IDFKyouのZIPファイルを `$HOME/gkill/caches/zip_cache/{user_id}/{rep_name}/{sha1}/` に展開し、ZipEntry リスト（ファイル名・サイズ・パス等）を返却する。セッション認証必須。パストラバーサル防止、Shift_JISファイル名デコード、アトミック展開に対応 |
| `/api/get_idf_kyou_by_relative_path` | IDFKyou相対パス解決。基準IDFKyou（`target_id`）のファイルからの相対パス（`relative_path`）を同一Rep内で解決し、対象ファイルのIDFKyou IDを返却する（Markdown内相対リンクのKyouDialog表示用）。見つからない場合は `kyou_id` 空文字。セッション認証必須。パストラバーサル防止対応 |
| `/api/get_idf_file_path` | IDFファイル絶対パス解決。`rep_name` + `file_name` から実ファイルの絶対パス（`file_path`）を返却する。MCPクライアントがbase64転送を経ずにファイルを直接読むための導線。**リクエスト元がlocalhostのときのみ応答**し、それ以外は `file_path` 空 + `ERR000389`。DB登録済みファイルしか引けないためパストラバーサル不可。リポジトリに無い場合は `exists` false。セッション認証必須 |

## KFTL（2件）

| パス | 説明 |
|---|---|
| `/api/get_kftl_template` | KFTLテンプレート構造取得（※アドレス定義のみ、ハンドラ未実装。リクエストは404となる。テンプレートは `get_application_config` 経由で取得する） |
| `/api/submit_kftl_text` | KFTLテキスト送信・パース・保存 |

## トランザクション（2件）

| パス | 説明 |
|---|---|
| `/api/commit_tx` | トランザクションコミット（一時リポジトリ → 本リポジトリに反映） |
| `/api/discard_tx` | トランザクション破棄（一時リポジトリを破棄） |

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

## MCP連携（1件 + MCPツール10個）

| パス | 説明 |
|---|---|
| `/api/get_kyous_mcp` | MCP経由でのKyouデータ取得（IDFペイロードに`rep_name`/`is_image`等含む） |

MCPサーバは10個のReadツールを提供する。内訳は固有の8つ（`gkill_get_kyous`, `gkill_get_mi_board_list`, `gkill_get_all_tag_names`, `gkill_get_all_rep_names`, `gkill_get_gps_log`, `gkill_get_application_config`, `gkill_get_idf_file`, `gkill_get_idf_file_path`）と、3サーバ共通のプラグインツール2つ（`gkill_get_plugin_list`, `gkill_get_plugin_content`。`src/mcp/lib/plugin-tools.mjs` の `PLUGIN_TOOLS` を各サーバの `TOOLS` 配列に展開している）。`gkill_get_idf_file` はバックエンドの `/files/{repName}/{filePath}` エンドポイントをプロキシしてIDFファイルの実データを返す。`gkill_get_idf_file_path` は `/api/get_idf_file_path` を経由してファイルの絶対パスを返す（stdio接続のローカルクライアント用）。

## TLS・セキュリティ（1件）

| パス | 説明 |
|---|---|
| `/api/generate_tls_file` | TLS証明書ファイル生成 |

## その他（4件）

| パス | 説明 |
|---|---|
| `/api/urlog_bookmarklet` | URLogブックマークレット用エンドポイント。ブラウザのブックマークレットから現在のページのURL・タイトルをURLogとして直接追加する。ログイン時にブックマークレット専用セッション（`ApplicationName="urlog_bookmarklet"`）が自動作成され、通常のセッションとは分離される |
| `/api/urlog_bookmarklet_page` | URLogブックマークレット導入ページ配信（GET）。ブックマークレット登録用のHTMLページを返す |
| `/api/update_cache` | キャッシュ更新トリガー。**管理者セッション必須**（`wrapAuth` + `IsAdmin`）。`session_id` と `user_ids` を受け取り、指定ユーザーのインメモリキャッシュを再構築する。CLI `gkill_server update_cache ユーザーID...` は対象ユーザーIDの文字列配列を受け取り、**認証情報の指定は不要**（ローカルの `configs/account.db` から有効な管理者アカウントを自動選択し、その名義で有効期限5分のログインセッションを `configs/account_state.db` へ直接発行して呼び出し、完了後に削除する。パスワードはArgon2idで保存されており DB から復元できないため `/api/login` は経由しない） |
| `/api/get_gkill_info` | アプリケーション情報取得（※アドレス定義のみ、ハンドラ未実装。リクエストは404となる。将来の拡張用と推定） |

## プラグイン（4件）

| パス | 説明 |
|---|---|
| `/api/get_plugin_list` | インストール済みプラグイン一覧取得（名前・バージョン・説明・rep_name・is_alive） |
| `/api/get_plugin_content_html` | プラグイン Kyou のコンテンツ HTML 取得 |
| `/api/get_plugin_config_html` | プラグイン設定画面 HTML 取得 |
| `/api/post_plugin_config` | プラグイン設定フォームのデータ保存 |

### get_plugin_list 詳細

| 項目 | 内容 |
|---|---|
| リクエスト型 | `GetPluginListRequest` |
| 主要フィールド | `session_id`, `locale_name` |
| レスポンス型 | `GetPluginListResponse` |
| レスポンスフィールド | `plugins: Array<PluginInfo>`（`name`, `version`, `description`, `data_type`, `rep_name`, `is_alive`） |
| 備考 | MCPの `gkill_get_plugin_list` もこのエンドポイントをそのまま使う |
| 備考 | `is_alive` はプラグインプロセスの生存状態を示す。プロセスが起動済みの場合 `true`、停止中は `false` |

### get_plugin_content_html 詳細

| 項目 | 内容 |
|---|---|
| リクエスト型 | `GetPluginContentHTMLRequest` |
| 主要フィールド | `session_id`, `rep_name`, `kyou_id` |
| レスポンス型 | `GetPluginContentHTMLResponse` |
| レスポンスフィールド | `html` — iframe srcdoc に直接セット |
| PWAキャッシュキー | `/cache/api/plugin_content_html/{kyou_id}` |
| 備考 | キャッシュ識別子に `kyou_id` を使用（他エンドポイントの `id` / `target_id` とは命名が異なる）。MCPの `gkill_get_plugin_content` もこのエンドポイントを叩き、既定でHTMLをプレーンテキストに変換してAIに返す |

### post_plugin_config 詳細

| 項目 | 内容 |
|---|---|
| リクエスト型 | `PostPluginConfigRequest` |
| 主要フィールド | `session_id`, `rep_name`, `form_data: Record<string, string>` |
| レスポンス型 | `PostPluginConfigResponse` |
| 備考 | `plugin-config-dialog.vue` の iframe srcdoc 内フォームの送信データを受け取り、プラグインプロセスの `post_config` コマンドへ転送する |

---

## 非APIルート

`serve.go` に登録される API 以外のルートは19件（`PathPrefix` 18 + `Path` 1）。

| パス | メソッド | 説明 |
|---|---|---|
| `/files/*` | GET | アップロードファイル配信。**cookie**（`gkill_session_id`）で認証する |
| `/zip_cache/*` | GET | ZIP展開済みファイル配信。`/api/browse_zip_contents` で展開されたファイルを配信する。ルータ上は `wrapNoAuth` で、認証はハンドラ内の cookie で行う。**配信の起点はセッションから引いた利用者のディレクトリ `$HOME/gkill/caches/zip_cache/{user_id}/` に固定**され、利用者IDはURLに現れないため他人のディレクトリを指名できない。rep名は利用者間で重複しうるので、rep名の照合ではなくディレクトリの分離で担保している |
| `/serviceWorker.js` | GET | PWA Service Worker 配信 |
| `/resources/manual/*` | GET | HTMLマニュアル配信（7言語）。`filterLocalOnly` によるアクセス制御付き |
| `/` | GET | Vue SPA（embed された index.html）。`router.Path("/")` として個別登録 |
| `/rykv` `/kftl` `/mi` `/mkfl` `/kyou` `/dashboard` `/saihate` `/plaing` | GET | 同一SPAを配信（各パスが `PathPrefix` として個別登録される） |
| `/shared_page` `/shared_mi` `/shared_rykv` | GET | 共有ページ用SPA。認証不要 |
| `/set_new_password` `/regist_first_account` | GET | SPA。**この2つだけ `ifRedirectResetAdminAccountIsNotFound` を通らない**（`serve.go:262-276`）。管理者アカウント未設定時のリダイレクト先そのものなので、リダイレクト判定を通すとループするため |
| （上記以外） | GET | catch-all の `PathPrefix("/")` が同一SPAを配信 |

> `/shared_rykv` はサーバ側では配信されるが、`src/client/router/index.ts`（13ルート）に
> 対応するルート定義が無い。

---

## 補足

- **合計:** `/api/` エンドポイント 90件定義（うち88件はハンドラ登録済み、2件はアドレス定義のみ。メソッドはPOST中心、一部GET）+ 非APIルート 19件（PathPrefix 18 + Path 1）
- **全エンドポイント定義:** `src/server/gkill/api/gkill_server_api/gkill_server_api_address.go`
- **ハンドラ実装:** `src/server/gkill/api/gkill_server_api/handle_*.go`（1ハンドラ1ファイル）
- **リクエスト/レスポンス型:** `src/server/gkill/api/req_res/` 配下に各エンドポイント対応の構造体（182ファイル）
- **ビジネスロジック:** `src/server/gkill/usecase/` 配下にHTTP非依存のユースケース関数（17ファイル）
- `get_kftl_template` と `get_gkill_info` はアドレス定義（`gkill_server_api_address.go`）が存在するが、`HandleFunc` 登録もハンドラ関数実装も存在しない。コードベース全体を調査した結果、これらは**未実装のエンドポイント**であることが確認された。リクエストは404となる
