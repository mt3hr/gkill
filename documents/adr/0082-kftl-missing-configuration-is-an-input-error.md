# ADR-0082: KFTL の実行フェーズの失敗も、設定不足なら行別の入力エラーにする

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-25 |
| Sources | 2026-08-25 の実利用レビュー（「KFTL の `~~` が MCP 経由で常に失敗する」）と、`user_config.db` / アクセスログ / 実データによる原因特定 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/kftl/kftl_mirekyou.go`（`DoRequest` の事前検査）/ `src/server/gkill/api/kftl/kftl_statement.go`（実行ループの `errors.As`） |

## Context

実利用レビューが「`~~`（リポストタスク）が MCP 経由だと最小形すら通らない。3回試して3回とも失敗」と報告し、
**MCP 経路固有の不具合**と診断した。

原因は転送路ではなかった。**MCP が繋いでいるアカウントに `mirekyou` 型のリポジトリが1件も無かった**。
`user_config.db` の `REPOSITORY` にその行が無く、`WriteMiReKyouRep` が nil のまま
`kftl_mirekyou.go` の事前検査に落ちていた。同じアカウントなら **Web UI からでも同じく失敗する**。

にもかかわらず「MCP だけ壊れている」と読めてしまったのは、返るエラーがこれだけだったから。

```
ERR000351: メモ帳のテキストの記録に失敗しました
```

行番号も理由も無い。同じ `gkill_submit_kftl` でも、`/mood 9` のような書き方の誤りには

```
ERR000416: おかしな行があります (line 1: "/mood 9"): 記号は行に単独で書き、値は次の行に書いてください
```

が返る（[ADR-0080](0080-kftl-errors-are-per-line.md) / [ADR-0081](0081-kftl-prefix-misuse-is-an-input-error.md)）。
落差の理由は2つ。

1. **フェーズが違う。** 行番号を付ける `withLine` は Apply フェーズのループにあり、
   `~~` の失敗はすべて実行フェーズ（`DoRequest`）で起きる
2. **型が違う。** `kftl_mirekyou.go` の失敗5種はどれも `fmt.Errorf` で、
   `*KFTLInputError` を作らない。他の kftl ファイルには `newKFTLInputError` が22箇所あるのに、
   このファイルは**0箇所**だった

ADR-0080 は「実行フェーズは既定でサーバ障害」と決めており、それ自体は正しい。
だが **`~~` の失敗5種はどれもサーバ障害ではない** —— 3つは「送ったテキストの書き方」、
1つは「アカウントの設定」で、どちらも利用者が直せる。

さらに、実行フェーズは非トランザクションなので**失敗のたびに手前の kmemo が残る**。
実測でリトライ4回ぶんの孤児が残っていた。

## Decision

**実行フェーズの失敗でも、原因が「利用者が直せる状態」なら `*KFTLInputError` にする。**
判定の境界は「サーバが壊れているか」ではなく「呼び出し側が動けるか」。

- `~~` の対象が見つからない3種 → `NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE`
  （TS 側が同じ検査で使っている既存キー。Go だけが参照していなかった）
- MiReKyou の書き込み先が無い → 新設の `KFTL_MI_REKYOU_NO_WRITE_REP_MESSAGE_TITLE`
- `AddMiReKyouInfo` の失敗（DBの書き込みエラー）→ **`fmt.Errorf` のまま**。これは本物のサーバ障害

**原因の文面を利用者へ返さない。** `MessageID` が空だと
`handle_submit_kftl_text.go` の `formatKFTLInputErrorMessage` が `Cause` の英文をそのまま応答へ載せる。
書き込み先が無いときの `Cause` には利用者IDと端末名が入っているので、多言語キーの設定は必須
（[ADR-0046](0046-redact-environment-specific-strings.md)）。

あわせて **MCP からも `idempotency_key` を送る**。受け口はサーバに前からあり Wear OS は送っていたのに、
MCP だけ送っていなかった。同じ鍵で再送すれば孤児が積まない。

## Rejected alternatives

- **実行フェーズ全体を入力エラー扱いにする** — DB の書き込み失敗や rep の I/O エラーまで
  HTTP 400 になる。「あなたの入力が悪い」と言われた利用者は直しようがない。
  ADR-0080 の「既定はサーバ障害」を崩さず、**名指しした分だけ**入力エラーへ倒す
- **`~~` を使えるアカウントかどうかを事前フェーズで検査する** — Apply フェーズは
  リポジトリを見ない設計（1バイトも書かずに全行を評価する）。ここで rep を見に行くと
  「まだ書いていない」の保証が壊れる。加えて、書き込み先の有無は
  `Repositories` を解決した後にしか分からない
- **`claude` アカウントに `mirekyou` rep を足して終わりにする** — 症状は消えるが、
  次に同じ状況（新しい rep 種別を持たない既存の設定DB）へ当たった利用者がまた5回試すことになる。
  設定の不足は**設定の不足として名乗るべき**
- **`~~` を「既存レコードもタスク化できる」ように拡張する** — レビューはツール説明の
  「`~~ to task an existing record`」を読んでそう期待していたが、対象IDは
  `ctx.ThisStatementLineTargetID` ＝**同じ送信テキストの直前の行が採番したUUID**で、
  既存レコードを指す構文が無い。拡張ではなく**説明のほうが間違っていた**ので説明を直した
- **KFTL に `dry_run` を足す** — 解析だけ走らせる口がサーバに無く、`kftl_factory` の副作用と
  分離する必要がある。今回の落差（行番号が出ない）は型を変えるだけで消えるので、まずそちらを直す

## Consequences

- `~~` の失敗が **HTTP 500 → 行番号つき HTTP 400** になる。応答本文は従来どおり `errors` 配列で判定できる
- i18n が 924 → **925キー**（7言語 × `src/locales/` と `src/server/gkill/api/embed/i18n/locales/` の2箇所）。
  embed 側へコピーし忘れると `MustLocalizeMessage` が panic する
- `gkill_submit_kftl` に任意の `idempotency_key` が増える。省略時の挙動は従来どおり（毎回書く）
- MCP 経由の KFTL は依然として `create_app="gkill_kftl"` / `create_device=<サーバのdevice>` で、
  手打ちのメモ帳と区別が付かない。**区別する欄は足していない**（ワイヤ契約の変更になるため）。
  代わりに `create_apps` の説明から「これが MCP で作った記録の探し方」という誤った案内を消した

## Evidence

- `kftl_mirekyou.go` の `newKFTLInputError` 呼び出しは **0箇所**、他の kftl ファイルは合計22箇所
- `src/client` に `submit_kftl_text` の文字列は**1件も無い**。この API を呼ぶのは MCP と Wear OS だけで、
  Web は TS 側で解析して `add_*` へ fan-out し、失敗時は `discard_tx` で巻き戻す（＝孤児が出ない）
- MCP のアクセスログに `gkill_submit_kftl` の ERR000351 が4回並び、
  同じ時刻の kmemo が4件、書き込み先のDBに残っていた
- `create_device` の実測値は `mcp` ではなく**サーバのデバイス名**だった
  （`gkill_add_*` は `mcp`。KFTL だけ経路が違う）

## Related tests

- `src/server/gkill/api/kftl/kftl_mirekyou_test.go`
  - `TestDoRequest_MiReKyouWithoutWriteRepIsError`（`KFTL_MI_REKYOU_NO_WRITE_REP_MESSAGE_TITLE` で返ること）
  - `TestDoRequest_MiReKyouWithPrototypeOnlyTargetIsError`（`NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE`）
  - `assertKFTLInputError`（MessageID が空でない＝`Cause` の英文が漏れない、も同時に見る）
