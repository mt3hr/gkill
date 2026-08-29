# ADR-0613: 追加と更新は1つのフィールド表から作る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-25 |
| Sources | 2026-08-25 の「MCP層に同じ機能が複数実装されていたら良い方に一本化してほしい」という依頼と、その調査で見つかった `assertUrlWithScheme` の片落ち |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/write-normalization.mjs`（`ENTITY_FIELD_SPECS` / `normalizeEntityArgs`）/ `src/mcp/lib/write-handlers.mjs`（`ENTITY_SUMMARIZERS`）/ `src/mcp/lib/mcp-server-base.mjs`（`handleToolCall`）/ `src/mcp/lib/mcp-server-bootstrap.mjs` |

## Context

[ADR-0611](0611-one-table-not-two-forms.md) で `gkill_update_*` 9本の取得→patch→更新の手順を表へ寄せたが、
**引数の正規化はまだ型ごとに add / update の2関数を手書き**していた（計18本）。
違うのはフィールド名・種別・追加時に必須かどうかだけで、手順は同じ。

そして予告どおりずれていた。

> `gkill_add_urlog` は `assertUrlWithScheme` でスキーム付きURLを要求するのに、
> `gkill_update_urlog` は `assertTrimmedString` のまま。
> `assertUrlWithScheme` は `write-normalization.mjs` 全体で**追加側の1箇所でしか呼ばれていなかった**。

スキームの無い URL で更新すると gkill はページ取得を試みず、`title` が空のまま
**エラーも出さずに**保存される。呼び出し側からは「タイトルの自動補完が効かなかった」としか見えない。
入口で弾く決定（追加側）はしてあったのに、更新側だけが取り残されていた。

同じ形の重複が MCP 層にあと3つあった。

1. **`handleToolCall` が3サーバへ逐語コピー** — plugin → read → write の同じ順で、
   違うのは「write を持つか」と「read の選抜集合を持つか」の2値だけ
2. **起動ブロックが3ファイル** — diff は7行の**値**だけ
   （client クラス / ログ名 / スクリプト名 / 既定ポート / OAuth 状態ファイル名 / scope / file-link の可否）
3. **`summarizeWriteToolPayload` の 42 case のうち18本**が
   `` `Created ${型}: ${payload.added_${型}?.id}` `` / `Updated …` の一行

(1) は実害も出していた。2026-08-25 の実利用レビューが「read と readwrite でスキーマが違う」と
報告したが、コードは既に同一で、**昨日起動したままの MCP プロセスが昨日の定義を配っていた**だけだった。
それでも「3つに分かれている」という見た目自体が、その誤読を許していた。

## Decision

**手順が同じで型ごとに違うのが「データ」だけなら、そのデータを表にする。**（ADR-0611 の適用範囲を広げる）

- 引数の正規化は `ENTITY_FIELD_SPECS`（1型1行。`{name, kind, requiredOnAdd, defaultOnAdd, addOnly, nullClears}`）
  ＋ `normalizeEntityArgs(dataType, args, mode)` へ。18本の `normalizeXxxArgs` は表を呼ぶだけの口として残す
  （呼び出し側とテストの名前を変えないため）
- `handleToolCall` は `McpServerBase` へ。`options` に `readToolNames`（null なら全部）と
  `writeAppName`（null なら書き込みを持たない）を足す
- 起動処理は `lib/mcp-server-bootstrap.mjs` の `startMcpServer(spec)` へ。3ファイルは表の1行だけ持つ
- 1行要約は `ENTITY_SUMMARIZERS`（`UPDATE_TARGETS` のキーから生成）へ

**種別の定義が1箇所になった結果、`gkill_update_urlog` にも `assertUrlWithScheme` が掛かる。**
これは表化の副産物ではなく、表化する理由そのもの。

## Rejected alternatives

- **18本を残したまま `assertUrlWithScheme` を更新側にも足す** — 直るのはこの1件だけ。
  `limit_time` の「日付のみ→23:59:59」や `mood` の範囲も同じ構造で二重に書かれており、
  次に欄を1つ足すとき18箇所を触ることになる。1つ落としても**テストは緑のまま**
  （各ツールのテストは自分の欄しか見ない）
- **`ENTITY_TARGETS`（`constants.mjs`）にフィールド表も持たせて表を1つにする** — ADR-0611 と同じ理由で却下。
  `constants.mjs` は定数の置き場で、正規化関数の関心事を入れると read からも見える。
  表は用途ごとに分け、**エンドポイントの対応だけを共有**する
- **`readwrite ⊇ read` を assert するテストを足す** — 依頼者の判断で見送り。
  `handleToolCall` を基底へ畳めば、readwrite が read の上位集合であることは**構造で担保される**
  （同じ `READ_TOOLS` 配列を spread し、同じディスパッチを通る）ので、テストは二重の網になる
- **3サーバを1ファイルに統合する** — MCP クライアントの設定ファイルがコマンドラインで
  `gkill-read-server.mjs` / `gkill-readwrite-server.mjs` を名指ししており、
  統合するとポート・scope・ログの分離を引数で表すことになる。**分かれているのは入口だけでよい**
- **`normalizeEntityArgs` を write-handlers 側へ置く** — 正規化は write-tools のスキーマと対で読むものなので、
  `write-normalization.mjs` に残す

## Consequences

- `write-normalization.mjs` が 566行 → 456行。`write-handlers.mjs` が 297行 →（前回194行）→ 652行
  （バッチ削除と表の追加を含む）。3つのサーバファイルは合計388行 → 176行
- **`gkill_update_urlog` の挙動が変わる。** スキームの無い URL は受理されなくなる（従来は静かに保存）
- 古スキーマの補足（「ツールスキーマが古い」の印）が read の要約にしか無かった非対称も、
  `appendStaleSchemaNoteToSummary`（`payload.mjs`）1本にして解消した
- 表に無い kind を書くと `unknown field kind` で即死する。黙って素通しにはしない

## Evidence

- `assertUrlWithScheme` の `write-normalization.mjs` 内での呼び出しは**1箇所だけ**だった（定義を除く）
- `handleToolCall` の3実装は、plugin 分岐と read 分岐が逐語一致
- 起動ブロックの read と readwrite の diff は7行、すべて値の違い
- 実利用レビューが挙げた「read と readwrite のスキーマ差」6件は、
  **6件すべてが `git show HEAD:` 版の文言と一字一句一致**した（コードの差ではなくプロセスの世代差）

修正の検証: `ENTITY_FIELD_SPECS` の `url` の kind を `string` へ戻すと
`update_urlog もスキームの無いURLを弾く` / `add_urlog も同じ文言で弾く` /
`rejects a URL without a scheme` の3本が落ちる（実施済み）。

## Related tests

- `src/mcp/__tests__/write-normalization.test.mjs`
  - `追加と更新は同じフィールド表から作る`（add と update のURL検証が同一文言であること、
    `target_id` が update に無いこと、patch セマンティクスが保たれること）
- `src/mcp/__tests__/normalization.test.mjs`
  - `normalizeKyouArgs — 集計と cursor の併用`（GPS と get_kyous が同じ規則・同じ文言で弾く）
