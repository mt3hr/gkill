# gkill_server_api - HTTP ハンドラ

## 概要

gorilla/mux を使用した HTTP API ハンドラ層。`GkillServerAPI` 構造体に全ハンドラメソッドを集約し、
認証ミドルウェア、レートリミット、アクセスログ、ローカルアクセス制御を提供する。
ビジネスロジックは `usecase/` 層に委譲し、本パッケージは HTTP リクエスト/レスポンスの変換に専念する。

## ディレクトリ構造

```
gkill_server_api/
├── gkill_server_api.go              # GkillServerAPI 構造体・初期化
├── gkill_server_api_address.go      # ルート表 apiRoutes()（パス・メソッド・認証区分・ハンドラの正本）と registerAPIRoutes
├── serve.go                         # gorilla/mux ルーター設定（ミドルウェア・PathPrefix 配信。API はルート表を登録するだけ）
├── close.go                         # グレースフルシャットダウン
├── auth.go                          # セッション・アカウント検証
├── auth_context.go                  # AuthContext 構造体・コンテキストキー
├── auth_middleware.go               # 認証ミドルウェアラッパー
├── recover_middleware.go            # リカバリミドルウェア
├── gzip_middleware.go               # /api/ 配下の応答を gzip 圧縮するミドルウェア
├── filter_local_only.go             # ローカルアクセス制限
├── write_response_status.go         # errors 配列から HTTP ステータスを決めて書く（エンコードより前に呼ぶ）
├── shared_file_authz.go             # 共有経路のファイル配信の認可（共有クエリ結果の許可集合と突き合わせ）
├── kftl_idempotency.go              # KFTL 送信の冪等キー台帳（TTL 付きインメモリ）
├── get_kyous_mcp_helpers.go         # get_kyous_mcp v2 の補助（複合カーソル・フィルタ・group_by）
├── utils.go                         # ユーティリティ関数（540+行）
├── web_push.go                      # Web Push 通知送信
├── gkill_server_api_access_log.go   # アクセスログ
├── gkill_server_api_rate_limit.go   # ログインレートリミット
├── plugin_content_html_cache.go     # プラグイン本文HTMLのキャッシュ（TTL・件数上限・singleflight）
├── handle_*.go                      # 各エンドポイントのハンドラ（実装98ファイル + テスト21ファイル）
└── *_test.go                        # テスト全31ファイル（handle_*_test.go 14本を含む。一覧は ABOUT_TEST.md）
```

**合計: 143ファイル**（基盤19 + ハンドラ実装98 + テスト31 + README.md 1 + ABOUT_TEST.md 1）
`.go` だけなら141ファイル。`handle_*.go` という名前のファイルは105あるが、うち14はテスト。

## GkillServerAPI 構造体

```go
type GkillServerAPI struct {
    server           *http.Server
    GkillDAOManager  *dao.GkillDAOManager
    FindFilter       *api.FindFilter
    UsecaseCtx       *usecase.UsecaseContext
    RebootServerCh   chan (struct{})

    deviceOnce sync.Once
    device     string
    deviceErr  error

    loginRateLimiter         *loginRateLimiter
    passwordResetRateLimiter *loginRateLimiter

    pluginContentHTMLCacheOnce sync.Once
    pluginContentHTMLCacheRef  *pluginContentHTMLCache

    closeOnce sync.Once
    closeErr  error
}
```

`device` は取得元の `GetAllServerConfigs` が重いSQLなので `sync.Once` でキャッシュする。設定更新時は `GkillServerAPI` ごと作り直されるので無効化は不要。

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

書き込み先ではないリポジトリを1本だけ読み込めない場合、利用可能なリポジトリで処理を続ける。
Web検索は既存の警告メッセージ、MCP検索は `warnings` で欠落を通知する。書き込み先または全体の取得失敗は
`ERR000018`、HTTP 500、Error ログへ伝播し、認証失敗や検索0件に見せない。

## セキュリティ機能

### ローカルアクセス制限（`filter_local_only.go`）

`ServerConfig.IsLocalOnlyAccess` が有効な場合、`localhost`、`127.0.0.1`、`[::1]`、`::1` からのアクセスのみ許可。
それ以外のリクエストには HTTP 403 を返す。**既定で有効**（初回起動ブロック `gkill_server_api.go` が
`server_config.DefaultIsLocalOnlyAccess` を引く。待受も `server_config.DefaultListenAddress` = `127.0.0.1:9999` で
ループバック限定）。LAN 公開はサーバ設定画面で両方を開く明示操作。既存の設定は移行しない（ADR-0708）。

### ログインレートリミット（`gkill_server_api_rate_limit.go`）

IP アドレス単位で 15 分間に 10 回までのログイン試行を許可。超過時はエラーを返す。
レートリミッタはログイン用（`loginRateLimiter`）とパスワードリセット用（`passwordResetRateLimiter`）の2本を持つ。後者は `handle_set_new_password.go` から使う。

### プラグイン本文HTMLキャッシュ（`plugin_content_html_cache.go`）

プラグインKyouの本文はgkillに保存されておらず、要求のたびにプラグインプロセスへ問い合わせる。プラグインの stdio は1本で呼び出しが直列化されるため、仮想スクロールで同じKyouのHTMLが繰り返し要求されると待ち行列ができる。`pluginContentHTMLCache` が userID + rep_name + kyou_id をキーに TTL 10分・最大2000件（挿入順に古いものから破棄）でキャッシュし、`singleflight` で同時要求を1回の問い合わせにまとめる。

### アクセスログ（`gkill_server_api_access_log.go`）

全リクエストのリモートアドレス、メソッド、パス、ステータスコード、処理時間、ユーザ ID を記録。
ポインタパターンにより、認証後にユーザ ID を後から設定可能。

## ハンドラパターン

全92ハンドラは共通のパターンに従う。`handle_add_kmemo.go` を例に:

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
| `handle_submit_kftl_text.go` | KFTL テキストのパース・実行（Web / Wear OS / MCP 共通の入口） |
| `handle_parse_kftl_text.go` | KFTL テキストの解析だけ（書かない）。おかしな行・付くタグ・板名を返す。Web のメモ帳のピンク表示と未知タグ・板名の確認が使う |

## フロントエンドルーティング

`serve.go` 内で Vue SPA のクライアントサイドルーティングに対応。
`/rykv`、`/kftl`、`/mi`、`/kyou`、`/saihate`、`/playing`、`/mkfl`、`/shared_*`、`/set_new_password`、`/register_first_account`（旧 `/regist_first_account` も残置）、`/` の各パスに対して、
埋め込み SPA の `index.html` を返す PathPrefix ルートを登録。

## ユーティリティ（`utils.go`）

540+ 行の汎用ユーティリティ関数群:
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
2. `handle_xxx.go` を新規作成（1ハンドラ1ファイル）
3. ビジネスロジックは `usecase/` 層に実装し、ハンドラから呼び出す
4. `gkill_server_api_address.go` のルート表 `apiRoutes()` に1行足す。認証区分は `Auth`（`authNone` / `authSession` / `authSessionRepos` = `wrapNoAuth` / `wrapAuth` / `wrapAuthRepos`）で選び、無認証でボディを読む経路は `Body` に上限（`bodyAuth` / `bodyUpload`）を付ける。`serve.go` やテストハーネスに `HandleFunc` を直に書かない（本番とテストで登録がずれる。ADR-0709）
5. `api_routes_test.go` の golden に認証区分を1行足す。表とハンドラ・doc コメントの突き合わせは同じテストが機械検査する

### doc コメントの方針

`HandleXxx` は **92/92 で doc コメント 100% を維持**する（`verify_docs` が網羅率を機械検査する）。
書式は「1行説明 / 空行 / パス・HTTPメソッド・認証区分 / req_res 型」。
ハンドラを追加したら doc コメントも必ず書くこと（書かないと `npm test` が落ちる）。

### 命名規則

- ハンドラファイル: `handle_xxx.go`（snake_case）
- ハンドラメソッド: `HandleXxx(w, r)` パターン
- ルート表の行: `{Path: "/api/xxx", Method: "POST", Auth: authXxx, Body: bodyXxx, Handler: g.HandleXxx},`

## 関連ドキュメント

- [api/README.md](../README.md) — API 共通基盤
- [usecase/README.md](../../usecase/README.md) — ユースケース層（ビジネスロジック委譲先）
- [req_res/README.md](../req_res/README.md) — Request/Response 構造体
- [message/README.md](../message/README.md) — エラーコード・メッセージコード
- [ABOUT_TEST.md](ABOUT_TEST.md) — テスト仕様
