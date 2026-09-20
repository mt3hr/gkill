# mcp テスト仕様

## 概要

MCP (Model Context Protocol) サーバのテスト。1127テスト（36ファイル）で3種のMCPサーバ（Read専用・Write専用・Read/Write統合）の入力バリデーション、データ正規化、定数定義、ツールハンドラ（Read サーバ 11 + プラグイン1 = 12ツール、Write サーバ 28（書き込み21 + Read便利7）+ プラグイン1 = 29ツール、統合サーバ 32 + プラグイン1 = 33ツール。プラグインツールは3サーバ共通）、APIクライアント、サーバライフサイクル、OAuth 2.1認証（RFC 9728/8707/7591対応）、ファイルリンク配信、プラグイン本文の get_kyous へのインライン埋め込みとHTML→テキスト変換、ログ、設定ファイル、旧 Node 実装とのゴールデン一致をカバーする（テスト数は `t.Run` のサブテスト宣言の静的計数）。

2026-09-20 に Node.js 実装（旧 `src/mcp`、vitest 28 ファイル）を Go へ移した。旧テストの `describe` → `TestXxx`、`test` → `t.Run("<原文のタイトル>")` で 1:1 に対応し、タイトル集合の照合で未移植 0 を確認してある（意図した例外は [ADR-0631](../../../../documents/adr/0631-mcp-lives-in-gkill-server.md)）。

## テストフレームワーク

Go `testing` パッケージ（gkill 本体への往復は `mockClient`（`mock_client_test.go`）と偽 gkill（`internal/fakegkill`）で代替。実行中の gkill_server は不要）

## テストファイル一覧

### 共通・Read専用サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `validation_test.go` | MCP ツール入力のバリデーション |
| `normalization_test.go` | クエリデータの正規化処理。`count_only` / `group_by` と `cursor` の併用、および `count_only` と `group_by` の併用をMCP層で弾くこと（GPS と同じ文言であることも含む）、`gkill_get_rep_infos` の `data_kinds` 絞り込みと行絞り込み（`writable_only` / `rep_types` / `rep_names` / `contains`。古スキーマ経由の文字列復元も）、古スキーマ救済表 `StaleSchemaArgKindsByTool` の全エントリを表駆動で回す検出網羅 |
| `constants_test.go` | 定数定義の検証 |
| `tool_handlers_test.go` | Read 11ツール分のハンドラ実行ロジック（`read_tools.go` のツール名一覧・エンドポイント対応表・summarize）。`gkill_get_kyous` の Description が `partial=false` と独立に `warnings` を確認し、読み込めない保管場所を `query.reps` へ指定し直さないようAIへ伝えることも固定する |
| `read_handlers_test.go` | 読み取りディスパッチの正本 `read_handlers.go`（get_kyous v2 パラメータの転送と応答の素通し、トップレベル `plugins[]` の条件付きコピー、application_config の fields 射影 + UI状態キー strip、GPS の MCP 側ページング（複合カーソル・count_only・日別バケット・count_only×group_by の拒否）、rep_infos の行絞り込み、idf_file の `/files/` クエリ組み立て・thumb エコー・サイズ上限超過の案内） |
| `client_test.go` | GkillClient（httptest のスタブ、ログイン・認証リトライ等） |
| `server_test.go` | Server のセットアップとトランスポート管理、セッションオーバーライド、プラグインツール振り分け、IDF base64 の text/structuredContent 分離と `image_content_attached`、`warnings` / `partial` の1行要約への昇格 |
| `access_log_test.go` | Logger（レベルフィルタリング・JSON形式・source。未知レベルは拒否・nil Logger は無害） |
| `pkce_test.go` | PKCE (S256/plain) のコード検証、バリデーション |
| `oauth_store_test.go` | OAuthストア（認可コード、アクセストークン、リフレッシュトークン、クライアント登録、TTL期限切れ、定期クリーンアップ、JSONファイル永続化。旧 Node 実装が書いた状態ファイルをそのまま読めること） |
| `oauth_server_test.go` | OAuth 2.1サーバ（メタデータ、認可フロー、トークン交換、PKCE検証、リフレッシュトークンローテーション、動的クライアント登録、E2Eフロー）。scope の1値強制、同意画面の3 scope 表示と DCR 由来 `client_name` の HTML エスケープ |
| `file_link_test.go` | FileLinkStore（HTTPモード用の期限付きファイルリンクトークンの発行・解決・失効、`GET /files/{token}` 配信） |
| `http_transport_test.go` | HttpTransport の `/mcp` 経路の統合・回帰（実ポートで OAuth→Bearer→tools を通す。Bearer 401、並行リクエストの user/session 分離、公開ファイル配信の nosniff / CSP sandbox、ボディ上限 413・明示タイムアウト・アクセスログのクエリ除去、scope 境界と `token_scope_rejected` 監査ログ） |
| `readme_examples_test.go` | README の ```json 例を実物の正規化器（`NormalizeKyouArgs`）へ通す同期検査 |
| `status_tool_test.go` | ツール一覧の世代 `schema_revision`（`status_tool.go`）の計算と `gkill_status` への焼き込み |
| `help_topics_test.go` | `gkill_get_mcp_help`（`help_topics.go`）: 全 topic に本文があること、名指しするツール名の実在、3サーバ搭載、未知の topic の拒否 |
| `schema_contract_test.go` | 「tools/list どおりに呼ぶと失敗しない」の契約: スキーマのキー集合 = 正規化器の受理集合 − 廃止済み、全ツールをスキーマの全プロパティ指定で呼んで未知キーで拒否されないこと、3サーバの同名ツールが同じ JSON、`initialize` の version・説明文の印・応答の `schema_revision` の一致 |
| `tool_schema_budget_test.go` | tools/list のバイト量が予算ファイル `tool_schema_budget.json` の範囲内であること（超過で失敗、1024 バイト以上の減少でも予算の追随を要求）、計測の決定性、over / under / missing の判定と文言 |
| `start_spec_test.go` | 3サーバの `StartSpec` の宣言値固定。scope / 既定ポート / file-link 可否・3サーバ間の重複禁止・bootstrap が `spec.Scope` を OAuthServer へ渡す配線。`server_start` ログの世代情報 `startInfo` と、stdio・http の両トランスポートがそれを出すこと |
| `config_test.go` | 設定ファイル `gkill_mcp.json` の生成（既定値・0600・既存は書き換えない・壊れていれば起動を止める）と、フラグ > 環境変数 > ファイル > 既定値の優先順位（`ResolveSettings`） |
| `import_graph_test.go` | package `mcp` が `gkill/api` / `dao` / `usecase` / `main/common` を import しないこと（MCP は起動中サーバの HTTP クライアント） |
| `stdio_e2e_test.go` | stdio の端から端まで。テストバイナリ自身を子プロセスにして NDJSON と Content-Length の両枠組みで initialize → tools/call → ping を通し、stdout に JSON-RPC 以外の行が無いこと・壊れた行が stderr に警告されることを固定する |
| `golden_test.go` | 旧 Node 実装から採ったゴールデン（`testdata/golden/`。要求コーパス 331 件）との**バイト一致**: tools/list（3サーバ）と `schema_revision`、tools/call の応答（stdio / http × 3サーバ）、gkill へ送った要求（パス・クエリ・Cookie・本文）。時刻・UUID・トークンは採取時と同じ固定列。ゴールデン自身が `jsonobj` で往復してもバイト単位で変わらないことも固定。ツールを意図して変えたときは `GKILL_MCP_UPDATE_GOLDEN=1 go test ./gkill/mcp/ -run Golden` で Go の出力へ書き直し、`git diff testdata/golden` を読んでからコミットする（Node 実装はもう無いので、以後は前回コミットした Go の出力との回帰検査になる） |

### プラグインツール（3サーバ共通）

| ファイル | テスト内容 |
|---------|-----------|
| `plugin_tools_test.go` | プラグインツール（`gkill_get_plugin_list`）の定義とエンドポイント振り分け、`CollectPluginPayloads` / `RunGroupedWithConcurrency`（キー内直列・キー間並列・例外隔離）、`InlinePluginContents`（format別レスポンス・truncated・重複取得の集約・rep単位の失敗打ち切り・max_kyous/budget/deadline のskip・統計の整合）・summarize関数、`provides` を宣言しないプラグインを warnings で名指しすること |
| `html_text_test.go` | `HtmlToText` / `DecodeHtmlEntities`（script/style/コメント破棄、ブロック境界の改行化、エンティティデコード順、maxLength切り詰め） |

### Write専用サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `write_normalization_test.go` | Write入力の正規化（11 normalizer関数、mood範囲検証、data_type検証等）。追加と更新が同じ `entityFieldSpecs` から作られること、`idempotency_key` の受理、後付けフラグの既定値・addOnly・古スキーマ文字列の復元、delete / restore の対象無指定を verb 入り文言で拒否すること |
| `write_client_test.go` | GkillClient の書き込み側（環境変数、login、callApi、認証リトライ） |
| `write_server_test.go` | Write サーバ（JSON-RPC、29ツールディスパッチ、プラグインツール振り分け、エンティティデフォルト値、レスポンス構造、warnings の1行要約昇格） |
| `write_tool_handlers_test.go` | Write 21ツール定義（実物 import）・削除の語彙が enum / `DeleteDataTypes` / 対応表2つで一致すること・`SummarizeWriteToolPayload`・後付け boolean 引数が検出表と型復元の両方に載ることの表駆動メタ検査 |

### Read/Write統合サーバ

| ファイル | テスト内容 |
|---------|-----------|
| `readwrite_client_test.go` | GkillClient（callApi統合メソッド、fetchFile、認証リトライ） |
| `readwrite_server_test.go` | 統合サーバ（33ツール全ディスパッチ、プラグインツール振り分け、IDF画像ブロック、エンティティデフォルト値） |
| `write_handlers_test.go` | 書き込みディスパッチの正本（add/update/delete/restore のエンドポイント、update の patch セマンティクス、create_app がサーバ種別で埋まること、既削除の delete / 未削除の restore を拒む冪等ガード、update_time が同一秒でも必ず進むこと、応答がサーバ保存版を返すこと、`end_time: null` の3値パッチ、`allow_create_board:false` の板名照合） |

### テスト補助（テスト件数には数えない）

| ファイル | 内容 |
|---------|------|
| `helpers_test.go` | `expectEqual`（`jsonobj.Equal` = vitest の `toEqual`）ほか、旧テストの表明を写す補助 |
| `mock_client_test.go` | `GkillAPI` のモック（`vi.fn()` 相当。呼び出し記録と Once キュー） |
| `fetch_mock_test.go` | HTTP クライアント検査用の `httptest` スタブ |
| `internal/fakegkill/` | 偽 gkill（固定応答・受信要求の記録・セッション切れの1回再現・ファイル配信）。ゴールデンの採取と再生が共用する |

## テスト内容

- **Validation / Normalization / Write Normalization / Constants**: 各ツールの入力パラメータ検証、日付フォーマット、デフォルト値補完、data_type 列挙値、unknown keys 拒否
- **Tool Handlers**: Read 11ツール + Write 21ツール（add系9 + update系9 + submit_kftl + delete_kyou + restore_kyou）+ Read便利7ツール + プラグイン1ツール（3サーバ共通）
- **Plugin Tools**: `gkill_get_plugin_list` と `include_plugin_content` によるプラグイン本文のインライン埋め込み（並列度・予算・デッドライン・失敗隔離）、コンテンツHTMLのテキスト変換
- **Client / Server / Transport**: gkill への往復（認証、エラーハンドリング、レスポンスパース）、3サーバのディスパッチ、JSON-RPC、stdio の枠組み、HTTP の OAuth 2.1 と file-link 配信
- **Config / Log**: 設定ファイルの生成と優先順位、gkill_log 上のロガーのレベル
- **Golden**: 旧 Node 実装とのバイト一致（tools/list・応答・上流要求）

## 実装の置き場所

ディスパッチと起動処理は3サーバ共通で、各サーバファイル（`server_read.go` / `server_write.go` / `server_readwrite.go`）は
「どのツールを載せ、どの名前とポートで名乗るか」の表1行だけを持つ。

- トランスポート: `stdio_transport.go` / `http_transport.go`
- JSON-RPC の受け口: `server_base.go`
- ツール定義: `read_tools.go` / `write_tools.go` / `plugin_tools.go`
- gkill 本体との通信: `gkill_client.go`
- サブコマンドの配線: `src/server/gkill/main/common/mcp.go`（テストは `main/common/mcp_test.go`）

ツール数（Read 12 / Write 29 / ReadWrite 33）は `verify_docs` が `composeTools(...)` の連結を辿って実測と突き合わせる。

## 実行方法

```bash
npm run test_mcp
```

または:

```bash
cd src/server && go test ./gkill/mcp/...
```
