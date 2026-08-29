# ADR-0109: hide_tags はタグ絞り込み(tags)の有無と独立に適用する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-23 |
| Sources | `df6cd60d` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_filter.go` |

## Context

`hide_tags` は「このタグが付いた記録を消す。ただし同じ名前が `tags` にも入っていれば消さない」
という除外リストだが、実装は**タグ絞り込み（`tags` / `timeis_tags`）を使うときにしか**
収集も適用もしていなかった（`needHideTags := usesTagFilter && ...` と `Tags == nil` の早期return）。

Webクライアントは `hide_tags` を ApplicationConfig の強制非表示タグ（`is_force_hide`）から
組み立てて**常に `tags` と同時に送る**ため、このゲートは一度も露呈しなかった。
一方MCPスキーマは `hide_tags` を "Explicit tag exclusion list" と単独で使える体で宣言しており、
1年分の外部監査で「`tags` 未指定 + `hide_tags` 指定が、エラーも警告も出さずに
1件も除外しない」ことが実測で確定した（total_count 2,289 が hide_tags を足しても不変、
先頭行が除外対象タグ持ちのまま）。

## Decision

- `hide_tags` の収集ゲートを `len(HideTags) != 0` だけにする（`tags` 非依存）
- 適用を `filterTagsKyous` の中から独立ステップ **`filterHideTagsKyous`** へ切り出し、
  `Tags == nil` でも走らせる。「同名が `tags` にあれば消さない」意味論は
  集合を作る `getMatchHideTagsWhenUnchecked` 側がそのまま担う
  （`checkedTagNames` が nil なら「チェック済みタグ無し」＝全 hide_tags 有効）
- TimeIs 側も対称に、`TimeIsTags == nil` の分岐で非表示集合を適用する
  （以前ここにあった delete は集合が常に空で発火しない死にコードだった）
- クライアントの局所挿入ミラー（`kyou-local-insert.ts` の `is_hidden_by_hide_tags`）も
  対で変更する（`query.tags === null` ゲートを外す）。放置すると
  「追加直後は見えるが再検索で消える」静かなズレになる

## Rejected alternatives

- **現状維持（tags 併用時のみ有効）** — MCPスキーマの宣言と実装が食い違ったまま、
  AIクライアントの hide_tags 単独指定が黙って無視され続ける。「効かないフィルタ」は
  0件系の静かな壊れ方の中でも最悪の形（結果が*減らない*ので検算でも気付きにくい）。
- **Node(MCP)側で hide_tags を可視タグ全列挙の tags へ変換する** — タグ数依存で
  巨大クエリになり、「タグ無し」記録の扱い（NoTags 仮想タグ）と衝突する。
- **単独指定をエラーにする** — 宣言済みの契約を後から拒否に変える理由が無い。
  有効化のほうが意図に合致し、Webへの影響も実質無い（常に tags と併送のため）。

## Consequences

- Web の挙動は実質不変（`hide_tags` は常に `tags` と併送されるため、従来から適用されていた
  経路がそのまま通る）。変わるのは `tags` 未指定の検索＝MCP経由と、
  plaing の既定クエリのような hide_tags 空のケース（空なら no-op で不変）
- 旧挙動を明示的に固定していた `TestFilterTagsTimeIs_NoTagFilter_DoesNotApplyHideTags` は
  期待を反転して `_AppliesHideTags` に改名した

## Evidence

- 監査実測: `{calendar 8/13-8/22, rep_types:["timeis"]}` の total_count 2,289 に
  `hide_tags:["autolog_device"]` を足しても 2,289 のまま、kyous[0] は
  `tags:["autolog_device"]` だった（エラー・メッセージ無し）
- 改修後: tags 無し × SQL経路/Go照合経路（名前33個で閾値超え）の両方で除外が効くことを
  E2E で固定

## Related tests

- `src/server/gkill/api/find_filter_tags_test.go`
- `src/server/gkill/api/find_filter_timeis_test.go`
- `src/server/gkill/api/gkill_server_api/get_kyous_tag_filter_test.go`
- `src/client/__tests__/unit/classes/kyou-local-insert.test.ts`
