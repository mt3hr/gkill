# ADR-0003: タグ絞り込みの取得は2経路を持ち、切り替えはタグ名の個数（閾値32）で決める

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-19 |
| Sources | `f2c4f280` / `a6e4967c` / `bb364253` / `CLAUDE.md`「タグ絞り込みの取得は2経路。切り替えは『タグ名の個数』で決める」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_filter.go` |

## Context

本番データのプロファイル（2026-08-19）で、rykv の検索の**実質CPUの44%＝40.4秒がタグ名の絞り込み**だった。内訳は cached SQLite tag rep が24.7秒、プラグインのタグアダプタが15.7秒で、うち sqlite の `_lowerFunc` だけで17.1秒。

tag rep のワード検索は `findWordUseLike=false` / `ignoreCase=true` なので `LOWER(TAG) = LOWER(?) OR LOWER(ID) = LOWER(?)` を出す。**列に関数がかかるため索引が効かず、全行に `LOWER()` を適用**したうえで、それをクエリのタグ名の数だけ繰り返す（O(行数 × 名前の数)）。利用者はサイドバーで多数のタグにチェックが入る構成なので、名前の数がそのまま倍率になっていた。

## Decision

タグ絞り込みの取得を `collectTagsForFilter` に統合し、**2経路を持たせて「タグ名の個数」で切り替える**。閾値は `maxTagNamesForSQLFilter = 32`（クエリのタグ名と強制非表示タグ名の**合計**で見る）。

「タグ無し」仮想タグ（`NoTags`）を使う検索では、名前の個数によらず Go 側で照合する。

## Rejected alternatives

- **SQL 側（`LOWER(TAG) = LOWER(?)`）に一本化する** — 名前の数に比例する。実測で100名前のとき724.7ms。利用者のサイドバー構成では名前が数十になるのが普通なので、そのまま倍率になる。

- **「全部取って Go で照合」に一本化する（最初に試した案）** — **ベンチで否決した。** 名前が少ないうちは全タグの実体化（`reps.Tag` は240B ＋ 文字列10本）のほうが高い。1名前のとき SQL 12.5ms / 91確保 に対し Go 141.8ms / 580,074確保。

- **行数に応じて閾値を動かす** — 不要。SQL側は名前の数に比例、Go側はほぼ一定で、**どちらも行数には比例する**ので、交差する「名前の個数」は行数によらずほぼ一定になる。よって固定の閾値でよい。

- **照合を SQL と Go で別の意味論にする（片方だけ大小区別、など）** — タグの個数によって検索結果が変わるという静かな壊れ方になる。`strings.EqualFold` の完全一致・大小無視で、`filterTagsKyous` のAND分岐と同じ意味論に揃える。SQL は `TAG` 列だけでなく `ID` 列とも突き合わせていたので、そこも Go 側へ写してある。

- **`NoTags` を使う検索でも個数で切り替える** — 損。`RelatedTagIDs`（タグが1つでも付いているIDの集合）のために結局は全タグを取るので、**そこから名前を拾うぶんはタダになる**。以前は「全タグの取得」「クエリのタグ名で絞る検索」「非表示タグ名で絞る検索」を別々に投げていた。

## Consequences

閾値の 32 は交差点（約30）より**やや上**に置いてある。確保の少ないSQL側へ寄せる意図。

**2経路が同じ結果を出すことをテストで固定してある**（`find_filter_test.go`）。ずれると「タグの個数によって検索結果が変わる」という、エラーも警告も出ない壊れ方になる。強制非表示タグについても2経路の一致を `TestHandleGetKyous_HideTagsBothPaths` が固定する。

`RelatedTagIDs` の読み手は `NoTags` 分岐しか無いので、`NoTags` を使わない検索では作らない（→ ADR-0004）。走らせ忘れると**全件が「タグなし」扱い**になり、タグの付いた記録まで返る。

閾値を動かすときは `dao/reps/tag_find_bench_test.go` の実測表を取り直すこと。**`ns/op` で判断してはいけない**（→ ADR-0008）。

## Evidence

2万タグでの実測（`dao/reps/tag_find_bench_test.go` に表を残してある）:

| タグ名の数 | SQLで絞る | 全部取ってGoで照合 |
|---|---|---|
| 1 | 12.5ms / 91確保 | 141.8ms / 580,074確保 |
| 10 | 69.2ms / 425確保 | 153.2ms / 580,074確保 |
| 30 | 197.6ms / 1,148確保 | 180.5ms / 580,074確保 ← 交差 |
| 100 | 724.7ms / 3,684確保 | 257.0ms / 580,076確保 |

統合後、rykv の検索の実質CPUは 90.9秒 → 52.3秒（強制非表示タグの取得 11.2秒ぶんの削減を含む）。

判断は `allocs/op` と `B/op` で行っている。`ns/op` はこのマシンで倍近くぶれる。

## Related tests

- `src/server/gkill/api/find_filter_test.go`（2経路の結果一致を固定）
- `src/server/gkill/dao/reps/tag_find_bench_test.go`（実測表）
- `src/server/gkill/api/gkill_server_api/get_kyous_tag_filter_test.go`
- `src/server/gkill/api/find_filter_tags_test.go`
- `src/server/gkill/api/find_filter_timeis_test.go`（TimeIs 側も同じ2経路・同じ閾値で対称化されている）
