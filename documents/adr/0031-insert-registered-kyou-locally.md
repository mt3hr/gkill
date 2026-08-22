# ADR-0031: 記録の追加は列を再検索せず、その1件をクライアントで判定して差し込む

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-16 |
| Sources | `c1fbc9de` / `6c4bdfd9` / `6ee545b3` / `.claude/skills/gkill-client-columns/SKILL.md`「記録の追加は再検索せず、その1件を列へ差し込む」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/kyou-local-insert.ts` |

## Context

記録を1件追加するたびに、開いている全列の再検索が走っていた。実データは30万件規模なので、メモを1行書くたびに全件検索が列の数だけ飛ぶ。

## Decision

追加系ビューは `registered_kyou` だけを出し、受け口（`use-registered-kyou-local-insert.ts`）が**その1件を列の条件に照らして in-place で差し込む**。`requested_reload_list` は「サーバが Kyou を返さなかった」ときのフォールバックへ退く。

判定と整列は純関数 `classes/kyou-local-insert.ts` にあり、意味論は `server/gkill/api/find_filter.go` の写し。

## Rejected alternatives

- **追加のたびに再検索する（改修前）** — 30万件規模の検索が列の数ぶん飛ぶ。

- **サーバに「この Kyou はこのクエリに一致するか」を問い合わせる** — 往復が増えるうえ、`/api/get_kyou` は `FindQuery` を受けない。エンドポイントを足すと、判定の実装がサーバ・クライアントの2箇所になる。

- **クライアントで全条件を判定できることにする** — **できない条件がある。** 本文検索・TimeIs・地図・plaing・画像のみ・`rep_types`（rep_name→rep_type の写像がクライアントに無い）。**これらを使う列だけ従来どおり再検索する。** 判定を足すときは「判定できないものを判定できると誤って宣言しない」ことがすべてで、誤ると例外もエラーも出ずに黙って一致しない行が出る。

- **差し込み先の配列を作り直す（copy-on-write）** — `focused_kyous_list` は `match_kyous_list[focused_column_index]` への**エイリアス**なので、参照ごと差し替えると件数カレンダーや Dnote と縁が切れる。30万件のコピーも避けられる。**in-place `splice`** にする。

- **`add_*` の応答をそのまま列へ入れる** — 応答は **hydrate を通っていない生 JSON**（`related_time` が文字列、`clone()` も無い）。受け口で必ず実体化する。

## Consequences

並び順の規則が2つ要る。非mi は `RelatedTime` 降順（**`.Unix()` 相当に秒へ切り捨ててから**比較。ミリ秒のままだと同一秒の隣接行で位置がずれる）、mi はソート基準の時刻の昇順で未設定は末尾。

mi の「未設定」は `typed_mi` ではなく **`data_type` の接尾辞**で判定する。一覧の既存行は `typed_mi` が未ロードなので、`typed_mi` を要求する比較子は既存行に対して動かない。

Dnote は命令的 reload なので配列を触るだけでは追随しない。明示的に呼び直すこと。

KFTL は送信全体を tx で包むが、**tx 中の `add_*` は `added_kyou` を返せない**。リクエストクラスが id だけ積み（`get_result_kyou_ids()`）、`commit_tx` の**あと**に `get_kyou` で引いてから emit する。

判定の意味論が Go 側と**2箇所**になるので、`kyou-local-insert-mi-parity.test.ts` を `find_filter_mi_test.go` と対にしてある。

## Evidence

実測なし — 構造からの判断（30万件規模の再検索を、1件の判定に置き換える）。

## Related tests

- `src/client/__tests__/unit/classes/kyou-local-insert.test.ts`
- `src/client/__tests__/unit/classes/kyou-local-insert-mi-parity.test.ts`（Go の `find_filter_mi_test.go` と対）
- `src/client/__tests__/unit/composables/registered-kyou-local-insert.test.ts`（rykv・mi 両方でパラメタライズ）
- `src/client/__tests__/unit/kftl/kftl-submit-emits.test.ts`
- `src/server/gkill/api/find_filter_mi_test.go`
