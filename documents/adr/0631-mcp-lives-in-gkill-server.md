# ADR-0631: MCP サーバは gkill_server のサブコマンドとして Go で手書きし、旧 Node 実装とはゴールデンでバイト一致させる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-20 |
| Sources | 利用者の要求「MCP サーバを Go へ厳密にリプレイス。テストを先に移行し、自己チェックしてから実装。新旧の結果一致を最重視」。途中の判断「gkill_server に一本化」「ログは gkill_log を改変して別名で出す」。旧 Node 実装の最終コミットは `02634f8e`（`src/mcp`、約 9,100 行 + テスト 12,500 行・28 ファイル） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/`（package `mcp`。旧 `src/mcp/lib/*.mjs` と 1:1）/ `src/server/gkill/mcp/jsonobj/`（JSON.stringify 互換の順序つき JSON）/ `src/server/gkill/mcp/internal/fakegkill/`（偽 gkill）/ `src/server/gkill/mcp/testdata/golden/`（ゴールデン）/ `src/server/gkill/main/common/mcp.go`（`gkill_server mcp`）/ `src/server/gkill/main/common/gkill_log/gkill_log.go`（`InitNamed`） |

## Context

MCP サーバは Node.js（`src/mcp`）で書かれ、NSSM のサービス 3 本が `node` で**作業ツリーを直接実行**していた。
Go の本体と 2 言語・2 配布形態に分かれていて、コミットしても再起動しないと配られない（ADR-0619）、
Node のバージョン・`undici` の依存・vitest の設定が本体のリリースゲートとは別に要る、
ログが Node 独自の JSON 行で本体の `gkill_log` と形式が違い、error ログは経路だけあって書かれていなかった、
という運用上の割れがあった。

一方で MCP は 2026-08〜09 に実利用報告を受けて細かな挙動（未知キーの文言・古スキーマの救済・
カーソル・要約の1行）を大量に積み上げており、**書き直しで挙動が1つでも変わると、
それを頼りにしている AI クライアントが静かに壊れる**。移行の合否は「Go で動く」ではなく
「旧実装と同じバイト列を返す」でなければならなかった。

## Decision

1. **Go への手書きミラー。** package `mcp`（`src/server/gkill/mcp/`）を旧 `lib/*.mjs` と 1:1 のファイルで書く
   旧名と違うのは次の 2 つだけで、他は同名（拡張子だけ `.go`）。

   ```text
   mcp-server-base.mjs      → server_base.go
   mcp-server-bootstrap.mjs → bootstrap.go
   ```

   MCP の Go SDK は使わない。値は `jsonobj`（順序つき JSON。`JSON.stringify` 互換の直列化。
   `undefined` の番兵・キー順・`\b` / `\f` の短形式・U+2028 非エスケープ）で持ち回る。
2. **gkill_server のサブコマンド `gkill_server mcp --kind read|write|readwrite [--transport stdio|http] [--config <path>]`。**
   別バイナリは作らない。MCP は引き続き**起動中の gkill_server への HTTP クライアント**で、DB は直接読まない
   （`update_cache` / `add_tag` と同じ型）。NSSM の MCP サービス 3 本は `gkill_server.exe mcp --kind <kind>` を
   実行する別プロセスのまま（本体サービスとプロセス寿命を分ける）。
3. **設定は 1 ファイル `$GKILL_HOME/configs/gkill_mcp.json`**（無ければ初回起動時に既定値で生成）と、
   旧実装と同名の環境変数。優先順位はフラグ > 環境変数 > ファイル > 既定値。壊れたファイルは起動を止める。
4. **ログは `gkill_log` を改変して別名で出す。** `Init()` の本体を `InitNamed(prefix, app, extra...)` に切り出し
   （`Init()` = `InitNamed("gkill", "gkill")` で既存の出力は 1 バイトも変わらない）、MCP は
   `InitNamed("gkill_mcp_<kind>", "gkill_mcp", "kind", kind)` で `logs/gkill_mcp_<kind>*.log` を開く。
   stdout は JSON-RPC 専用で、`StdoutMirror` は常に無効。既定レベルは `access`（旧実装の既定 `info` で
   見えていた `http_request` / `tool_call` を残すため）。未知のレベルは起動を止める（旧実装は黙って info へ落としていた）。
5. **テストを先に厳密移行してから実装する。** vitest 28 ファイル・1,104 タイトルを `describe → TestXxx`、
   `test → t.Run("<原文のタイトル>")` で 1:1 に移し、タイトル集合の照合スクリプトで**未移植 0** を機械確認した
   （意図した例外は 6 件: ファイルの遅延オープン / close / 多重 close は `gkill_log` の `Router.Close` 側へ、
   「未知レベルは info へ落ちる」は「起動を止める」へ、`Object.isFrozen(TTL)` は Go の const なので不成立）。
6. **新旧の結果一致はゴールデンで機械証明する。** Node 実装が消える前に、偽 gkill（`internal/fakegkill`。
   固定応答 + 受信要求の記録）を別プロセスで立て、時刻と乱数を固定したプリロード付きで 3 つの stdio サーバへ
   要求コーパス（`testdata/golden/requests.json`、328 件: プロトコル・33 ツール・未知キー・古スキーマ文字列・
   廃止済み引数・enum 違反・地図 3 値欠け・集計と cursor の併用・IDF base64・`include_plugin_content`・
   履歴 offset・更新 no-op・delete→restore・KFTL replayed など）を流し、応答と偽 gkill が受けた要求を採った。
   http モードはサーバモジュールを直接 import して `handlePayload(message, requestContext)` を呼んだ。
   `golden_test.go` が tools/list（3 サーバ）・応答（stdio / http × 3 サーバ）・上流要求を**バイト単位**で照合する。
   唯一マスクするのは JSON でない応答本文のパーサ文言（`detail.cause`。V8 と Go で違う）だけ。
7. **Node 実装は同じコミットで削除する**（下の 4 つ）。

   ```text
   src/mcp/                       （ライブラリ・3 サーバ・テスト 28 本）
   vitest.config.mcp.ts
   src/tools/mcp_schema_budget.mjs
   undici                         （package.json の依存）
   ```

  `npm run test_mcp` は `go test ./gkill/mcp/...` になる（attestation のスイート名は互換のため残す）。

## Rejected alternatives

- **MCP の Go SDK（`modelcontextprotocol/go-sdk`）を使う** — 応答の形（`content[].text` の1行要約 + 整形 JSON、
  `structuredContent`、エラー時の `isError`）とキー順は旧実装が手で組んでいたもので、SDK に載せると
  バイト一致が取れず、既存クライアントが握っている `schema_revision` も変わる。プロトコル層は 300 行程度で、
  ミラーのほうが安い。
- **別バイナリ `gkill_mcp` を作る** — 配布物が 1 つ増え、リリースゲート・配置スクリプト・マニュアルの 7 言語が
  もう1系統要る。`gkill_server` に入れれば本体と同じ exe を配るだけで済む。代償は「MCP を直すたびに本体の
  停止窓が要る」で、これは受け入れた。
- **両方（Node と Go）をしばらく保守する** — 2 実装の挙動が割れた瞬間にゴールデンの意味が無くなる。
  ゴールデンは「消える前に採る」ためのもので、並走のためではない。
- **gkill_server 本体プロセスの中で MCP を同居させる（HTTP ルート `/mcp` を本体に生やす）** — 本体の
  認証・セッション・`create_app`・`isLocalRequest` の意味が変わり結果一致が崩れる。stdio モードは別プロセスが
  どのみち要る。
- **DB を直接読む** — 同上。MCP は本体の API 契約の上に居るからこそ、本体の変更を 1 箇所で受けられる。
- **ログを旧実装と同じ独自 JSON 行のままにする** — 利用者が「gkill_log を改変して別名で出す」を選んだ。
  本体と同じ回転・同じレベル語彙・同じ `source` 構造になり、error ログが実際に書かれるようになる。

## Consequences

- 結果一致の実測: tools/list は 3 サーバともバイト一致（`schema_revision` は read / write / readwrite とも旧値のまま。
  `tool_schema_budget.json` の 3 値 40,033 / 59,278 / 83,760 も据え置き）。要求コーパス 328 件 × 3 サーバ × 2 モードの
  応答と上流要求がバイト一致（`detail.cause` の 2 件だけマスク。rep をまたいで並列に投げるプラグイン本文取得の
  着順は Node でも Go でも決まっていないので、その並びだけ本文順に揃えて比べる）。
- **応答の形・文言・キー順を変えたらゴールデンが落ちる。** 意図した変更なら、Node 実装はもう動かせないので
  ゴールデンの当該行を手で更新し、理由をコミットメッセージに書く。
- ログの形式が変わる: `source` が文字列から `{function, file, line}` へ、`http_request` / `tool_call` は ACCESS、
  ファイルは統合 + レベル別の 8 本（`gkill_mcp_<kind>_access.log` は旧実装と同じ名前だが中身の形式が違う）。
- MCP を直したら本体と同じ exe を配り直す（`npm run install_server` は本番サービスの実体を上書きするので停止窓が要る）。
- `undici` が依存から消え、`package-lock.json` が変わる。
- ゴールデンで明らかになった旧実装の癖は温存した（`gkill_status` は `locale_name` を受けない、`gkill_add_kmemo` の
  `related_time:"now"` は `playing_time` と違って展開されない、`Date.parse` の受理範囲は RFC 3339 の派生形に絞った）。

## Evidence

- タイトル照合: vitest 1,104 タイトルに対し Go の `t.Run` 1,109（未移植 0・意図した例外 6・Go 側だけの追加 5:
  「未知レベルは拒否」「access レベルの通過」「level none は何も書かない」「nil Logger」「Node が書いた状態ファイルの読み込み」）。
- `go test ./gkill/mcp/...`: 全緑（ゴールデン再生を含む）。`gkill_log` の `TestInitNamedUsesPrefixForEveryFile` が
  接頭辞の付け忘れ（本体の `gkill_error.log` へ MCP の行が混ざる）を検出する。
- stdio の端から端まで（`stdio_e2e_test.go`）: テストバイナリを子プロセスにして NDJSON と Content-Length の両枠組みで
  initialize → tools/call → ping を通し、stdout に JSON-RPC 以外の行が無いこと・壊れた行が stderr に警告されることを固定。

## Related tests

- `src/server/gkill/mcp/golden_test.go`（tools/list・応答・上流要求のバイト一致、ゴールデンの jsonobj 往復）
- `src/server/gkill/mcp/stdio_e2e_test.go` / `config_test.go` / `import_graph_test.go`
- `src/server/gkill/main/common/mcp_test.go`（フラグ解決・設定ファイル生成・`--log` の明示判定・`GKILL_HOME`）
- `src/server/gkill/main/common/gkill_log/gkill_log_named_test.go`（`InitNamed` / `Close`）
