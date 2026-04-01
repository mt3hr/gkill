# gkill_server_api - HTTP ハンドラ

## 概要

gorilla/mux を使用した HTTP API ハンドラ層。`GkillServerAPI` 構造体に全ハンドラメソッドを集約し、
認証ミドルウェア、レートリミット、アクセスログ、ローカルアクセス制御を提供する。
ビジネスロジックは `usecase/` 層に委譲し、本パッケージは HTTP リクエスト/レスポンスの変換に専念する。

## ディレクトリ構造

```
gkill_server_api/
├── gkill_server_api.go              # GkillServerAPI 構造体・初期化
├── gkill_server_api_address.go      # 全エンドポイントの URL パス定義
├── serve.go                         # gorilla/mux ルーター設定・全78ルート登録
├── close.go                         # グレースフルシャットダウン
├── auth.go                          # セッション・アカウント検証
├── auth_context.go                  # AuthContext 構造体・コンテキストキー
├── auth_middleware.go               # 認証ミドルウェアラッパー
├── filter_local_only.go             # ローカルアクセス制限
├── utils.go                         # ユーティリティ関数（490+行）
├── web_push.go                      # Web Push 通知送信
├── gkill_server_api_access_log.go   # アクセスログ
├── gkill_server_api_rate_limit.go   # ログインレートリミット
├── gkill_server_api_test.go         # 統合テスト
├── gkill_server_api_rate_limit_test.go # レートリミットテスト
└── handle_*.go                      # 各エンドポイントのハンドラ（79ファイル）
```

**合計: 94ファイル**（基盤12 + ハンドラ79 + テスト2 + ABOUT_TEST.md 1）

## GkillServerAPI 構造体

```go
type GkillServerAPI struct {
    server           *http.Server
    APIAddress       *GkillServerAPIAddress
    GkillDAOManager  *dao.GkillDAOManager
    FindFilter       *api.FindFilter
    UsecaseCtx       *usecase.UsecaseContext
    RebootServerCh   chan (struct{})
    device           string
    loginRateLimiter *loginRateLimiter
    closeOnce        sync.Once
    closeErr         error
}
```

`NewGkillServerAPI()` で初期化。初回起動時に admin アカウント（パスワードなし）を自動作成し、
VAPID 鍵を含むサーバ設定とアプリケーション設定を初期化する。

## 認証・認可

### ミドルウェア3段階

| ラッパー関数 | 認証 | リポジトリ | 用途 |
|-------------|------|-----------|------|
| `wrapNoAuth` | なし | なし | ログイン、ログアウト、パスワードリセット、共有ページ等 |
| `wrapAuth` | セッション必須 | なし | 設定取得・更新、アカウント管理、通知登録等 |
| `wrapAuthRepos` | セッション必須 | 読み込み | 全 CRUD 操作（データ追加・更新・取得・削除） |

全ラッパーにローカルアクセスフィルタ（`filter_local_only.go`）が適用される。

### 認証フロー

1. リクエスト JSON から `session_id` と `locale_name` を抽出
2. セッション検証: 有効期限チェック、アプリケーション名照合（"gkill"）
3. アカウント検証: 存在チェック、有効状態チェック
4. デバイス情報取得
5. `wrapAuthRepos` の場合: `GkillDAOManager.GetRepositories(userID, device)` でリポジトリ読み込み
6. `AuthContext` を `context.Context` に格納

## セキュリティ機能

### ローカルアクセス制限（`filter_local_only.go`）

`ServerConfig.IsLocalOnlyAccess` が有効な場合、`localhost`、`127.0.0.1`、`[::1]`、`::1` からのアクセスのみ許可。
それ以外のリクエストには HTTP 403 を返す。

### ログインレートリミット（`gkill_server_api_rate_limit.go`）

IP アドレス単位で 15 分間に 10 回までのログイン試行を許可。超過時はエラーを返す。

### アクセスログ（`gkill_server_api_access_log.go`）

全リクエストのリモートアドレス、メソッド、パス、ステータスコード、処理時間、ユーザ ID を記録。
ポインタパターンにより、認証後にユーザ ID を後から設定可能。

## ハンドラパターン

全79ハンドラは共通のパターンに従う。`handle_add_kmemo.go` を例に:

```
1. Content-Type: application/json 設定
2. req_res パッケージから Request/Response 構造体を生成
3. defer で Request Body のクローズと Response の JSON エンコードを設定
4. JSON リクエストをデコード
5. AuthFromContext(r.Context()) で認証情報を取得
6. UsecaseCtx.AddKmemo(...) でユースケース層を呼び出し
7. レスポンスに Messages と Errors を設定して返却
```

### 特殊ハンドラ

| ファイル | 説明 |
|---------|------|
| `handle_browse_zip_contents.go` | ZIP ファイル内容のブラウジング。パストラバーサル防止、Shift_JIS→UTF-8 変換、singleflight 重複防止 |
| `handle_file_serve.go` | `/files/` パスでリポジトリ内ファイルを配信 |
| `handle_get_kyous_mcp.go` | MCP サーバ向け専用 Kyou 取得エンドポイント |
| `handle_submit_kftl_text.go` | KFTL テキストのパース・実行 |

## フロントエンドルーティング

`serve.go` 内で Vue SPA のクライアントサイドルーティングに対応。
`/rykv`、`/kftl`、`/mi`、`/kyou`、`/saihate`、`/plaing`、`/mkfl`、`/shared_*`、`/set_new_password`、`/regist_first_account`、`/` の各パスに対して、
埋め込み SPA の `index.html` を返す PathPrefix ルートを登録。

## ユーティリティ（`utils.go`）

490+ 行の汎用ユーティリティ関数群:
- ファイル操作（サムネイル・動画キャッシュ、GPS ログインポート）
- ID 生成
- GPX エクスポート
- TLS 証明書ファイル生成
- IP アドレス検出
- ユーザリポジトリ初期化
- Windows ファイルダイアログ連携

## 開発ガイドライン

### 新しいハンドラの追加方法

1. `req_res/` に Request/Response 構造体を追加
2. `gkill_server_api_address.go` にアドレス定数を追加
3. `handle_xxx.go` を新規作成（1ハンドラ1ファイル）
4. ビジネスロジックは `usecase/` 層に実装し、ハンドラから呼び出す
5. `serve.go` の `resetRouter()` 内で `router.HandleFunc()` を登録
6. 適切なミドルウェアラッパー（`wrapNoAuth`/`wrapAuth`/`wrapAuthRepos`）を選択

### 命名規則

- ハンドラファイル: `handle_xxx.go`（snake_case）
- ハンドラメソッド: `HandleXxx(w, r)` パターン
- アドレス定数: `XxxAddress` パターン

## 関連ドキュメント

- [api/README.md](../README.md) — API 共通基盤
- [usecase/README.md](../../usecase/README.md) — ユースケース層（ビジネスロジック委譲先）
- [req_res/README.md](../req_res/README.md) — Request/Response 構造体
- [message/README.md](../message/README.md) — エラーコード・メッセージコード
- [ABOUT_TEST.md](ABOUT_TEST.md) — テスト仕様
