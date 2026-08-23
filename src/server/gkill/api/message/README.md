# message - エラーコード・メッセージコード

## 概要

gkill API のエラーコードとメッセージコードの定数定義、およびレスポンス用構造体を提供する。
全 API レスポンスは `[]GkillMessage` と `[]GkillError` を含み、クライアント側でエラー判定・表示に使用される。

## ファイル一覧（5ファイル）

| ファイル | 説明 |
|---------|------|
| `gkill_error.go` | `GkillError` 構造体 — `ErrorCode` + `ErrorMessage` |
| `gkill_message.go` | `GkillMessage` 構造体 — `MessageCode` + `Message` |
| `error_codes.go` | エラーコード定数（411 定数: `ERR000001` 〜 `ERR000415`、`ERR000243` は欠番） |
| `message_codes.go` | メッセージコード定数（89 定数: `MSG000001` 〜 `MSG000089`） |
| `message_test.go` | コード形式・空文字チェックのテスト |

## レスポンス構造体

```go
type GkillError struct {
    ErrorCode    string `json:"error_code"`
    ErrorMessage string `json:"error_message"`
}

type GkillMessage struct {
    MessageCode string `json:"message_code"`
    Message     string `json:"message"`
}
```

## エラーコード体系（411 コード）

| コード範囲 | カテゴリ |
|-----------|---------|
| `ERR000001` 〜 `ERR000020` | アカウント・認証（ログイン、パスワードリセット等） |
| `ERR000023` 〜 `ERR000095` | CRUD 操作（Add, Update, Get, NotFound — 全データ型） |
| `ERR000098` 〜 `ERR000199` | 複合操作（共有リスト、MiSharedTasks、GPSLog 等） |
| `ERR000203` 〜 `ERR000280` | 拡張機能（KFTL、通知、構造体、Danote） |
| `ERR000300` 〜 `ERR000407` | 新機能（KC 数値記録、トランザクション、ZIP ブラウズ、レートリミット、プラグイン、IDF パス解決、ReKyou/MiReKyou の対象ID逆引き） |
| `ERR000408` 〜 `ERR000409` | アカウント運用の防護（リセットトークンの期限切れ、自分自身の無効化の拒否） |
| `ERR000410` | 検索そのものの失敗（repのSQLエラーなど、個別のGkillErrorが立たない失敗の受け皿） |
| `ERR000411`, `ERR000412` | MCP向けrep一覧（get_rep_infos_mcp）のリクエスト/レスポンス不正 |
| `ERR000413` 〜 `ERR000415` | 操作対象アカウント不在（404）、ローカル限定アクセス拒否（403）、panic回収（500） |

`ERR9000xx` 帯はフロントエンドだけで採番するコードで、Go 側の `error_codes.go` には存在しない（定義元は `src/client/classes/api/message/gkill_error.ts`、現在98定数）。番号が衝突しないよう帯を分けてあるので、Go 側でこの帯を使ってはならない。

## メッセージコード体系（87 コード）

| コード範囲 | カテゴリ |
|-----------|---------|
| `MSG000001` 〜 `MSG000004` | 認証（ログイン、ログアウト、パスワードリセット） |
| `MSG000005` 〜 `MSG000024` | CRUD 成功（Add, Update 系） |
| `MSG000025` 〜 `MSG000040` | Get 操作成功 |
| `MSG000041` 〜 `MSG000074` | 設定・アップロード・KC・トランザクション |
| `MSG000075` 〜 `MSG000089` | KFTL、MCP、キャッシュ、通知、ZIP ブラウズ、ReKyou/MiReKyou の対象ID逆引き、プラグイン検索失敗の警告、MCP向けrep一覧 |

## HTTP ステータスコードとの関係

**エラーコードごとの対応表が `http_status.go` にある**（`errorCodeHTTPStatus`）。
`HTTPStatusOf(errorCode)` で引ける。`errors` が空なら 200。

| HTTP ステータス | 割り当てているエラーコード | 件数 |
|----------------|--------------------------|------|
| 200 | （`errors` が空のとき） | — |
| 400 | `Invalid*RequestDataError` ほか入力値のバリデーション失敗 | 103 |
| 401 | `AccountSessionNotFoundError` / `AccountSessionExpiredError` / `AccountInvalidPasswordError` / `AccountNotFoundError` | 4 |
| 403 | `AccountNotHasAdminError` / `AccountDisabledError` / `LocalOnlyAccessDeniedError` ほか | 12 |
| 404 | `NotFound*` / `TargetAccountNotFoundError` | 18 |
| 409 | `AlreadyExist*` / `AccountPasswordResetTokenIsNotNilError` | 16 |
| 429 | `LoginRateLimitError` | 1 |
| 500 | `Get*` / `Add*` / `Update*` / `Delete*` / `CommitTx*` / `Invalid*ResponseDataError` ほか | 260 |

**エラーコードを足したら表にも1行足すこと。** `http_status_test.go` が
`error_codes.go` をソース走査して未分類のコードを落とす。
名前の規則からは導けない（`Invalid*` が 400 と 500 に、`NotFound*` が 401 と 404 に跨る）ので、
推論に置き換えないこと。経緯と却下案は
[ADR-0045](../../../../../documents/adr/0045-http-status-from-error-code.md)。

## 関連ドキュメント

- [gkill_server_api/README.md](../gkill_server_api/README.md) — ハンドラ層（エラー・メッセージを生成）
- [usecase/README.md](../../usecase/README.md) — ユースケース層（GkillError を返却）
- [documents/reverse/error-handling-and-security.md](../../../../../documents/reverse/error-handling-and-security.md) — エラーハンドリング設計
