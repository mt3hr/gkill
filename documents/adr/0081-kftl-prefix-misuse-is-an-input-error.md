# ADR-0081: 引数の無い／引数を同じ行に書いたメモ帳のプレフィックスは、ゼロ値を書かずに行別エラーにする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | 2026-08-24 の実利用レビュー（書き込み破壊試験）。[ADR-0080](0080-kftl-errors-are-per-line.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/kftl/kftl_factory.go`（`prefixWrittenWithArgument`）/ `src/server/gkill/api/kftl/kftl_statement.go`（`requireNextLineText`）/ `src/server/gkill/api/kftl/kftl_kmemo.go` |

## Context

メモ帳（KFTL）のプレフィックス判定は**完全一致**で、値は次の行に書く。
ところが書き方を外したときの結末が型ごとにばらばらで、しかもどれも黙っていた。

| 入力 | 従来の結末 |
|---|---|
| `/mood`（単独） | **気分値 0（最低）の記録が黙って1件書かれる** |
| `/num`（単独） | タイトルも数値も空の数値記録が黙って1件書かれる |
| `/mood 8`（1行に） | 本文「/mood 8」のメモが1件。応答は「wrote 1 record(s) — kmemo」 |
| `/expense` `/url` `/mi` `/start` `/timeis`（単独） | 無言で0件 |

`/mood` 単独が最低値を書くのが最も実害が大きい。**本人の記録として残り、
気分の推移グラフに 0 が混ざる。**しかも「気分を記録したつもり」の利用者には気づく手段が無い。

`/mood 8` の方は MCP のツール説明が「1行に書くと kmemo になる」と正しく書いていた。
問題は説明ではなく、**サーバが黙っていること**だった。

あわせて、ADR-0080 が「打ち間違いを 500 で返さない」と定めたのに、
`newKFTLInputError` を使っているのは5箇所だけで、日時の解釈失敗・金額の解釈失敗・
無効行への入力などは `fmt.Errorf` のまま **ERR000351（HTTP 500）＋英語の生文言**で返っていた。
`？2026-13-45` と打つだけで 500 になる。対応する i18n キーは7言語すべてに存在し、
使われていないだけだった。

## Decision

- **単独プレフィックス**（次に値の行が無い）は、`requireNextLineText` で行別エラーにする。
  対象は `/mood` `/num` `/expense` `/url` `/mi` `/start` `/timeis` と日本語版
- **プレフィックス＋同じ行の引数**は、`prefixWrittenWithArgument` で行別エラーにする。
  判定は「既知のプレフィックスで始まり、直後が空白」。長いプレフィックスから見る
  （短い側から見ると `/end?` が「`/end` に引数 `?` を付けた行」に化ける）
- どちらも `kftl_kmemo.go` / 各 start 行の `ApplyThisLineToRequestMap` で見る。
  **このフェーズはまだ1バイトも書いていない**ので、全行を評価して `errors.Join` で束ね、
  1往復で全部直せる（ADR-0080 の仕組みにそのまま乗る）
- `fmt.Errorf` のままだった打ち間違い8箇所を `newKFTLInputError` へ差し替える。
  メッセージIDは**既存のキーを使う**（新設しない）
- 新しい i18n キーは2つだけ足す:
  `KFTL_REQUIRE_VALUE_LINE_MESSAGE_TITLE` と `KFTL_PREFIX_MUST_BE_ALONE_ON_LINE_MESSAGE_TITLE`

## Rejected alternatives

- **単独プレフィックスを無視して0件にする** — `/expense` などの既存挙動へ揃える案。
  だが「なぜ作られなかったのか」が分からないままで、`/mood` の 0 書き込みだけ直しても
  利用者から見た不可解さは残る。ADR-0080 の趣旨（打ち間違いを見えるようにする）とも合わない
- **`HasPrefix` で受理して同じ行の引数を読む**（`/mood 8` を気分値8として解釈する） —
  タグ（`。` / `#`）と関連時刻（`？` / `?`）は**既に前方一致**で受理しており、
  ここを前方一致に広げると「`# 見出し`」がタグ、「`? を含む英文`」が関連時刻、という
  既存の書き方と衝突する。TS 実装（`kftl-prefixes.ts`）とも二重に食い違う
- **`DoRequest`（書き込みフェーズ）で検査する** — そこまで来ると、同じ送信の前の行は
  もう書き込まれている。KFTL は DB トランザクションではないので、
  検査は「1バイトも書く前」でなければ意味が薄い
- **単独プレフィックスをゼロ値ではなく「スキップ」にする**（kmemo の空スキップと同じ扱い） —
  0 が書かれる事故は消えるが、`/expense` 系の「無言で0件」と同じ問題に合流するだけ
- **`api/kftl/` に検査用の .go を新設する** — `verify_docs.mjs` が
  `api/kftl/` の .go ファイル数を数えて `api/README.md` と突き合わせている。
  判定ヘルパはプレフィックス定数の隣（`kftl_factory.go`）に置けば足りる

## Consequences

- 気分値 0 と空の数値記録が黙って作られることは無くなった
- `/mood 8` のような書き間違いが 400 と行番号つきで返る。**他の行は今までどおり書かれる**
  （KFTL はトランザクションではない。`created[]` が何が残ったかを返す契約は変わらない）
- 打ち間違いで HTTP 500 が返る経路が消えた。Wear / Android もこの1文しか読めないので効く
- i18n キーが 922 → 924 になった（7言語 + Go の embed コピー + 件数を書いた資料4箇所）
- タグと関連時刻の前方一致は**意図的に据え置き**。`# 見出し` がタグになる挙動は変わらない
  （MCP のツール説明にその旨を明記した）

## Evidence

- 2026-08-24 の破壊試験: `/mood 8` の1行送信が「wrote 1 record(s) — kmemo」で完了。
  気分記録は作られていないのに成功に見えた
- コード読みで判明した追加分: `/mood` 単独は `kftl_lantana.go` の `DoRequest` が
  `Mood: r.mood`（ゼロ値 0）をそのまま書く。空ガードが無い

## Related tests

- `src/server/gkill/api/kftl/kftl_statement_test.go`
  - `TestStatement_PrefixWithoutValueLineIsInputError`（7プレフィックス）
  - `TestStatement_JapanesePrefixWithoutValueLineIsInputError`
  - `TestStatement_PrefixWithArgumentOnSameLineIsInputError`
  - `TestStatement_PrefixOnOwnLineStillWorks`（正しい書き方が通る）
  - `TestStatement_PrefixDetectionDoesNotCatchOrdinaryText`（`# 見出し` などを巻き込まない）
  - `TestStatement_PrefixDetectionPrefersLongestPrefix`（`/end?` を誤検出しない）
