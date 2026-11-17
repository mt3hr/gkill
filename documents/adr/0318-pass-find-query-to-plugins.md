# ADR-0318: プラグインへ検索条件の全体（FindQuery）を find_query で渡し、旧来の欄も残す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-17 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md` / `src/plugins/README.md`「`sdk.Query` フィールド」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_plugin/plugin_protocol.go`（`PluginQuery.FindQuery` / `WordsSkipIDMatch`） / `src/server/gkill/dao/reps/plugin_repository_impl.go`（`findQueryToPluginQuery` / `indexFetchAll` / `pluginKyouMatchesQuery`） / `src/server/gkill/plugin/sdk/types.go`（`Query` / `IDMatcher`） |

## Context

画面は検索条件に `include_end_timeis` を付けて送り（既定は true）、サーバーは `FindQuery.IncludeEndTimeIs` で受け取って、プラグイン本体（`pluginRepositoryImpl.FindKyous`）まで渡していた。
ところがプラグインへ送る問い合わせを作る `findQueryToPluginQuery` は、FindQuery を縮めた `PluginQuery`（単語・タグ・期間・`only_latest_data`）しか送っていなかった。
IncludeEndTimeIs・IDs・Reps など残りの条件は**途中で抜け落ち**、プロトコルにも SDK の `sdk.Query` にも受け取る欄が無かった。

表に出たのは `gkill_plugin_vrchat`（[ADR-0317](0317-plugin-mixed-typed-kinds-and-playing-time.md)）だった。TimeIs を返すプラグインは、native と同じく「終了の Kyou（`timeis_end`）は IncludeEndTimeIs のときだけ」を守りたくても判断の材料が無い。
かといって常に終了の Kyou を返すと、既定（false）の検索でも TimeIs が2件ずつ出る。本体も `timeis_end` を IncludeEndTimeIs で落としていなかった（ADR-0317 の Consequences）。

同じ欠けは他の条件にもあった:

- **IDs**: 本文やタグに当たった記録を ID で引き直す検索（`find_filter.go` の matchTextFindByIDQuery）で、プラグインは毎回全件を返し、本体が ID で捨てていた。
- **WordsSkipIDMatch**（`json:"-"` の内部用の欄）: 除外語を肯定語として引き直す検索で運ばれず、SDK の判定が ID の前方一致も見てしまう。ID が除外語で始まるプラグインの記録まで除外されていた。

## Decision

- `PluginQuery` に `find_query`（gkill が受けた `find.FindQuery` の浅いコピー。`UpdateCache` は落とす）と `words_skip_id_match` を足す。旧来の欄（words〜limit）はそのまま埋め続ける。
- SDK の `Query` に `FindQuery` と、よく使うものの写し `IDs` / `Reps` / `IncludeEndTimeIs` / `WordsSkipIDMatch` を足す。判定の手助けとして `MatchID` / `IDMatcher` / `MatchRep` を用意し、`Matcher` は WordsSkipIDMatch を効かせる。
- 本体は返ってきた Kyou を **期間・IDs・IncludeEndTimeIs で必ず絞り直す**（`pluginKyouMatchesQuery`）。プラグインがこれらを見なくても結果は変わらず、読む量が増えるだけにしておく。プラグイン本体の経路と型別アダプタの経路の両方がここを通る。
- 型別索引を作る問い合わせ（`indexFetchAll`）は、**IncludeEndTimeIs だけを立てた絞り込みなしの FindQuery** を送る。索引は全てのアダプタの材料なので、IDs や Reps を尊重するプラグインに絞り込みを渡すと索引が欠ける。終了の Kyou も索引に持たせ、返すかどうかは問い合わせごとに決める。
- `protocol_version` は `"1"` のまま。SDK も本体も知らないキーを無視するので、古いプラグイン・古い gkill とはどちら向きにも混ぜて動く（古い gkill からは `FindQuery` が nil）。
- SDK が読む1リクエストの上限を 1MB から 16MB に上げる（IDs は本体が4000件ずつに分けて送るので数百 KB に収まるが、余裕を持たせる）。
- 同梱プラグインは IDs で読む量を減らす（archived_git_commit_log は Reps も）。gkill_plugin_vrchat は IncludeEndTimeIs のとき終了の Kyou も返す。

## Rejected alternatives

- **PluginQuery に FindQuery の欄を1つずつ足す（IncludeEndTimeIs だけ等）** — 今回欲しいのは IncludeEndTimeIs と IDs だが、FindQuery に欄が増えるたびに、プロトコル・SDK の受け口・SDK の Query の3か所を直すことになる。抜け落ちに気づくのは今回のように、誰かが「画面から送っているのに届かない」と調べたときだけ。全体を1つの欄で運べば、プラグインは要るものを自分で読める。

- **FindQuery を query の中に直接混ぜる（`query` を FindQuery の JSON にする）** — `words` / `tags` / `calendar_*` など同じ名前の欄が両方にあり、`hide_tags` と `not_tags` のように名前の違う同じ意味の欄もある。旧来の欄を残したまま混ぜると、どちらが正か曖昧になる。別の欄（`find_query`）に分けた。

- **protocol_version を `"2"` に上げる** — 本体は manifest の値が `"1"` 以外のプラグインを読み込まない（`plugin_manager.go`）。上げると古い SDK で作ったプラグイン（gkill の外にあるものを含む）が全部読み込まれなくなる。追加の欄は無視されるので上げる必要が無い。

- **本体は絞り直さず、プラグインに任せる** — 古い SDK のプラグインは新しい欄を読めないので、ID 指定の検索に全件が混ざり、`timeis_end` が IncludeEndTimeIs と関係なく出る。本体の絞り直しは安い（返ってきた Kyou への比較だけ）。

## Consequences

- `sdk.Query.IsDeleted`（「削除済み**だけ**」）と `FindQuery.IncludeDeletedData`（「削除済みも含める」）は意味が違う。写しの欄は作っていない。IsDeleted は今までどおり本体から常に false。
- `FindQuery` は gkill 本体の型なので、gkill が欄を足すとプラグインにもそのまま届く。プラグインは読むだけにし、書き換えた値を返さないこと。
- gkill の外にあるプラグインは、SDK を新しい版に上げるまで新しい欄を読めない（動きは今までどおり）。
- 1件の TimeIs から Kyou が2つ（開始と終了）返るので、プラグインの LIMIT は2つを合わせた数で掛ける。終了の Kyou は開始と同じ版（UpdateTime）にすること。版が違うと索引は新しいほうだけを残す。

## Evidence

実測なし — コードの読み取りによる判断。届くことと絞り直しは下のテストで固定した。

## Related tests

- `src/server/gkill/dao/reps/plugin_repository_impl_test.go`（`TestPluginRepository_FindQueryReachesPlugin` / `TestPluginRepository_IndexFetchAllSendsUnfilteredFindQuery` / `TestPluginRepository_FindKyousFiltersTimeIsEndByIncludeEndTimeIs` / `TestPluginRepository_FindKyousAppliesIDFilter`）
- `src/server/gkill/dao/reps/plugin_typed_adapters_test.go`（`TestPluginTimeIsAdapter_FindKyousIncludesEndOnlyWhenAsked`）
- `src/server/gkill/plugin/sdk/find_query_test.go`（`TestRunLoop_FindKyousReceivesFindQuery` / `TestRunLoop_FindKyousWithoutFindQuery` / `TestQueryMatchIDAndRep` / `TestQueryMatchTextSkipIDMatch` / `TestRunLoop_GetKyouFallbackAsksByID`）
- `src/plugins/gkill_plugin_vrchat/end_timeis_test.go`
