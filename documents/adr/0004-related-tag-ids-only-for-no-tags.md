# ADR-0004: 全タグ走査（RelatedTagIDs）は「タグ無し」仮想タグを使う検索のときだけ走らせる

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-19 |
| Sources | `b45e6236` / `src/server/gkill/api/find_filter.go` / `.claude/skills/gkill-go-backend/SKILL.md`「タグ絞り込みの取得は2経路」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_filter.go` |

## Context

`find_filter.go` の `getAllTags` は**全repの全タグ**を集めて `RelatedTagIDs`（タグが1つでも付いているIDの集合）を作る。実データでは1検索あたり数十万行になる。

ところが `RelatedTagIDs` の読み手は `filterTagsKyous` / `filterTagsTimeIs` の **`NoTags`（"no tags" 仮想タグ）分岐しか無い**（OR・ANDそれぞれ2箇所）。つまり `NoTags` が条件に入っていなければ、この全走査は結果に一切影響しない。

起動条件が `Tags != nil || (HasTimeIsFilter() && TimeIsTags != nil)` だったので、rykv の既定クエリは `tags` が常に非null（コンストラクタ既定が `[]`）なぶん、**タグを明示的に選んだだけの検索でも毎回まるごと無駄に走っていた**。

## Decision

起動条件を `containsNoTags` ベースへ絞る。`NoTags` を使わない検索では `RelatedTagIDs` を作らない。

強制非表示タグの集合（`getAllHideTagsWhenUnChecked`）は `NoTags` と無関係なので、同じ `if` から切り出して従来の条件のまま残す。

## Rejected alternatives

- **条件はそのままにして `getAllTags` を速くする** — 消しているのは「**結果に影響しないと確認できた仕事**」なので、速くするより走らせないほうが正しい。量に関係なく消してよい。

- **`RelatedTagIDs` を遅延生成にして、読み手が必要になったときに作る** — 読み手は `NoTags` 分岐の4箇所だけで、そこに到達するかどうかは検索の頭で分かる。遅延にすると「作られたかどうか」の状態が増えるだけで、判定の場所が読み手側へ移るぶん見通しが悪くなる。

- **強制非表示タグの取得も同じ条件に載せる** — 別物。`AllHideTagsWhenUnchecked` は `RelatedTagIDs` に依存していないので、載せると `NoTags` を使わない検索で強制非表示タグが効かなくなる。

## Consequences

**走らせ忘れると壊れ方が静か。** `RelatedTagIDs` が空になると**全件が「タグなし」扱い**になり、タグの付いた記録まで返る。エラーも警告も出ない。

そのため両方向を固定してある。実装を `if false` にすると「タグ無しで絞ると、タグの付いていない記録だけが返る」と、既存の `TestHandleGetKyous_TimeIsTagsFilterWorksWithoutKyouTagFilter` が落ちることを確認済み。

タグ条件が空配列（＝フィルタ有効かつ0件指定）のケースも走らなくなる。これは `NoTags` を含まないので正しい。

## Evidence

実測なし — 到達可能性からの判断（読み手が `NoTags` 分岐の4箇所しか無いことをソースで確認した）。削減幅そのものは → ADR-0003 の統合後の数字（実質CPU 90.9秒→52.3秒）に含まれる。

## Related tests

- `src/server/gkill/api/gkill_server_api/get_kyous_tag_filter_test.go`（「タグ名で絞る」「タグ無しで絞る」「両方(OR)」「空配列は0件」を固定）
- `src/server/gkill/api/gkill_server_api/get_kyous_regressions_test.go`（`TestHandleGetKyous_TimeIsTagsFilterWorksWithoutKyouTagFilter`）
