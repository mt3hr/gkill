# ADR-0091: 更新は「変わる欄が1つも無い」なら書かずに断る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-25 |
| Sources | 2026-08-25 の実利用レビュー（`gkill_update_kmemo({id})` が no-op の版を追記した） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/write-handlers.mjs`（`runUpdate` の空パッチ判定と `nextUpdateTime`） |

## Context

`gkill_update_kmemo` に `id` だけを渡すと、エラーにならずに**内容がまったく同じ版**が1つ積まれていた。
gkill は追記型なので、更新は「新しい版の追加」であり、消せない。レビューが1件のメモを試した結果、
履歴は6版になり、そのうち1版は何も変えていないものだった。

同じファイルの隣にある `softDeleteOne` は、逆にこれを防いでいる。削除済みをもう一度削除しようとすると
`Entity is already deleted`、生きているものを復活させようとすると `Entity is already active` で止まり、
コメントにも「無意味な版を積まない」と書いてある。**取得 → patch → 更新という手順まで同じ双子の関数で、
片方にだけガードが無かった**。

もう1つ、同じ関数に別の欠落があった。`softDeleteOne` は `update_time` を `nextUpdateTime` に通して
「現在値より必ず後」を保証しているのに、`runUpdate` は素の `new Date()` を使っていた。
`UPDATE_TIME` は1秒解像度で保存され、最新版の判定は厳密な `After` なので、
**同じ秒の中で2回更新すると2回目が最新版と見なされない**。エラーは出ず、更新が消える。

## Decision

**`runUpdate` は、patch 対象の欄が1つも渡されていなければ、書かずにエラーを返す。**
判定は `undefined` のみを「未指定」とみなす（`null` は「消す」という意思表示を持つ欄があるため）。
併せて `update_time` は `nextUpdateTime` を通す。

## Rejected alternatives

- **JSON Schema の `anyOf` で「id 以外に最低1つ」を宣言し、サーバ到達前に止める** — 送信前に弾けるのは
  利点だが、`anyOf` と `additionalProperties:false` の組み合わせはクライアント実装によって
  解釈が割れ、正しい呼び出しまで弾かれることがある。**スキーマは案内、ハンドラが正本**という
  delete / restore と同じ形に揃えた（スキーマ側へ足すのは多重防御として後からできる）
- **API 往復の前（`normalizeEntityArgs` の末尾）で弾く** — 無駄な `/api/get_*` を1回省けるが、
  「対象が存在しない」より先に「欄が無い」を返すことになる。**存在しない id を渡したときに
  まず知りたいのは id のほう**なので、取得の後に置いた（`softDeleteOne` と同じ位置でもある）
- **黙って何もせず成功を返す（冪等な no-op にする）** — 呼び出し側は「更新できた」と読む。
  実際には何も起きていないので、次に読んだときに変わっていない理由が分からない
- **Go の `usecase.UpdateXxx` で既存版と内容比較して弾く** — Web UI・Wear OS・同期にも効くので
  一見こちらが本筋だが、9型ぶんの比較を手書きすることになり、比較漏れ（新しい欄を足したときに
  比較へ足し忘れる）が静かに no-op を通す。MCP 側は `UPDATE_TARGETS` の `patchFields` という
  **既にある1つの表**で判定できるので、まずこちらへ置いた

## Consequences

- `gkill_update_*` 9本すべてが「欄を1つも渡さない呼び出し」を拒否する。ガードは `runUpdate` の
  1箇所なので、型を足しても自動的に効く
- **`null` を「消す」に使う欄（`timeis.end_time`）は、`null` だけを渡した更新が通る。** これは意図どおりで、
  判定を `!= null` へ書き換えると打刻の再開ができなくなる
- 同じ秒に2回更新しても、2回目が消えなくなる。この壊れ方は**エラーも警告も出ない**ので、
  戻すと気づけない

## Evidence

- レビューが作った1件の kmemo の履歴が6版で、うち1版が内容・`related_time` ともに同一の no-op
- `softDeleteOne` は同じ形のガードを持っており（`already deleted` / `already active`）、
  ツール説明にも露出済み。`runUpdate` だけが持っていなかった
- `nextUpdateTime` の doc コメントは、delete → restore を同じ秒に行うと復活が黙って無視される
  という事故を記録している。`runUpdate` はその防御を通っていなかった

## Related tests

- `src/mcp/__tests__/write-handlers.test.mjs`（空パッチの拒否、「見つからない」が先に出ること、同じ秒の更新でも `update_time` が前へ進むこと）
