# gkill MCP サーバ（`gkill_server mcp`）

> **セットアップ手順**: Claude（Claude.ai / Claude Code / Claude Desktop）からの接続手順は [MCP セットアップガイド](../../../../documents/reverse/mcp-setup-guide.md) を参照してください。

gkill の API を MCP サーバとして公開します。MCP サーバは **`gkill_server` のサブコマンド**で、
起動中の gkill_server への HTTP クライアントとして動きます（`update_cache` / `add_tag` と同じ型。DB は直接読みません）。
3 種類のサーバを提供します:

| サーバー | 起動コマンド | ツール数 | デフォルトポート | 用途 |
|---|---|---|---|---|
| **Read専用** | `gkill_server mcp --kind read` | 12 (11 read + 1 plugin) | 8808 | 読み取りのみ |
| **Write専用** | `gkill_server mcp --kind write` | 29 (21 write + 7 read convenience + 1 plugin) | 8809 | 書き込み中心 |
| **Read/Write統合** | `gkill_server mcp --kind readwrite` | 33 (11 read + 21 write + 1 plugin) | 8810 | 全機能 |

プラグインツール `gkill_get_plugin_list` は3サーバ共通で提供します（読み取り専用）。プラグインKyouの本文は `gkill_get_kyous` の `include_plugin_content` でレスポンスに埋め込みます。

`gkill_get_kyous` の利用時は、`partial` だけでなく `warnings` も毎回確認してください。
`partial=false` でも、一部の記録保管場所を読み込めず、利用可能な場所だけの結果になっている場合があります。
警告された名前を `query.reps` に指定し直すと、利用可能な場所まで検索対象から外れて0件に見えるため、
復旧するまではその名前を検索条件に入れません。この警告は通常検索・`count_only`・`group_by` のどれでも同じです。

2026-09-20 に Node.js 実装（旧 `src/mcp`）を Go へ移しました。tools/list と tools/call の応答は旧実装とバイト単位で一致します
（`testdata/golden/` に旧実装から採取したゴールデンがあり、`golden_test.go` が固定します。経緯は [ADR-0631](../../../../documents/adr/0631-mcp-lives-in-gkill-server.md)）。

### ファイル構成

package `mcp`（`src/server/gkill/mcp/`）は旧実装の `lib/*.mjs` と 1:1 に対応します。サブコマンドの配線は `src/server/gkill/main/common/mcp.go`。

| ファイル | 共有範囲 | 内容 |
|---|---|---|
| `server_base.go` | 3サーバ | JSON-RPC の受け口（`initialize` / `tools/list` / `tools/call` の骨組み） |
| `stdio_transport.go` | 3サーバ | stdio の JSON-RPC トランスポート（入力は NDJSON と Content-Length の両対応、出力は NDJSON） |
| `http_transport.go` | 3サーバ | Streamable HTTP トランスポート（OAuth 2.1 の配線を含む） |
| `gkill_client.go` | 3サーバ | gkill 本体を叩く HTTP クライアント（ログイン・認証リトライ・ファイル取得） |
| `payload.go` | 3サーバ | レスポンスのペイロード加工 |
| `read_tools.go` | read / readwrite | 読み取りツールの定義。**書き込み専用サーバも rep名 / 板名 / タグ名などの便利ツールをここから取る** |
| `write_tools.go` | write / readwrite | 書き込みツールの定義 |
| `plugin_tools.go` | 3サーバ | プラグインツールと、本文のインライン埋め込み |
| `find_query_schema.go` | read / readwrite | `gkill_get_kyous` の検索条件スキーマ |
| `normalization.go` / `write_normalization.go` / `validation.go` | — | 入力の正規化と検証 |
| `oauth_server.go` / `oauth_store.go` / `oauth_html.go` / `pkce.go` | HTTPモード | OAuth 2.1 |
| `file_link_store.go` | HTTPモード | 期限付きファイルリンク（`GET /files/{token}`） |
| `html_text.go` | 3サーバ | プラグインのコンテンツHTML → プレーンテキスト |
| `access_log.go` / `errors.go` / `constants.go` | 3サーバ | gkill_log 上のロガー・エラー型・定数 |
| `status_tool.go` | 3サーバ | ツール一覧の世代 `schema_revision` の計算と `gkill_status` への焼き込み |
| `tool_schema_budget.go` / `tool_schema_budget.json` | 3サーバ | tools/list のバイト量の計測と予算（`gkill_server mcp schema-budget --update` で更新） |
| `bootstrap.go` / `config.go` | 3サーバ | 起動ブロック（`Start`）と設定ファイルの解決 |
| `jsonobj/` | — | 順序つき JSON（JavaScript の `JSON.stringify` 互換の直列化。応答のバイト一致のため） |

> ツール数（上の表の 12 / 29 / 33）は `verify_docs` が `*_tools.go` の定義を辿って
> 実測と突き合わせます。サーバ本体だけを見ても数えられないので、ツールを増やすときは
> 必ず `read_tools.go` / `write_tools.go` / `plugin_tools.go` の配列へ足してください。

2つのトランスポートモードに対応：
- **stdio** (デフォルト): Claude Desktop等のローカルMCPクライアント向け
- **HTTP** (Streamable HTTP): Claude.ai、ChatGPT等のリモートMCPクライアント向け

### 設定ファイル

`$GKILL_HOME/configs/gkill_mcp.json`。**無ければ初回起動時に既定値で生成**（0600）し、`config_created` をログに残します。
既存のファイルは書き換えません。壊れていれば起動を止めます（黙って既定へは落ちません）。`--config <path>` で場所を変えられます。

```jsonc
{
  "gkill": { "base_url": "http://127.0.0.1:9999", "insecure": false, "locale": "ja",
             "fetch_timeout_ms": 120000, "user": "", "password_sha256": "", "session_id": "" },
  "log_level": "access", "max_file_bytes": 8388608, "file_link_ttl_ms": 3600000,
  "transport": "stdio", "bind_addr": "0.0.0.0",
  "servers": { "read": { "port": 8808, "oauth_issuer": "" },
               "write": { "port": 8809, "oauth_issuer": "" },
               "readwrite": { "port": 8810, "oauth_issuer": "" } }
}
```

優先順位は **フラグ（`--transport`、親の `--log`） > 環境変数 > ファイル > 既定値**。旧実装と同じ環境変数（下記）はそのまま効きます。
平文パスワード `GKILL_PASSWORD` は環境変数からだけ受け、ファイルには置きません。

### stdioモード（デフォルト）
1. gkill_server を起動（例: `http://127.0.0.1:9999`）
2. 環境変数か設定ファイルで接続先と資格情報を設定
3. MCP サーバを起動

```powershell
$env:GKILL_BASE_URL="http://127.0.0.1:9999"
$env:GKILL_USER="admin"
$env:GKILL_PASSWORD_SHA256="<sha256 hex>"  # または GKILL_PASSWORD
gkill_server mcp --kind read
```

stdio では **stdout が JSON-RPC の専用チャネル**です。ログは `$GKILL_HOME/logs/gkill_mcp_<kind>*.log` へだけ出て、stdout へは1バイトも書きません。

### HTTPモード（リモート接続）
OAuth 2.1認証付きのHTTPサーバとして起動します。Claude.ai、ChatGPT等のMCPクライアントからOAuth認証で接続できます。

```powershell
$env:GKILL_BASE_URL="http://127.0.0.1:9999"
$env:GKILL_USER="admin"
$env:GKILL_PASSWORD_SHA256="<sha256 hex>"
$env:MCP_OAUTH_ISSUER="https://<your-host>"  # リモート接続時は必須（クライアントがアクセス可能な公開URL）
gkill_server mcp --kind read --transport http
```

```bash
# Linux/macOS
GKILL_BASE_URL=http://127.0.0.1:9999 \
GKILL_USER=admin \
GKILL_PASSWORD_SHA256="<sha256 hex>" \
MCP_OAUTH_ISSUER="https://<your-host>" \
MCP_PORT=8808 \
gkill_server mcp --kind read --transport http
```

> **重要**: `MCP_OAUTH_ISSUER` はリモート接続時に必須です。未設定の場合 `http://localhost:<port>` がデフォルトになり、Claude.ai/ChatGPT等のクラウドサービスからOAuthエンドポイントに到達できません。Cloudflare Tunnel等を使う場合は公開URL（例: `https://example.com`）を設定してください。

エンドポイント: `POST /mcp`（Streamable HTTP仕様準拠）

認証: OAuth 2.1 (Authorization Code + PKCE)。MCPクライアントが自動的にOAuthエンドポイントを検出し、ユーザーに認証を要求します。

#### Claude.ai Connectorsでの設定
1. HTTPモードで起動（グローバルIPまたはトンネル経由でアクセス可能にする）
2. Claude.ai → Settings → Connectors → カスタムコネクタを追加
3. リモートMCPサーバーURL: `http://<your-host>:8808/mcp`
4. OAuth Client ID / シークレット: 空欄のまま（DCRで自動登録）
5. 「追加」→ ログイン画面でgkillのユーザーID・パスワードを入力

#### ChatGPTでの設定

> **既知の制限（2026-03時点）**: ChatGPTのMCP統合はベータ版であり、OAuth認証・初回のデータ取得は成功するものの、**cursorを使ったページング継続時にChatGPTプラットフォーム側でMCPリソース参照が失われ「Resource not found」エラーが発生する**不具合が確認されています。この問題はgkill側ではなくChatGPTプラットフォーム内部の問題であり、ページングリクエストがgkillサーバーに到達しません。1ページに収まる小規模クエリは正常に動作します。

1. ChatGPT → Settings → MCP → 新しいアプリ
2. MCPサーバーのURL: `http://<your-host>:8808/mcp`
3. 認証: 「OAuth」を選択
4. 高度な設定 → OAuthエンドポイントが自動検出される
5. 「作成する」→ ログイン画面でgkillのユーザーID・パスワードを入力

### OAuth 2.1 認証

HTTPモードではOAuth 2.1が常に有効です。MCP仕様に準拠し、ChatGPTとClaude.aiの両方で動作します。

#### OAuthエンドポイント
| エンドポイント | メソッド | 説明 |
|---|---|---|
| `/.well-known/oauth-protected-resource` | GET | Protected Resource Metadata (RFC 9728) |
| `/.well-known/oauth-authorization-server` | GET | OAuthサーバーメタデータ (RFC 8414) |
| `/oauth/authorize` (`/authorize`) | GET/POST | 認可エンドポイント（ログインフォーム表示・認証） |
| `/oauth/token` (`/token`) | POST | トークンエンドポイント（コード交換・リフレッシュ） |
| `/oauth/register` (`/register`) | POST | 動的クライアント登録 (RFC 7591) |

※ 括弧内はClaude.aiフォールバック用の短縮パス（既知のバグ対応）

#### 認証フロー
1. MCPクライアントが `POST /mcp` を送信 → 401 + `WWW-Authenticate` ヘッダーで検出情報取得
2. `/.well-known/oauth-protected-resource` から認可サーバーURLを取得
3. `/.well-known/oauth-authorization-server` からOAuthメタデータを取得
4. `/oauth/register` で動的クライアント登録（DCR）
5. `/oauth/authorize` にリダイレクト（PKCE + `resource` パラメータ付き）
6. ユーザーがgkillログイン画面で認証 → 認可コード発行
7. `/oauth/token` でコード交換 → アクセストークン取得
8. 以降は `Authorization: Bearer <access_token>` でMCPエンドポイントにアクセス

#### トークン仕様
- アクセストークン有効期間: 1時間（インメモリ、再起動で消失→リフレッシュトークンで再発行）
- リフレッシュトークン有効期間: 30日（ローテーション方式）
- リフレッシュトークンとDCRクライアント登録は `$GKILL_HOME/configs/mcp_oauth_read_state.json` に自動永続化（書式は旧 Node 実装と互換。移行時に再認証は不要）。サーバー再起動後も再認証不要

#### 動作確認（curl）
```bash
# Protected Resource Metadata
curl http://localhost:8808/.well-known/oauth-protected-resource

# OAuth Authorization Server Metadata
curl http://localhost:8808/.well-known/oauth-authorization-server

# 動的クライアント登録
curl -X POST http://localhost:8808/oauth/register \
  -H "Content-Type: application/json" \
  -d '{"redirect_uris": ["http://localhost/callback"], "client_name": "My App"}'

# MCP (未認証 → 401 + WWW-Authenticate)
curl -v -X POST http://localhost:8808/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

### 主な環境変数（設定ファイルより優先）

#### gkill接続
- `GKILL_BASE_URL` (default: `http://127.0.0.1:9999`)
- `GKILL_USER`
- `GKILL_PASSWORD_SHA256` または `GKILL_PASSWORD`
- `GKILL_SESSION_ID` (任意。指定時はログインをスキップ)
- `GKILL_LOCALE` (default: `ja`)
- `GKILL_INSECURE` — `true` or `1` でgkill_serverへの接続時にTLS証明書検証をスキップ（自己署名証明書用）
- `GKILL_FETCH_TIMEOUT_MS` (default: `120000`)
- `GKILL_MCP_MAX_FILE_BYTES` (default: `8388608` = 8MB) — `gkill_get_idf_file`（base64）で返せる最大バイト数
- `GKILL_MCP_FILE_LINK_TTL_MS` (default: `3600000` = 1時間) — HTTPモードで発行するファイルURLトークンの有効期限

#### トランスポート
- `MCP_TRANSPORT` (default: `stdio`) — `stdio` or `http`（`--transport` が最優先）
- `MCP_PORT` (default: 種別ごとの既定ポート) — HTTPサーバのポート番号
- `MCP_BIND_ADDR` (default: `0.0.0.0`) — 待ち受けアドレス。トンネル・リバースプロキシの背後では `127.0.0.1` を推奨
- `MCP_OAUTH_ISSUER` (default: `http://localhost:<MCP_PORT>`) — OAuthメタデータのissuer URL。**リモート接続時は必須**。クライアントがアクセス可能な公開URL（例: `https://example.com`）を設定。未設定だとClaude.ai/ChatGPTからOAuth認証が失敗する。HTTPモードのファイルURL（`file_url`）もこのURLを基点に組み立てられ、`GET /files/{token}` で配信される
- `MCP_LOG` (default: `access`) — ログレベル（`none` / `error` / `warn` / `info` / `access` / `debug` / `trace` / `trace_sql`。gkill_server と同じ語彙）。未知の値は起動を止める
- `GKILL_HOME` (default: `$HOME/gkill`) — `--gkill_home_dir` が明示されていないときの gkill ホーム

### ログ

gkill_log を使い、`$GKILL_HOME/logs/` に `gkill_mcp_<kind>.log`（全レベル統合）と `gkill_mcp_<kind>_{error,warn,info,access,debug,trace,trace_sql}.log`（レベル別）を開きます。
静的フィールドは `app=gkill_mcp` と `kind=<kind>`。`http_request` と `tool_call` は ACCESS、`server_start` は INFO、`token_rejected` / `auth_failure` は WARN、`http_handler_error` / `file_fetch_error` は ERROR。
回転は `--log_rotate_max_bytes` / `--log_rotate_keep` がそのまま効きます。`http_request` のパスはクエリを落として記録します（認可フローの秘匿値をログへ残さない）。

### 提供ツール

#### Readツール（11 — Read専用/ReadWrite統合サーバで使用可能）
| ツール名 | 説明 |
|---|---|
| `gkill_status` | このサーバが何者かを返す（引数なし。3サーバ共通）: `server_kind`（read / write / readwrite）、接続先の `account.user_id` / `account.device`、gkill のビルド（`gkill.version` / `commit_hash` / `build_time`）、`transport`、`started_at` / `uptime_seconds`、`tool_count`、`schema_revision`。**`schema_revision` はツール一覧の世代**で、同じ値が `gkill_status` の説明文末尾にも焼き込まれている。応答と説明文の値が違えば、クライアントが握っている一覧が古い（一覧は接続時に1回しか取られない。サーバを再起動しても直らず、接続し直しが要る）。gkill へ届かないときも失敗にせず `gkill_reachable:false` + `gkill_error`（HTTP ステータスのみ）で返す |
| `gkill_get_mcp_help` | ツール説明の本文を topic ごとに返す（引数 `topic`: `search` / `pagination` / `mi` / `data_types` / `plugin` / `idf` / `deleted` / `rep` / `kftl` / `config`。省略で index。3サーバ共通、gkill へは往復しない）。**ツール一覧の説明文は要約**で、応答フィールドの一覧・Mi の射影・KFTL の文法全文などはここにある（正本は `help_topics.go`。ADR-0622） |
| `gkill_get_kyous` | Kyou一覧を取得（タグ・テキスト・型データをインライン返却）。`data_types` はエンティティ名 `timeis` / `mi` / `mirekyou` も受理して全射影へ展開する（ADR-0623）。`count_only` と `group_by` は cursor と同じく併用不可（エラー）。`query.ids` の不一致・`num_min` / `num_max` の種別混在は `warnings[]` に出る |
| `gkill_get_mi_board_list` | Miボード名一覧を取得 |
| `gkill_get_all_tag_names` | 全タグ名を取得 |
| `gkill_get_all_rep_names` | Kyouを供給するリポジトリ名を取得。`contains`（大小無視の部分一致）と `limit`（既定200）で絞れ、`total_count` / `truncated` が付く。rep が数百ある環境で「その名前があるか」を確かめるために全件を読まずに済む |
| `gkill_get_gps_log` | 期間指定でGPSログを取得。`limit` / `cursor`（不透明トークン。`next_cursor` をそのまま返す）でページングし、`count_only` / `group_by:"day"` で件数・日別カバレッジだけ取れる。**カーソルは `gkill_get_kyous` のものと別方式**（MCP 製 base64url。コーデックの正本は `gps_cursor.go` 1本で、発行側と受理側の両方がそこを使う） |
| `gkill_get_application_config` | アプリケーション設定を取得（タグ階層・ボード構造・テンプレート等）。`fields` で列を、`contains` で葉を絞り、`compact`（既定 true）が既定値の欄（`children` の null / 空・`is_dir:false`・`ignore_check_rep_rykv:false`・識別欄と同じ `name`）を落とす。`max_size_mb`（既定 0.25）を超える struct は `{omitted_bytes}` に置き換えて必ず収める（ADR-0629） |
| `gkill_get_rep_infos` | リポジトリ一覧を構造化メタデータ付きで取得。`query.rep_types` が受理する正準値 `canonical_rep_types[]`（表示ラベルと1:1でない）、索引付きrepの最終更新 `indexed_at`（古いと「追加したはずのファイルが検索に出ない」の原因）、タグ・テキスト・通知・GPSログの格納先 `attached_data_reps[]`（`query.reps` には渡せない。`use_to_write` 付き）を返す。列は `fields`、行は `writable_only` / `rep_types` / `rep_names` / `contains` / `data_kinds` で絞る（「`gkill_add_tag` はどこへ書くか」は `writable_only:true, data_kinds:["tag"]` で1行） |
| `gkill_get_idf_file` | IDFファイルの実データを取得（画像はMCP image blockで返却）。`thumb=WxH`（一辺最大1024、動画は `is_video: true` 併用）で縮小取得できる。上限は `GKILL_MCP_MAX_FILE_BYTES`（既定8MB） |
| `gkill_get_kyou_history` | 1件の全版を取得（削除済みの版も含む）。`gkill_get_kyous` から見えなくなった記録を読み返す唯一の経路。`limit`（既定20・上限200）と `offset` で頁を送り、`has_more` のとき `next_offset` が続きの位置（ADR-0626）。版の `data_type` はエンティティ名 |

##### ファイル実パス導線

ローカルのAIクライアントは、base64転送を経ずにファイルを直接読める。`gkill_get_kyous` のIDFペイロードには `file_path`（絶対パス）が入り、AIはそれをファイルシステムから直接読めばよい。画像はサイズ上限なくそのまま閲覧できる。

絶対パスは**同一マシンのクライアントにしか渡さない**。二重にゲートしている:

- **gkillサーバ側**: リクエスト元がlocalhostのときだけ `file_path` を返す。
- **MCPサーバ側**: stdioトランスポート（＝クライアントが同一マシン）のときだけAIに見せる。HTTP/OAuth接続では `file_path` を削ぎ落とす（代わりに後述の `file_url` を注入する）。

MCPサーバはHTTPモードでもgkillと同居しうるため、gkill側のlocalhost判定だけではリモートのAIにパスが渡ってしまう。MCPサーバ側のゲートが実質的な防御線になる。

##### リモート向けファイルURL導線（HTTPモード）

リモートのAIクライアント（クラウド上のChatGPT等）は実パスを読めないので、代わりに**MCPサーバ自身が発行する期限付き公開URL**を渡す。HTTPモードで動いているとき、`gkill_get_kyous` に `include_file_urls: true` を渡すと IDFペイロードに以下が入る:

- `file_url` — 画像は既定でサムネ（長辺〜1024のJPEG）、それ以外は原寸。
- `file_url_full` — 画像の原寸URL（画像のときだけ）。
- `file_url_expires_at` — 期限。

このURLは **Bearer無しでGET** できる（base64を経由せず、サイズ上限なし）。配信は各MCPサーバの `GET /files/{token}` ルートが担い、`MCP_OAUTH_ISSUER`（公開URL）を基点にURLを組み立てる。**ただしこのURLを取りに行くのはAI自身ではない**（後述の「AIが画像を見る経路」）。

セキュリティは**トークン自体**が担保する:

- トークンは特定の1ファイル(rep_name + file_name)に束縛され、期限付き（`GKILL_MCP_FILE_LINK_TTL_MS`、既定1時間）、暗号学的乱数で推測不能。
- gkillからのバイト取得は発行時のOAuthセッションで行い、**URLにセッションは載らない**。URLを知る者は「期限内・そのファイルだけ」取得できる。
- トークン発行はOAuth認証済みのツールコール内でのみ。gkill本体を公開する必要はない（gkillは非公開のまま、MCPサーバの公開面だけで完結）。

大きすぎて `gkill_get_idf_file`（base64、`GKILL_MCP_MAX_FILE_BYTES` 上限）に収まらないファイルも、この `file_url` なら渡せる。ただし次節のとおり**AIはその中身を見られない**（人間が開くリンクとして渡すことになる）。

##### AIが画像を「見る」経路

上の2つの導線は**優先順位ではなく用途の違い**。MCPはツール出力のURLを代わりに取りに行く仕組みを持たないし、クライアントが自前の機能でURLを取れたとしても、得られるのは会話の外側のバイトであって**見られる画像にはならない**。`file_url` は**人間へ渡すリンク**（返答に貼る・ブラウザで開く・HTMLへ埋める）と考える。

| クライアント | 画像を**見る** | ファイルを人間へ渡す |
|---|---|---|
| stdio（Claude Code等） | `file_path` を直接読む | `file_path` |
| HTTP（ChatGPT / Claude.ai Connectors） | **`gkill_get_idf_file` だけ** | `file_url` / `file_url_full` |

`type: "image"` のMCP contentブロックを組み立てるのは `server_base.go` の `BuildToolResult` 1箇所だけで、その入力は `gkill_get_idf_file` が返す `file_content_base64` に限られる。したがって**HTTP接続のAIが画像を見る手段は `gkill_get_idf_file` が唯一**で、画像生成ツールへ参照画像として渡す場合もこれを使う。

大きすぎて上限に当たる画像・動画は、`thumb`（`"1024x1024"` 等、一辺最大1024。動画は `is_video: true` を併用してフレームを抜く）を渡せば縮小JPEGで収まる。クエリの形は `file_url` の `?thumb=` と同一で、非対象のファイルは原本がそのまま返る。

実測（2026-08-24）: ChatGPTで「キーワード検索 → ヒットしたイラスト3枚を参照して新規イラストを生成」を1回実行したところ、`gkill_get_kyous` 1回に続いて `gkill_get_idf_file` が3回呼ばれ、`/files/` へのアクセスは0件だった。経緯と却下案は [ADR-0606](../../../../documents/adr/0606-idf-file-reaches-ai-through-payload.md)。

#### Writeツール（21 — Write専用/ReadWrite統合サーバで使用可能）
| ツール名 | 説明 |
|---|---|
| `gkill_add_kmemo` | テキストメモ作成 |
| `gkill_add_urlog` | ブックマーク/URL記録作成。サーバが対象サイトと faviconサービスへ外向き取得して空の title・description・favicon・サムネイルを埋める。`fetch_metadata: false` / `fetch_favicon: false` で項目別に抑止できる（両方 false なら外向き通信なし） |
| `gkill_add_nlog` | 支出/収入記録作成 |
| `gkill_add_lantana` | 気分記録作成（0-10） |
| `gkill_add_timeis` | 時間記録作成（開始/終了） |
| `gkill_add_mi` | タスク作成。未知の板名は新しい板を作る（既定）。`allow_create_board: false` で実在しない板名を typo として拒否できる（update_mi の板移動も同じ） |
| `gkill_add_kc` | 数値記録作成 |
| `gkill_add_tag` | 既存エントリにタグ追加 |
| `gkill_add_text` | 既存エントリにテキスト注釈追加 |
| `gkill_update_kmemo` | テキストメモ更新 |
| `gkill_update_urlog` | ブックマーク/URL記録更新 |
| `gkill_update_nlog` | 支出/収入記録更新 |
| `gkill_update_lantana` | 気分記録更新 |
| `gkill_update_timeis` | 時間記録更新 |
| `gkill_update_mi` | タスク更新 |
| `gkill_update_kc` | 数値記録更新 |
| `gkill_update_tag` | タグ更新 |
| `gkill_update_text` | テキスト注釈更新 |
| `gkill_submit_kftl` | KFTLテキスト一括処理。応答の `created[]`（`{id, data_type, updated, related_time}`。失敗時は `[]`、`related_time` は秒精度）に実際に書かれた記録が書かれた順で並ぶ。`replayed: true` は同じ `idempotency_key`・同じ本文の再送で、`created[]` は元の送信の控え（同じキーで別の本文は 409 `ERR000423`。ADR-0510）。`created[].id` を `gkill_add_tag` / `gkill_add_text` の `target_id` に使えば、KFTLで作った記録へ後からタグ・注釈を付けられる |
| `gkill_delete_kyou` | エントリのソフト削除。`gkill_submit_kftl` の `created[]` は `updated` / `related_time` を持つのでそのまま `targets` に渡せない —— `created.filter(c => !c.updated).map(({id, data_type}) => ({id, data_type}))` を渡す（`updated:true` は既存記録の更新で、消すと元から在った打刻が消える） |
| `gkill_restore_kyou` | ソフト削除の取り消し（`is_deleted` を戻す） |

Write専用サーバにはRead便利ツール7つ（`gkill_status`, `gkill_get_mcp_help`, `gkill_get_application_config`, `gkill_get_all_rep_names`, `gkill_get_mi_board_list`, `gkill_get_all_tag_names`, `gkill_get_kyou_history`）も含まれます。`gkill_status` / `gkill_get_application_config` は「どのアカウントへ書くのか」を書く前に確かめるためのものです。`gkill_get_kyou_history` を載せているのは、`gkill_delete_kyou` / `gkill_restore_kyou` と同じサーバから「いま何を消したのか」を確かめられないと、取り消しが当てずっぽうになるためです。

##### 更新系の引数

`gkill_update_*` は **patch セマンティクス**で、`id` 以外はすべて省略可能です。送らなかった項目は現在値のまま残ります。スキーマの `required` も `["id"]` だけです（以前は `title` などを要求しており、スキーマに従って推測値を送ると既存の値を静かに上書きしていました）。

`gkill_update_timeis` の `end_time` だけは3値です — 省略で現状維持、日時指定で終了、**`null` で終了時刻を消して進行中へ差し戻す**。省略が現状維持である以上、終了済み TimeIs を再開する手段は `end_time: null` だけです。

##### 削除と復活

削除は論理削除（`is_deleted=true` の版を追加）です。消したものは通常の検索から外れますが、
`gkill_get_kyou_history` で全版を読み返せ、`gkill_restore_kyou` で戻せます。
一覧したいときは `gkill_get_kyous` の `query.include_deleted_data: true` を使ってください
（既定は false で従来どおり除外。返る行は `is_deleted` で見分けられます）。
`query.only_latest_data: false` は受理されますが**無視されます** — 詳しくは
[ADR-0605](../../../../documents/adr/0605-mcp-version-history-is-a-dedicated-tool.md)。

#### プラグインツール（1つ — Read/Write/ReadWrite すべてのサーバで使用可能）
| ツール名 | 説明 |
|---|---|
| `gkill_get_plugin_list` | インストール済みプラグイン一覧を取得（name/version/description/data_type/rep_name/**emits_kyou**/**provides**/is_alive/process_running/**has_last_error**/typed_index/gps_index。診断文の中身（last_error / typed_index.last_build_error）は端末のディレクトリ構成を含むため返さない。[ADR-0707](../../../../documents/adr/0707-redact-environment-specific-strings.md)）。`emits_kyou:false` のプラグインは Kyou を1件も出さないので、その `data_type` / `rep_name` は**検索値ではない** —— `provides` に対応する経路（`gpslog` なら `gkill_get_gps_log`）から読む |

##### プラグイン内容取得の導線

プラグインKyou（Claude Code / Claude.ai / ChatGPT の会話ログ等）の**本文はgkillのDBに保存されていない**。プラグインプロセスが持っていて、コンテンツHTMLとして取り出すしかない。

かつては1件ずつ取る `gkill_get_plugin_content` ツールを用意していたが、N件読むのにツール呼び出しがN+1回必要になり、そのたびにLLMのターンを消費して非効率だったため廃止した。いまは `gkill_get_kyous` に **`include_plugin_content: true`** を渡すと、検索結果のプラグインペイロードに本文が直接入る。

導線は次の2ステップ:

1. `gkill_get_plugin_list` でプラグインの `data_type` / `rep_name` と役割（`emits_kyou` / `provides`）を把握する。`emits_kyou:false` ならこの導線には乗らない（Kyou が存在しないので `query.reps` に渡しても0件）。
2. `gkill_get_kyous` に `include_plugin_content: true` を付けて検索する（`query.reps` でそのプラグインに絞れる）。プラグインKyouの `payload` は `kind: "plugin"` で、`plugin_name` に加えて本文フィールドを持つ（`rep_name` / `id` は Kyou 側）。

各プラグインペイロードに付くフィールド:

| フィールド | 説明 |
|---|---|
| `content_status` | `ok` / `truncated` / `skipped` / `error`。**`ok` のときだけ本文が完全** |
| `content_text` | 変換後のテキスト（`plugin_content_format` が `text` / `both` のとき） |
| `content_html` | 生HTML（`plugin_content_format` が `html` / `both` のとき） |
| `content_skipped_reason` | `max_kyous` / `budget` / `deadline` / `rep_error`（`content_status` が `skipped` のとき） |
| `content_error` | 失敗理由（200文字で切る。`content_status` が `error` のとき） |

トップレベルには集計 `plugin_content: { requested, inlined, truncated, skipped, errors, total_text_length }` が付く。

`plugin_content_format` は `text`（既定）/ `html` / `both`:

- `text` — プラグインのコンテンツHTMLを `html_text.go` でプレーンテキストに変換する。プラグインのHTMLは表示用のCSS/JSでバイト数の大半が埋まっているため、既定でテキストにしている（`<script>` / `<style>` / コメントは中身ごと破棄、`<details>`/`<summary>` の中身は残す）。`plugin_content_max_text_length`（既定4000文字、最大200000）を超えたら切り詰め、`content_status: "truncated"` にする。
- `html` — 生HTMLをそのまま返す（表示やマークアップ自体が必要なとき）。
- `both` — 両方返す。

長い記録1件の全文が欲しいときは、`query.ids` でその1件に絞ったうえで `plugin_content_max_text_length` を上げる。

**上限と安全弁**: 1回のリクエストで本文を埋めるのは20件・合計200000文字まで、全体30秒で打ち切る。取得は `rep_name` ごとにグループ化し、**同一プラグインへは必ず1件ずつ直列**、プラグイン間は並列（既定4）で叩く。gkill側は1プラグイン1プロセスで呼び出しが直列化され（容量1のチャネル `callSlot`）、同時に投げても速くならず順番待ちが伸びるだけで、待ちきれなかったぶんは `ErrPluginBusy` になるからである。実行中のリクエストをabortしないのも同じ配慮。あるプラグインで1件失敗したら、そのプラグインの残りは投げずに `rep_error` として諦める。

内部で叩くgkill APIは `/api/get_plugin_list` と `/api/get_plugin_content_html`。プラグイン設定の書き換え（`/api/post_plugin_config`）はMCPに公開していない。

> Write専用サーバには `gkill_get_kyous` が無いため、プラグイン本文を読む導線も無い。本文が要るときは ReadWrite サーバを使う。

### AI用運用ガイド（MCP）
AIが安定して呼び出せるよう、以下のルールを推奨します。

#### 1) ドメイン用語の簡易辞書
- `kyou`: 記録の親データ（一覧検索の主対象）
- `kmemo`: テキストメモ
- `kc`: 数値記録
- `urlog`: URL/ブックマーク記録
- `nlog`: 支出/収入記録
- `timeis`: 打刻（開始/終了）
- `mi`: タスク記録
- `lantana`: 気分値記録
- `idf`: ファイル関連記録
- `git_commit_log`: Gitコミット記録

#### 2) ツール選択フロー（推奨）
1. まず `gkill_status`（引数なし）を呼び、**どのアカウントに接続しているか**（`account.user_id` / `account.device`）と、**握っているツール一覧が古くないか**（応答の `schema_revision` と `gkill_status` の説明文末尾の値が一致するか）を確かめる（read / write / readwrite が別アカウントを向いていることがある。`gkill_get_application_config` を `fields: ["user_id", "device"]` で呼んでも接続先は分かる）。タグ階層・ボード構造が要るときだけ `fields: ["tag_struct"]` 等を追加で取る（無指定の全量は実測94KBある）
2. 必要に応じて `gkill_get_all_tag_names` / `gkill_get_all_rep_names` / `gkill_get_mi_board_list` でメタ情報を補完
3. `query.rep_types` で絞るときは `gkill_get_rep_infos` で正準値（`canonical_rep_types`）を引く（ApplicationConfig の表示ラベルと受理値は1:1でない）。「追加したはずのファイルが検索に出ない」ときも `indexed_at` で索引の鮮度を確かめる
4. `gkill_get_kyous` でKyou一覧を取得（タグ・テキスト・型データはレスポンスにインライン）
5. 件数が多い場合は `cursor` / `next_cursor` でページングして追加取得
6. 地図系は `gkill_get_gps_log` を使う
7. プラグイン由来のKyou（`payload.kind === "plugin"`）の本文が要るときは `gkill_get_kyous` に `include_plugin_content: true` を付ける。どのプラグインが入っているかは `gkill_get_plugin_list` で分かる
8. 検索から消えた記録・過去版・削除済みの中身を読み返すときは `gkill_get_kyou_history`（`id` と `data_type` の両方が必須）。削除の取り消しは Write系サーバの `gkill_restore_kyou`

#### 3) `gkill_get_kyous` のパラメータ
| パラメータ | 型 | 説明 |
|---|---|---|
| `query` | object | FindQuery（後述） |
| `locale_name` | string | ロケール（例: ja, en） |
| `limit` | integer | 最大取得件数（default: 20） |
| `cursor` | string | 前回レスポンスの `next_cursor` をそのまま指定してページング（不透明文字列。v2は複合形式 `{RFC3339Nano}::{ID}`。組み立て・編集しない） |
| `max_size_mb` | number | `kyous[]` の最大サイズMB（default: 0.25。厳密上限。`include_plugin_content` で本文を埋め込んだ後にも守り直し、押し出した分は次頁へ。ADR-0624） |
| `is_include_timeis` | boolean | 各Kyouに付随する TimeIs を含めるか（default: false） |
| `include_attached_ids` | boolean | `tag_entities[]` / `text_entities[]`（注釈自身の id）を載せるか（default: false。ADR-0629） |
| `include_file_urls` | boolean | HTTP 接続のとき idf の公開URL（`file_url` / `file_url_full` / `file_url_expires_at`）を発行するか（default: false。ADR-0630） |
| `count_only` / `group_by` | boolean / string | 件数だけ・バケット集計（month/day/week_of_day/hour/data_type/rep_name/create_app/update_app/url_domain/file_extension）。cursor とは併用不可 |
| `data_types` / `num_min` / `num_max` / `idf_kinds` / `include_file_size` | - | リクエストレベルの絞り込み（v2。ADR-0604） |
| `create_apps` / `update_apps` | array | 作成アプリ / 最終更新アプリの許可リスト（各記録の `create_app` / `update_app` と照合）。「MCP経由で作った記録」（`"gkill_mcp_readwrite"` / `"gkill_mcp_write"`）の絞り込みに使う |
| `include_plugin_content` | boolean | プラグインKyouの本文をレスポンスに埋め込むか（default: false） |
| `plugin_content_max_text_length` | integer | 埋め込む本文の1件あたり上限文字数（default: 4000, max: 200000） |
| `plugin_content_format` | string | 埋め込む形式。`text`（既定）/ `html` / `both` |

レスポンスフィールド:
- `kyous[]`: Kyou DTOの配列（各要素に `id`, `rep_name`, `data_type`, `related_time`, `create_app` / `update_app`（作成/最終更新アプリ名）, `tags[]`, `texts[]`, `notifications[]`, `timeis[]`（`is_include_timeis: true` のときの付随TimeIs）, `is_deleted`（削除済みのときだけ。無ければ生きている）, `payload` を含む。`tag_entities[]` / `text_entities[]` は `include_attached_ids: true` のときだけ）
- `total_count`: クエリ全体の件数（**cursor 無し応答のみ**。count_only/group_by を含む）
- `returned_count`: 今回返却した件数
- `remaining_count`: この続きに残っている件数（全応答）
- `has_more`: 続きがある場合 true
- `next_cursor`: 次ページ取得用カーソル（不透明文字列。そのまま返送する）
- `buckets[]`: group_by 指定時の集計（{key, count}）
- `partial`: 付随データ（タグ・テキスト・通知・TimeIs）の一部取得に失敗し、返した Kyou の付随データが不完全なとき true（内訳は `warnings[]` に入る）。記録保管場所の読み込み失敗とは独立で、そちらの警告があっても false の場合がある
- `warnings[]`: 未知のフィルタ値、付随データ欠落、読み込めなかった記録保管場所など。通常検索・count_only・group_by のどれでも確認する
- `plugins[]`: このページに現れたプラグインの説明（rep_name ごとに1回。各Kyouのペイロードには plugin_name しか載らない）。プラグインKyouが無いページでは省略
- `plugin_content`: 本文埋め込みの集計（`include_plugin_content: true` のときのみ）

#### 4) ペイロード（payload）フィールド
`payload.kind` でデータ型を識別できます：

| kind | 追加フィールド |
|---|---|
| `kmemo` | content |
| `kc` | title, num_value |
| `timeis` | title, start_time, end_time |
| `nlog` | title, shop, amount |
| `lantana` | mood |
| `urlog` | title, url |
| `idf` | file_name, is_image, is_video, is_audio, rep_name, mime_type |
| `git_commit_log` | commit_message, addition, deletion |
| `mi` | title, is_checked, board_name, limit_time, estimate_start_time, estimate_end_time |
| `plugin` | plugin_name（+ `include_plugin_content: true` のとき content_status / content_text / content_html / content_skipped_reason / content_error）。**プラグインの説明文は各ペイロードには載らない** —— 応答トップレベルの `plugins[]` に rep_name ごと1回だけ入る（同じ説明を件数ぶん繰り返さないため） |

`kind: "plugin"` は上記の組み込み型に該当しないプラグイン由来のKyou。本文はgkillに入っていないので、`include_plugin_content: true` を渡して同じレスポンスに埋め込ませる。

#### 5) 日時フォーマット
- 日時は ISO-8601 を推奨
- 例: `2026-02-25T10:30:00+09:00`
- 一部パラメータは `YYYY-MM-DD` も受け付け、自動で RFC3339 に補正されます
- `gkill_get_gps_log` の `start_date` / `end_date` は必須

#### 6) `gkill_get_kyous` の実用クエリ例

フィルタは**値フィールドが非nullで存在すれば有効**になる（旧 `use_X` フラグは廃止済みで、公開スキーマには載らない。古い一覧を握ったクライアントからの送信だけ後方互換で受理し、届いたら古さの警告が付く。ADR-0620）。使わないフィルタはキーごと省略するか `null` を渡す。空配列 `[]` は「フィルタ有効だが0件指定」（例外: `timeis_words: []` は「任意のTimeIsに覆われたKyou」）。

最小（デフォルト条件）:
```json
{
  "query": {}
}
```

タグ絞り込み:
```json
{
  "query": {
    "tags": ["work", "meeting"],
    "tags_and": false
  }
}
```

期間絞り込み（カレンダ）:
```json
{
  "query": {
    "calendar_start_date": "2026-02-01T00:00:00+09:00",
    "calendar_end_date": "2026-02-28T23:59:59+09:00"
  }
}
```

Mi抽出（**`for_mi` は `include_*_mi` を最低1つ要求する**。全て無指定だと0件+warningになる。`mi_sort_type: "limit_time"` は対応する射影 `include_limit_mi` が必要）:
```json
{
  "query": {
    "for_mi": true,
    "mi_check_state": "uncheck",
    "mi_sort_type": "limit_time",
    "include_limit_mi": true
  }
}
```

実行中の絞り込み（`playing_time`）: その時刻に実行中だった TimeIs を取る。カレンダ範囲と違い「その一瞬のスナップショット」になる。**リテラル `"now"` を渡すと現在時刻に展開される**ので、いま動いているものを聞きたいときは時刻を組み立てなくてよい:
```json
{
  "query": {
    "playing_time": "now"
  }
}
```

ページング（2ページ目以降）。`cursor` には**前回応答の `next_cursor` をそのまま**渡す（v2 は `{RFC3339Nano}::{ID}` の複合形式。自分で組み立て・編集しない。旧クライアントの素のISO日時も後方互換で受理される）:
```json
{
  "query": {},
  "limit": 50,
  "cursor": "2026-02-25T10:30:00+09:00::01234567-89ab-cdef-0123-456789abcdef"
}
```

#### 7) `gkill_get_application_config` の詳細

タグ階層やボード構造などの設定情報を取得します。`gkill_get_kyous` の前に呼び出すことで、より的確なクエリを構築できます。

**パラメータ:**
| パラメータ | 型 | 説明 |
|---|---|---|
| `locale_name` | string | ロケール（例: ja, en） |
| `fields` | array | 返すトップレベルフィールドの許可リスト（射影）。接続先の確認だけなら `["user_id", "device"]` で足りる（全量取得は実測94KB、この射影なら42バイト）。`"descriptions"` は仮想欄で、6 ツリーから利用者が説明を書いたノードだけを `{struct, name, path, is_dir, description}` の平坦な一覧にして返す（fields で明示したときだけ。既定の全量には載らない。ADR-0632） |
| `include_ui_state` | boolean | ツリーエディタのUI一時状態キーを含めるか（default: false。既定で剥がされる） |
| `compact` | boolean | 既定 true。ノードの既定値の欄（`children` の null / 空、`is_dir:false`、`ignore_check_rep_rykv:false`、空の `description`、識別欄と同じ `name`）を落とす。`check_when_inited` / `is_force_hide` は落とさない（ADR-0629） |
| `contains` | string | 葉の名前（識別欄 `tag_name` / `rep_name` / `rep_type_name` / `device_name` / `board_name` / `title` と表示名 `name`）の大小無視の部分一致で刈る。葉が残らないフォルダは落ちる。`descriptions` 一覧には name / path / description で掛かる |
| `max_size_mb` | number | 応答の上限（既定 0.25）。超えたら大きいツリーから `{omitted_bytes}` に置き換え、`warnings[]` で知らせる |

**レスポンスフィールド:**
| フィールド | 説明 |
|---|---|
| `user_id` | 接続アカウントのユーザーID。read / write / readwrite が別アカウントを向いていることがあるため、**書き込み前の接続先確認に使う** |
| `device` | 接続アカウントのデバイス名 |
| `tag_struct` | タグの親子階層構造。ルート 1 オブジェクト（`{name:"__root__", children:[...], is_dir:true}`）で、各要素は `tag_name`, `check_when_inited`（デフォルトチェック状態）, `is_force_hide`（非表示設定）, `children`（子タグ配列）, `description`（利用者が書いた運用メモ。非空のときだけ）を持つ |
| `mi_board_struct` | タスクボードの構造（識別欄 `board_name`。フラット） |
| `rep_struct` | リポジトリの組織構造（識別欄 `rep_name`） |
| `rep_type_struct` | リポジトリの種別構造（識別欄 `rep_type_name`） |
| `device_struct` | デバイスの組織構造（識別欄 `device_name`） |
| `kftl_template_struct` | KFTLテンプレート構造（識別欄 `title`、本文 `template`） |
| `descriptions` | `fields:["descriptions"]` のときだけ。6 ツリーから `description` が非空のノードを `{struct, name, path, is_dir, description}` で平坦に並べた一覧（`path` はルートを除く祖先の表示名を `/` で連結） |
| `mi_default_board` | デフォルトのタスクボード名（例: "Inbox"） |
| `show_tags_in_list` | タグ表示がオンかオフか |

**推奨運用:**
- まず `fields:["descriptions"]` で利用者の運用メモ（何のためのタグ / 記録保管場所 / 板 / テンプレートで、どう使っているか）を読む。説明が無いノードは「書いていない」だけなので推測しない（本文は `gkill_get_mcp_help topic:config`）
- クエリ前にタグ構造を確認し、`is_force_hide: true` のタグを `hide_tags` に含める
- `check_when_inited: true` のタグはデフォルトで選択されるタグ（ユーザーが頻繁に使うタグ）
- `children` でタグの親子関係を辿り、関連タグをまとめて検索に利用する

#### 8) エラー時の推奨リトライ方針
- 認証系（例: `ERR000013`, `ERR000002`, `ERR000238`）:
  - セッション再取得（再ログイン）後に同一リクエストを1回再試行
- 入力不正系:
  - 引数を見直して再実行（特に日時フォーマット）
  - `Invalid argument ... is not supported` は書き間違いか、**握っているツール一覧が古い**（改名・削除前の名前を載せている）かのどちらか。一覧にその名前があるなら後者で、MCP クライアントを接続し直す（サーバの再起動では直らない）。`gkill_status` の応答の `schema_revision` と、その説明文末尾の値が違えば古い
- データなし:
  - エラーではなく空配列として扱う

#### 9) 大量データ取得時の運用
- `limit` と `max_size_mb` を適切に設定してレスポンスサイズを制御する
- ChatGPTではまず `limit=20` 前後、`max_size_mb=0.25` 前後、`is_include_timeis=false` で試す
- `has_more: true` の場合は `next_cursor` を使って続きを取得する
- まず期間を絞り込んでから取得する（`calendar_start_date` / `calendar_end_date` 推奨）
- 同一条件の連打を避け、必要最小限のクエリにする
