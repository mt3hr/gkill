# message - エラーコード・メッセージコード

## 概要

gkill API のエラーコードとメッセージコードの定数定義、およびレスポンス用構造体を提供する。
全 API レスポンスは `GkillMessages`（`messages`）と `GkillErrors`（`errors`）を含み、クライアント側でエラー判定・表示に使用される。
**成功時も `messages` / `errors` は `null` ではなく `[]`**（2026-09-15、[ADR-0710](../../../../../documents/adr/0710-error-kind-and-reason-on-the-wire.md)）。

## ファイル一覧（14ファイル）

| ファイル | 説明 |
|---------|------|
| `gkill_error.go` | `GkillError` 構造体 — `ErrorCode` + `ErrorMessage` + `Cause`（JSON には出ない Go の error）。`GkillErrors`（nil を `[]` で出す名前付きスライス）、`EnsureNotEmpty`（エラー無し失敗の受け皿）、`ErrorMessage` を伏せて `error_kind` / `reason` を付ける `MarshalJSON` もここ |
| `gkill_error_test.go` | `EnsureNotEmpty`、`MarshalJSON` の伏せ処理、ワイヤの形（`[]` / `error_kind` / `reason` / `Cause` が漏れない）のテスト |
| `error_kind.go` | `KindOf(code)` — エラーコード → `error_kind`（誰の問題か）。既定は HTTP ステータスから、`errorCodeKindOverride` で 500 のうち設定不備を `config` に |
| `error_kind_test.go` | 全コードに kind が決まること、上書き表の実在、名指しの割り当て、kind ごとの件数 |
| `error_reason.go` | `ReasonOf(err)` — Go の error → `reason`（何が起きたか）。`Reasoner` インタフェース / `ReasonError`、SQLite 結果コード・context・OS・net の分類、`errorCodeReason`（Cause 無しでコードから決まる理由） |
| `error_reason_test.go` | 分類の表駆動テスト（modernc.org/sqlite が実際に返すエラーで CANTOPEN / NOTADB / READONLY / BUSY を起こす）、コードからの既定、語彙の一覧 |
| `redact.go` | `RedactEnvironmentSpecific` — レスポンスへ載る自由文からホームのユーザー名・メールアドレスを伏せる（[ADR-0707](../../../../../documents/adr/0707-redact-environment-specific-strings.md)） |
| `redact_test.go` | 伏せ方の表駆動テスト（形は残す・二重適用しない・`@example.` は残す） |
| `gkill_message.go` | `GkillMessage` 構造体 — `MessageCode` + `Message` + `Level`（`info` 既定 / `warning`）。`GkillMessages`（nil を `[]` で出す） |
| `error_codes.go` | エラーコード定数（403 定数: `ERR000001` 〜 `ERR000444`、欠番 41。うち 37 は存在しないエンドポイント（`get_gkill_info` / `get_kftl_template` / `update_*_struct` 等）のコードを 2026-09-14 に削除した跡。ADR-0709） |
| `message_codes.go` | メッセージコード定数（86 定数: `MSG000001` 〜 `MSG000093`、欠番 7） |
| `http_status.go` | エラーコード → HTTP ステータス対応表（`errorCodeHTTPStatus`、`HTTPStatusOf` / `HTTPStatusForErrors`） |
| `http_status_test.go` | 全エラーコードが対応表に載っていることを `error_codes.go` のソース走査で固定するテスト |
| `message_test.go` | コード形式・空文字チェックのテスト |

## レスポンス構造体

```go
type GkillError struct {
    ErrorCode    string `json:"error_code"`
    ErrorMessage string `json:"error_message"`
    Cause        error  `json:"-"` // if err != nil の中で組み立てるときは必ず Cause: err
}
// ワイヤ: {"error_code":"ERR000023","error_message":"メモ追加に失敗しました","error_kind":"server","reason":"db_busy"}
// error_kind は必ず載る（KindOf）。reason は Cause から分類できたとき、またはコードから決まるときだけ（reasonForError）。

type GkillMessage struct {
    MessageCode string `json:"message_code"`
    Message     string `json:"message"`
    Level       string `json:"level"` // "info"（空なら MarshalJSON が補う）/ "warning"
}
```

### error_kind（誰の問題か）— 9 種

| kind | 由来 | 消費者の既定のヒント |
|---|---|---|
| `input` | 400 | 入力・送信内容を確認。画面から操作していて出るなら再読込して最新版で |
| `auth` | 401 | （Web の `check_auth` がログイン画面へ飛ばす） |
| `permission` | 403 | 権限が無い（管理者アカウント） |
| `not_found` | 404 | 別の端末で削除・更新された可能性。一覧を再読込 |
| `conflict` | 409 | 同じ記録が既にある（二重送信）。一覧を再読込 |
| `too_large` | 413 | 送信データを小さく |
| `rate_limit` | 429 | しばらく待つ |
| `config` | 500 のうち `errorCodeKindOverride`（TLS ファイル無し・書き込み rep 未設定） | 設定画面で直す |
| `server` | 500（既定）・未分類のコード | 再試行 → `gkill_error.log` |

### reason（何が起きたか）— 14 種

`ReasonOf` は `errors.Is` / `errors.As` だけで判定する（文字列は見ない）。順序は Reasoner → context → SQLite 結果コード → OS → net。

| reason | 判定 |
|---|---|
| `write_rep_missing` | `reps.CommitTxWriteRepMissingError` / コード `WriteRepMissingError` |
| `storage_unavailable` | `SQLITE_CANTOPEN` / `fs.ErrNotExist` |
| `storage_corrupted` | `SQLITE_CORRUPT` / `SQLITE_NOTADB` |
| `storage_readonly` | `SQLITE_READONLY` / `fs.ErrPermission` |
| `storage_full` | `SQLITE_FULL` / `syscall.ENOSPC` |
| `storage_io_error` | `SQLITE_IOERR` |
| `db_busy` | `SQLITE_BUSY` / `SQLITE_LOCKED` |
| `timeout` | `context.DeadlineExceeded` / `net.Error.Timeout()` |
| `canceled` | `context.Canceled`（Web は表示しない。ログも Debug） |
| `plugin_busy` | `reps.ErrPluginBusy` |
| `plugin_error` | `reps.ErrPluginReturnedErrors` |
| `external_fetch_failed` | `*url.Error` / `net.Error` |
| `local_only_access` | コード `LocalOnlyAccessDeniedError` |
| `tls_files_missing` | コード `NotFoundTLSCertFileError` / `NotFoundTLSKeyFileError` |

語彙の一覧は `reasonTokens` / `errorKinds` の1スライスに固定してあり、Web 側の表（`src/client/classes/api/message/error-hints.ts`）と `error-hints.test.ts` が突き合わせる。

## エラーコード体系（403 コード）

| コード範囲 | カテゴリ |
|-----------|---------|
| `ERR000001` 〜 `ERR000421` | アカウント・認証（ログイン、パスワードリセット等） |
| `ERR000023` 〜 `ERR000095` | CRUD 操作（Add, Update, Get, NotFound — 全データ型） |
| `ERR000098` 〜 `ERR000199` | 複合操作（共有リスト、MiSharedTasks、GPSLog 等） |
| `ERR000203` 〜 `ERR000280` | 拡張機能（KFTL、通知、構造体、Danote） |
| `ERR000300` 〜 `ERR000407` | 新機能（KC 数値記録、トランザクション、ZIP ブラウズ、レートリミット、プラグイン、ReKyou/MiReKyou の対象ID逆引き） |
| `ERR000408` 〜 `ERR000409` | アカウント運用の防護（リセットトークンの期限切れ、自分自身の無効化の拒否） |
| `ERR000410` | 検索そのものの失敗（repのSQLエラーなど、個別のGkillErrorが立たない失敗の受け皿） |
| `ERR000411`, `ERR000412` | MCP向けrep一覧（get_rep_infos_mcp）のリクエスト/レスポンス不正 |
| `ERR000413` 〜 `ERR000415` | 操作対象アカウント不在（404）、ローカル限定アクセス拒否（403）、panic回収（500） |
| `ERR000416` | メモ帳（KFTL）のテキスト自体が正しくない（400）。行ごとに1件ずつ立てる。サーバ側の失敗は `ERR000351`（500）のまま |
| `ERR000417` 〜 `ERR000418` | 認証系ミドルウェアの先読み `readAuthBody` の失敗（ボディ上限超過は 413、読み取り失敗は 500） |
| `ERR000419` 〜 `ERR000421` | commit_tx の ROLLBACK（何も書かれていない）、`parse_kftl_text` のリクエスト不正 / サーバ側失敗 |
| `ERR000422` | 書き込み先 rep 未設定（500 だが kind `config` / reason `write_rep_missing`。以前は nil ポインタ panic） |
| `ERR000423` | メモ帳（KFTL）の再送キー `idempotency_key` を別の本文で使い回した（409）。同じ本文なら元の `created[]` を `replayed:true` で返す |
| `ERR000424` 〜 `ERR000444` | スキル（`$GKILL_HOME/skills/<user_id>/<name>/`。ADR-0634）の API。リクエスト不正（400）・スキルやファイルが無い（404）・名前 / パス / SKILL.md / zip の不正（400）・既にある / revision の食い違い（409）・SKILL.md 単独の削除（400）・操作ごとのサーバ側失敗（500） |

`ERR9000xx` 帯はフロントエンドだけで採番するコードで、Go 側の `error_codes.go` には存在しない（定義元は `src/client/classes/api/message/gkill_error.ts`、現在101定数）。番号が衝突しないよう帯を分けてあるので、Go 側でこの帯を使ってはならない。

## メッセージコード体系（89 コード）

| コード範囲 | カテゴリ |
|-----------|---------|
| `MSG000001` 〜 `MSG000004` | 認証（ログイン、ログアウト、パスワードリセット） |
| `MSG000005` 〜 `MSG000024` | CRUD 成功（Add, Update 系） |
| `MSG000025` 〜 `MSG000040` | Get 操作成功 |
| `MSG000041` 〜 `MSG000074` | 設定・アップロード・KC・トランザクション |
| `MSG000075` 〜 `MSG000090` | KFTL、MCP、キャッシュ、通知、ZIP ブラウズ、ReKyou/MiReKyou の対象ID逆引き、プラグイン検索失敗の警告、MCP向けrep一覧、rep読み込み失敗の警告 |
| `MSG000091` 〜 `MSG000093` | スキルのアップロード（置き換え）・ファイルの書き込み・削除の成功（ADR-0634） |

`MSG000088`（プラグイン検索失敗）と `MSG000090`（rep 読み込み失敗）は `Level: warning` で返す。成功はしたが対処が要る知らせで、Web は閉じるまで残す（info は 2.5 秒で消える）。

## HTTP ステータスコードとの関係

**エラーコードごとの対応表が `http_status.go` にある**（`errorCodeHTTPStatus`）。
`HTTPStatusOf(errorCode)` で引ける。`errors` が空なら 200。

| HTTP ステータス | 割り当てているエラーコード | 件数 |
|----------------|--------------------------|------|
| 200 | （`errors` が空のとき） | — |
| 400 | `Invalid*RequestDataError` ほか入力値のバリデーション失敗 | 96 |
| 401 | `AccountSessionNotFoundError` / `AccountSessionExpiredError` / `AccountInvalidPasswordError` / `AccountNotFoundError` | 4 |
| 403 | `AccountNotHasAdminError` / `AccountDisabledError` / `LocalOnlyAccessDeniedError` ほか | 6 |
| 404 | `NotFound*` / `TargetAccountNotFoundError` | 18 |
| 409 | `AlreadyExist*` / `AccountPasswordResetTokenIsNotNilError` / `SubmitKFTLTextIdempotencyKeyConflictError` | 17 |
| 413 | `RequestBodyTooLargeError`（認証前ボディの上限超過。`writeGkillErrorResponse` 経由で JSON 本文つきで返る） | 1 |
| 429 | `LoginRateLimitError` | 1 |
| 500 | `Get*` / `Add*` / `Update*` / `Delete*` / `CommitTx*` / `Invalid*ResponseDataError` / `WriteRepMissingError` ほか | 239 |

**エラーコードを足したら表にも1行足すこと。** `http_status_test.go` が
`error_codes.go` をソース走査して未分類のコードを落とす（kind は `KindOf` がステータスから導くので、
`config` にしたいときだけ `errorCodeKindOverride` へ足す）。
名前の規則からは導けない（`Invalid*` が 400 と 500 に、`NotFound*` が 401 と 404 に跨る）ので、
推論に置き換えないこと。経緯と却下案は
[ADR-0706](../../../../../documents/adr/0706-http-status-from-error-code.md)。

## 関連ドキュメント

- [gkill_server_api/README.md](../gkill_server_api/README.md) — ハンドラ層（エラー・メッセージを生成）
- [usecase/README.md](../../usecase/README.md) — ユースケース層（GkillError を返却）
- [documents/reverse/error-handling-and-security.md](../../../../../documents/reverse/error-handling-and-security.md) — エラーハンドリング設計
