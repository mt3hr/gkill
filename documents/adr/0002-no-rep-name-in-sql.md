# ADR-0002: SQL の WHERE へ rep名の条件を押し込まない（暫定的な否決）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-19 |
| Sources | `06fcdec4` / `src/server/gkill/dao/sqlite3impl/sqlite3impl_util.go` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/sqlite3impl/sqlite3impl_util.go` |

## Context

ADR-0001 で rep名の絞り込みを検索結果側（`filterKyousByRepName`）へ移した。これは「取ってから捨てる」形なので、**「SQL の WHERE で最初から絞れば速いのでは」という提案が必ず出る**。実際 ADR-0001 の起案時にも同じ検討をしている。

この提案が危険なのは、**もっともらしく、かつ壊れ方が静かだから**である。一見すると「取得件数が減る＝速くなる」に見えるが、gkill の SQL 層はその前提を満たしていない。

## Decision

`FindQuery.Reps` を SQL へ降ろさない。`GenerateFindSQLCommon` に `Reps` の参照は0件のまま保つ。

**これは暫定的な否決である。** 必要になったときは実測してから別 ADR として起案する。「なんとなく速そう」を根拠に足さない、というのが決定の実体。

## Rejected alternatives

- **leaf表の WHERE に rep名の条件を足す** — **leaf表に `REP_NAME` 列が無い。** 各 `*_repository_sqlite3_impl.go` は `? AS REP_NAME` でバインド値を別名として返しているだけなので、述語として書く先が存在しない。書けるのは cached 実装の表だけで、そこでも `REP_NAME` は索引されていない。

- **`onlyLatestData` の相関サブクエリにも rep名の条件を足す** — **これをやると壊れる。** 相関サブクエリは「そのIDの最新版」を選ぶためのもので、rep名で絞ると**repごとの最新版が復活し、古い版が現在版として見える**。しかもエラーは出ない。この誘惑が leaf 表の WHERE のすぐ隣にあることが、この決定を明文化しておく最大の理由。

- **cached 表だけで rep名を述語にする** — `filterKyousByRepName` は **`RepName` が空の行を意図的に残す**（ADR-0001 の落とし穴4）。素朴な `REP_NAME IN (...)` はこの行を落とすので、**追加直後の記録が最大1分間一覧から消える**。同じ意味論にするには `REP_NAME IN (...) OR REP_NAME = ''` が要り、索引が効かないぶんの利得はさらに小さくなる。

- **`dao/reps` 層で `query.Reps` を見て絞る** — 別の理由で駄目。→ ADR-0001 の Rejected alternatives。

## Consequences

rep名で絞った検索も、絞る前の件数ぶんは取得してから捨てる。ADR-0001 の測定では、キャッシュを効かせたうえでのこのコストは問題になっていない（問題だったのはキャッシュのバイパスのほうだった）。

**この決定は「rep名を SQL に書くこと」全般の禁止ではない。** 管理系の DML（たとえば cached 実装の rep 単位の部分リビルド `DELETE FROM ... WHERE REP_NAME = ?`）は対象外で、あれは検索の絞り込みではない。禁じているのは**検索の絞り込みを SQL へ降ろすこと**の1点。

再提案するときに必要なもの: 実データのプロファイルで、rep絞り込みのある検索の時間の内訳を示し、「取ってから捨てる」ぶんが実際に効いていることを出すこと。ADR-0001 と同じく `allocs/op`・`B/op`・`EXPLAIN QUERY PLAN` で判断する（→ ADR-0008）。

## Evidence

実測なし — 構造上の背理による否決（述語を書く列が leaf 表に存在しない／相関サブクエリに足すと最新版判定が壊れる／空 `REP_NAME` の意味論が保てない）。

出典コミット `06fcdec4` も同じ判断を「必要なら実測してから別起案」として記録している。

## Related tests

- `src/server/gkill/api/find_kyou_rep_name_filter_test.go`
- `src/server/gkill/dao/reps/cached_find_only_latest_test.go`
- `src/server/gkill/usecase/source_conventions_scan_test.go`（`TestNoRepNameFilterInDaoReps`。`dao/reps` と `dao/sqlite3impl` の両方を走査し、SQL を組み立てる層に `query.Reps` が現れたら落とす）
