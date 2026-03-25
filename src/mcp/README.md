## MCP連携（Read専用）
gkill の Read API を MCP サーバとして公開できます。
この実装は `R` のみ対応で、`Add/Update/Delete` は公開しません。

2つのトランスポートモードに対応：
- **stdio** (デフォルト): Claude Desktop等のローカルMCPクライアント向け
- **HTTP** (Streamable HTTP): Claude.ai、ChatGPT等のリモートMCPクライアント向け

### stdioモード（デフォルト）
1. gkill_server を起動（例: `http://127.0.0.1:9999`）
2. 環境変数を設定
3. MCP サーバを起動

```powershell
$env:GKILL_BASE_URL="http://127.0.0.1:9999"
$env:GKILL_USER="admin"
$env:GKILL_PASSWORD_SHA256="<sha256 hex>"  # または GKILL_PASSWORD
npm run mcp:gkill-read
```

### HTTPモード（リモート接続）
OAuth 2.1認証付きのHTTPサーバとして起動します。Claude.ai、ChatGPT等のMCPクライアントからOAuth認証で接続できます。

```powershell
$env:GKILL_BASE_URL="http://127.0.0.1:9999"
$env:GKILL_USER="admin"
$env:GKILL_PASSWORD_SHA256="<sha256 hex>"
$env:MCP_TRANSPORT="http"
$env:MCP_OAUTH_ISSUER="https://<your-host>"  # リモート接続時は必須（クライアントがアクセス可能な公開URL）
$env:MCP_PORT="8808"  # 省略可（デフォルト: 8808）
npm run mcp:gkill-read-http
```

```bash
# Linux/macOS
GKILL_BASE_URL=http://127.0.0.1:9999 \
GKILL_USER=admin \
GKILL_PASSWORD_SHA256="<sha256 hex>" \
MCP_TRANSPORT=http \
MCP_OAUTH_ISSUER="https://<your-host>" \
MCP_PORT=8808 \
node src/mcp/gkill-read-server.mjs
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
- リフレッシュトークンとDCRクライアント登録は `$GKILL_HOME/configs/mcp_oauth_state.json` に自動永続化。サーバー再起動後も再認証不要

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

### 主な環境変数

#### gkill接続
- `GKILL_BASE_URL` (default: `http://127.0.0.1:9999`)
- `GKILL_USER`
- `GKILL_PASSWORD_SHA256` または `GKILL_PASSWORD`
- `GKILL_SESSION_ID` (任意。指定時はログインをスキップ)
- `GKILL_LOCALE` (default: `ja`)
- `GKILL_INSECURE` — `true` or `1` でgkill_serverへの接続時にTLS証明書検証をスキップ（自己署名証明書用）

#### トランスポート
- `MCP_TRANSPORT` (default: `stdio`) — `stdio` or `http`
- `MCP_PORT` (default: `8808`) — HTTPサーバのポート番号
- `MCP_OAUTH_ISSUER` (default: `http://localhost:<MCP_PORT>`) — OAuthメタデータのissuer URL。**リモート接続時は必須**。クライアントがアクセス可能な公開URL（例: `https://example.com`）を設定。未設定だとClaude.ai/ChatGPTからOAuth認証が失敗する

### 提供ツール（6つ）
| ツール名 | 説明 |
|---|---|
| `gkill_get_kyous` | Kyou一覧を取得（タグ・テキスト・型データをインライン返却） |
| `gkill_get_mi_board_list` | Miボード名一覧を取得 |
| `gkill_get_all_tag_names` | 全タグ名を取得 |
| `gkill_get_all_rep_names` | 全リポジトリ名を取得 |
| `gkill_get_gps_log` | 期間指定でGPSログを取得 |
| `gkill_get_application_config` | アプリケーション設定を取得（タグ階層・ボード構造・テンプレート等） |

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
1. まず `gkill_get_application_config` でタグ階層・ボード構造・リポジトリ構造を把握する
2. 必要に応じて `gkill_get_all_tag_names` / `gkill_get_all_rep_names` / `gkill_get_mi_board_list` でメタ情報を補完
3. `gkill_get_kyous` でKyou一覧を取得（タグ・テキスト・型データはレスポンスにインライン）
4. 件数が多い場合は `cursor` / `next_cursor` でページングして追加取得
5. 地図系は `gkill_get_gps_log` を使う

#### 3) `gkill_get_kyous` のパラメータ
| パラメータ | 型 | 説明 |
|---|---|---|
| `query` | object | FindQuery（後述） |
| `locale_name` | string | ロケール（例: ja, en） |
| `limit` | integer | 最大取得件数（default: 20） |
| `cursor` | string | 前回レスポンスの `next_cursor` を指定してページング。ISO-8601推奨 |
| `max_size_mb` | number | レスポンスの最大サイズMB（default: 0.25） |
| `is_include_timeis` | boolean | 各Kyouに付随する TimeIs を含めるか（default: false） |

レスポンスフィールド:
- `kyous[]`: Kyou DTOの配列（各要素に `data_type`, `related_time`, `tags[]`, `texts[]`, `notifications[]`, `payload` を含む）
- `total_count`: クエリ全体の件数
- `returned_count`: 今回返却した件数
- `has_more`: 続きがある場合 true
- `next_cursor`: 次ページ取得用カーソル（ISO-8601）

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
| `idf` | file_name |
| `git_commit_log` | commit_message, addition, deletion |
| `mi` | title, is_checked, board_name, limit_time, estimate_start_time, estimate_end_time |

#### 5) 日時フォーマット
- 日時は ISO-8601 を推奨
- 例: `2026-02-25T10:30:00+09:00`
- 一部パラメータは `YYYY-MM-DD` も受け付け、自動で RFC3339 に補正されます
- `gkill_get_gps_log` の `start_date` / `end_date` は必須

#### 6) `gkill_get_kyous` の実用クエリ例

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
    "use_tags": true,
    "tags": ["work", "meeting"],
    "tags_and": false
  }
}
```

期間絞り込み（カレンダ）:
```json
{
  "query": {
    "use_calendar": true,
    "calendar_start_date": "2026-02-01T00:00:00+09:00",
    "calendar_end_date": "2026-02-28T23:59:59+09:00"
  }
}
```

Mi抽出:
```json
{
  "query": {
    "for_mi": true,
    "mi_check_state": "uncheck",
    "mi_sort_type": "limit_time"
  }
}
```

ページング（2ページ目以降）:
```json
{
  "query": {},
  "limit": 50,
  "cursor": "2026-02-25T10:30:00+09:00"
}
```

#### 7) `gkill_get_application_config` の詳細

タグ階層やボード構造などの設定情報を取得します。`gkill_get_kyous` の前に呼び出すことで、より的確なクエリを構築できます。

**パラメータ:**
| パラメータ | 型 | 説明 |
|---|---|---|
| `locale_name` | string | ロケール（例: ja, en） |

**レスポンスフィールド:**
| フィールド | 説明 |
|---|---|
| `tag_struct` | タグの親子階層構造。各要素は `tag_name`, `check_when_inited`（デフォルトチェック状態）, `is_force_hide`（非表示設定）, `children`（子タグ配列）を持つ |
| `mi_board_struct` | タスクボードの構造 |
| `rep_struct` | リポジトリの組織構造 |
| `rep_type_struct` | リポジトリの種別構造 |
| `device_struct` | デバイスの組織構造 |
| `kftl_template_struct` | KFTLテンプレート構造 |
| `mi_default_board` | デフォルトのタスクボード名（例: "Inbox"） |
| `show_tags_in_list` | タグ表示がオンかオフか |

**推奨運用:**
- クエリ前にタグ構造を確認し、`is_force_hide: true` のタグを `hide_tags` に含める
- `check_when_inited: true` のタグはデフォルトで選択されるタグ（ユーザーが頻繁に使うタグ）
- `children` でタグの親子関係を辿り、関連タグをまとめて検索に利用する

#### 8) エラー時の推奨リトライ方針
- 認証系（例: `ERR000013`, `ERR000002`, `ERR000238`）:
  - セッション再取得（再ログイン）後に同一リクエストを1回再試行
- 入力不正系:
  - 引数を見直して再実行（特に日時フォーマット）
- データなし:
  - エラーではなく空配列として扱う

#### 9) 大量データ取得時の運用
- `limit` と `max_size_mb` を適切に設定してレスポンスサイズを制御する
- ChatGPTではまず `limit=20` 前後、`max_size_mb=0.25` 前後、`is_include_timeis=false` で試す
- `has_more: true` の場合は `next_cursor` を使って続きを取得する
- まず期間を絞り込んでから取得する（`use_calendar` 推奨）
- 同一条件の連打を避け、必要最小限のクエリにする
