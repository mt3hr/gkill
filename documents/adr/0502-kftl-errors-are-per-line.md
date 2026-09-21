# ADR-0502: メモ帳（KFTL）の失敗は行ごとに返し、入力ミスとサーバ障害を分ける

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | MCP の2巡目の指摘（P-20 / P-22 / P-40）。gkill-go-backend スキルの「HTTP ステータス」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/kftl/kftl_statement.go`（`KFTLInputError` / `CollectKFTLInputErrors`） / `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text.go` |

## Context

メモ帳（KFTL）は1つのテキストから複数の Kyou を作る。にもかかわらず応答は
**成功なら1文、失敗なら1文**しか返していなかった。

2巡目の指摘での実測で、**まったく違う2つの失敗が完全に同じ応答になった**:

| 入力 | 実際の原因 | 応答 |
|---|---|---|
| `/end` 単独 | 終了する打刻が無い | `ERR000351「メモ帳のテキストの記録に失敗しました」` |
| `/mood` + `99` | 気分値が 0-10 の範囲外 | **同上** |

問題は3つあった。

1. **原因が捨てられていた。** `kftl_statement.go` は失敗した行のテキストを
   `error applying line %q` と error 文言に埋めていたのに、
   `handle_submit_kftl_text.go` がそれを `slog` にだけ出して、応答には定型文を積んでいた
2. **利用者の打ち間違いが HTTP 500 で返っていた。** `ERR000351` は 500 に割り当てられており、
   ステータスを見る層（監視・プロキシ・素朴な HTTP クライアント）からは
   入力ミスとサーバ障害が区別できない
3. **成功しても何が作られたか分からなかった。** ID も件数も種別も返らず、
   ツール説明にも「KFTL は作成した ID を返さない」と書いてあった

効く先は MCP と Wear OS だけ（Web は TS 側で解析して `add_*` へ fan-out するのでこの API を呼ばない）。
Wear は `errors.first().error_message` **だけ**を読むので、文言が改善すると追加作業ゼロで効く。

TS 側には最初から `get_invalid_line_indexs()` と細かいエラーコード（`ERR9xxxxx` 帯）があり、
`do_request()` は複数エラーを集めて返す。**Go 側だけが1件で止まっていた。**

## Decision

**入力ミスは行ごとに1件ずつ、400 で返す。サーバ障害は従来どおり1件、500 で返す。**

- 行ごとの文脈（`KFTLStatementLineContext`）に `LineIndex` を1本足す。
  インタフェースを変えないので47ある行の実装には波及しない
- **行をリクエストへ適用するフェーズは全行を評価してから返す。** ここはまだ1バイトも書いていないので
  安全で、利用者は1往復で全部直せる。束ねは `errors.Join`
- `KFTLInputError`（行番号・行テキスト・多言語メッセージID・原因）を定義し、
  ハンドラは `errors.As` で振り分ける。実行フェーズは既定でサーバ障害、
  ただし `/end` の「終了対象が無い」「終了タイトルが空」は入力ミスとして型付ける
- エラーコードは **`ERR000416`（400）を1本だけ**新設する
- 応答へ `created[]`（`{id, data_type, updated}`）を足す。**書き込みが成功した直後に
  リクエスト側が控える** —— 本文が空の kmemo / Mi / Nlog は何も書かずに成功し、
  打刻の終了は既存レコードの更新なので、リクエストを事前に並べるだけでは嘘になる
- **失敗しても、そこまでに書けたぶんを `created[]` に載せる。** KFTL は DB トランザクションを
  使わないので部分保存が残る（`kftl_request.go` が明言）。何が残ったか分からないと後始末ができない

多言語キーは**新設しない**。「気分値が範囲外です」「終了対象の打刻が存在しませんでした」など、
各失敗に1対1で対応する文言が7言語で既に約100キー揃っている。

## Rejected alternatives

- **失敗ごとにエラーコードを増やす** — 1本足すごとに7ファイル（定数・ステータス表・
  ステータス別件数のテスト・README 2本・資料2本）が動く。分けたい軸は
  「利用者が直せるか / サーバが悪いか」の1本だけなので、コードも1本でよい。
  細かい内訳は `error_message` に載せる
- **`ERR000351` のまま文言だけ良くする** — 文言は直るが**ステータスが 500 のまま**で、
  打ち間違いがサーバ障害に見える問題が残る。ADR-0610 相当の HTTP ステータス整備の趣旨に反する
- **実行フェーズも全部評価して集める** — 書き込みが起きたあとに走り続けることになり、
  1つ目の失敗が後続を壊す状況（rep が落ちている等）で被害が広がる。止めるほうが安全
- **`requestMap` を事前に列挙して「作られたもの」とする** — 本文が空の kmemo / Mi / Nlog は
  何も書かずに成功し、打刻の終了は既存レコードの更新なので**嘘になる**
- **KFTL を DB トランザクションで包む** — 別の大きな決定で、rep が複数 DB に分かれている
  現在の構造では成立しない。ここでは「部分保存が残ること」を隠さずに返すほうを選ぶ
- **TS 側のエラーコード（`ERR9xxxxx`）を Go でも使う** — その帯はフロントエンド専用と
  `api/message/README.md` に明記されている

## Consequences

- メモ帳の入力ミスが **400 + 行ごとのエラー**で返る。Wear の失敗表示も追加作業ゼロで親切になる
- 応答に `created[]` が増える。既存クライアントはフィールドを無視するだけで壊れない
  （Wear の Kotlin は `ignoreUnknownKeys = true`、Web はこの API を呼ばない）
- **`/end` にタグを付けたときに実在しない ID 宛のタグが残る問題も一緒に消えた。**
  タグ・テキストを書く処理が「終了対象が無い」判定より前にあったので、後ろへ動かした
- 冪等キーで再送を畳んだときは実行していないので `created[]` は空になる。
  「1回目は ID あり、再送は無し」という非対称が残る

## Evidence

- 2巡目の指摘での実測（本番 gkill へ stdio 接続）: `/end` と `/mood`+`99` が
  同一の `ERR000351` を返した。成功時は `MSG000076` の1行のみ
- 原因の文字列は既に存在していた: `kftl_statement.go` が
  `error applying line %q` と失敗行のテキストを埋めており、
  `handle_submit_kftl_text.go` が `slog.Log(..., Debug, ...)` にだけ出して捨てていた
- 行番号の復元可能性: `generateKFTLLines` の `for i, lineText := range lineTexts` の `i` が
  0始まりの行位置そのもので、打ち切り（`！`）は append の前にあるため
  `lines[j]` の添字 == 元テキストの行番号が常に成立する
- 作成 ID の所在: 生成系リクエストはすべて `ID: r.RequestID` でエンティティを書いている

## Related tests

- `src/server/gkill/api/kftl/kftl_statement_test.go`
  - `TestGenerateAndExecuteRequests_InputErrorCarriesLineNumber`
  - `TestGenerateAndExecuteRequests_CollectsEveryBadLine`
  - `TestGenerateAndExecuteRequests_LineIndexMatchesSourceLine`
  - `TestCollectKFTLInputErrors_FlattensJoinedErrors` / `TestCollectKFTLInputErrors_ReturnsNilForServerFailure`
- `src/server/gkill/api/message/http_status_test.go`（新しいコードがステータス表に載っていること）
- `src/server/gkill/mcp/write_tool_handlers_test.go`（要約が作られたものを報告すること）
