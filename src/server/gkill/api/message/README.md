# message - エラーコード・メッセージコード

## 概要

gkill API のエラーコードとメッセージコードの定数定義、およびレスポンス用構造体を提供する。
全 API レスポンスは `[]GkillMessage` と `[]GkillError` を含み、クライアント側でエラー判定・表示に使用される。

## ファイル一覧（5ファイル）

| ファイル | 説明 |
|---------|------|
| `gkill_error.go` | `GkillError` 構造体 — `ErrorCode` + `ErrorMessage` |
| `gkill_message.go` | `GkillMessage` 構造体 — `MessageCode` + `Message` |
| `error_codes.go` | エラーコード定数（376 定数: `ERR000001` 〜 `ERR000376`） |
| `message_codes.go` | メッセージコード定数（86 定数: `MSG000001` 〜 `MSG000080`） |
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

## エラーコード体系（376 コード）

| コード範囲 | カテゴリ |
|-----------|---------|
| `ERR000001` 〜 `ERR000020` | アカウント・認証（ログイン、パスワードリセット等） |
| `ERR000023` 〜 `ERR000095` | CRUD 操作（Add, Update, Get, NotFound — 全データ型） |
| `ERR000098` 〜 `ERR000199` | 複合操作（共有リスト、MiSharedTasks、GPSLog 等） |
| `ERR000203` 〜 `ERR000280` | 拡張機能（KFTL、通知、構造体、Danote） |
| `ERR000300` 〜 `ERR000376` | 新機能（KC 数値記録、トランザクション、ZIP ブラウズ、レートリミット） |

## メッセージコード体系（86 コード）

| コード範囲 | カテゴリ |
|-----------|---------|
| `MSG000001` 〜 `MSG000004` | 認証（ログイン、ログアウト、パスワードリセット） |
| `MSG000005` 〜 `MSG000024` | CRUD 成功（Add, Update 系） |
| `MSG000025` 〜 `MSG000040` | Get 操作成功 |
| `MSG000041` 〜 `MSG000074` | 設定・アップロード・KC・トランザクション |
| `MSG000075` 〜 `MSG000080` | KFTL、MCP、キャッシュ、通知、ZIP ブラウズ |

## HTTP ステータスコードとの関係

| HTTP ステータス | 意味 |
|----------------|------|
| 200 | 正常レスポンス（`errors` 配列にエラーが含まれる場合もある — 要チェック） |
| 403 | アクセス拒否 |
| 500 | 予期しないエラー |

## 関連ドキュメント

- [gkill_server_api/README.md](../gkill_server_api/README.md) — ハンドラ層（エラー・メッセージを生成）
- [usecase/README.md](../../usecase/README.md) — ユースケース層（GkillError を返却）
- [documents/reverse/error-handling-and-security.md](../../../../documents/reverse/error-handling-and-security.md) — エラーハンドリング設計
