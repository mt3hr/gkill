# ADR-0317: 1本のプラグインが複数の型別の種類を返してよく、プラグインの TimeIs も playing_time で絞る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-17 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md`「プラグインの型別/付随データ」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/plugin_typed_adapters.go`（`pluginTimeIsPlayingAt`） / `src/server/gkill/api/gkill_plugin/plugin_manifest.go`（`Provides` のコメント） / `src/server/gkill/dao/plugin_manager.go`（data_type と provides の食い違いの警告） |

## Context

VRChat のログを読むプラグイン（`gkill_plugin_vrchat`）は、ワールドの滞在を TimeIs と URLog、同じ部屋にいた人を TimeIs、出来事を Kmemo で返す。
それまでの同梱プラグインは型別データを1種類しか返しておらず（fitbit は KC とタグ、archived_git_commit_log は GitCommitLog）、
manifest の `data_type` は1つしか書けない。manifest のコメントと `plugin-system.md` には「provides に型別データを書くなら data_type はその種別で始めること」とあり、1本1種類が前提に読めた。

調べると、型別の振り分けはすべて **Kyou ごとの `data_type` と `typed.*`** で行われていた:

- 索引 `PluginTypedIndex.applyTypedData` は Kyou ごとに型別データを1つ採る（manifest は見ない）
- クライアントは Kyou ごとの `data_type` の接頭辞で型別ビューを出し分ける（`kyou.ts`）
- MCP の payload も Kyou ごとの `data_type` で決まる（`kmemo` と `urlog` は完全一致、`timeis_` は接頭辞）

manifest の `data_type` を読むのは、読み込み時の警告（`plugin_manager.go`）と MCP の案内（既知の data_type の集合・空の索引のヒント・`get_plugin_list` の表示）だけだった。

同時に、**プラグインの TimeIs が `playing_time`（その時刻に実行中か）で絞られていない**ことが分かった。
native は SQL で `START_TIME <= t AND (t <= END_TIME OR END_TIME IS NULL)` を掛けるが（`time_is_repository_sqlite3_impl.go`）、
型別アダプタ `pluginAdapterBase.findKyous` は期間と ID しか見ていなかった。`playing_time` の検索は `find_filter.go` が TimeIs の rep だけに投げるので、
**プラグインの TimeIs が全件「実行中」として返り、どの Kyou にもその時刻の TimeIs として付く**。
TimeIs を provides に書いたプラグインがそれまで無かったので表に出ていなかった。例外もエラーも出ない壊れ方。

## Decision

- 1本のプラグインが複数の型別の種類を返してよい。各 Kyou の `data_type` はその種類の値（`timeis_start` / `urlog` / `kmemo` 等）にする。manifest の `data_type` は返す種類のうち**代表の1つ**にする。
- 読み込み時の警告は、型別の種類ごとではなく「manifest の `data_type` が provides のどの種類の接頭辞とも合わない」ときだけ1回出す。
- 型別アダプタの `findKyous` は `query.PlayingTime` があるとき、TimeIs を持ち、その時刻に実行中（両端を含む・終了なしは開始以降ずっと）のレコードだけを返す（`pluginTimeIsPlayingAt`）。

## Rejected alternatives

- **種類ごとにプラグインを分ける（VRChat の TimeIs 用・URLog 用・Kmemo 用の3本）** — 同じログを3本が別々に読み、別々のキャッシュを持つ。滞在の TimeIs と URLog は同じ解析結果から作るので、分けると解析の食い違いが起きうる。本体側は Kyou ごとに振り分けているので、分ける理由が無い。

- **manifest に `data_types`（複数形）を足す** — 本体・SDK・MCP・資料の境界対応表に新しいキーが増えるが、振り分けにはもともと使われていないので、得られるのは警告と案内文の正確さだけ。代表の1つを書く運用で足りる。

- **manifest の `data_type` を一意な名前（`vrchat_mixed` 等）にする** — 警告が出続けるうえ、MCP の「渡せる data_type」の集合に、どの Kyou も持たない値が載り、渡すと0件で警告も出ない（[ADR-0608](0608-plugin-role-is-emits-kyou-and-provides.md) が GPS 専用プラグインで潰したのと同じ形）。

- **プラグインに `playing_time` を渡してプラグイン側で絞らせる** — 型別アダプタは索引から即答し、プラグインへ往復しない約束（[ADR-0302](0302-plugin-provides-typed-index.md)）。索引は TimeIs の開始・終了を持っているので、本体側で native と同じ条件を掛ければ足り、プラグインごとに実装を揃える必要も無い。

## Consequences

- 種類をまたいで同じ ID を使ってはいけない。索引は同じ ID の Kyou を1レコードにまとめるので、TimeIs と Kmemo が同じ ID だと両方の型別データが1レコードに入り、`get_kyou` がどちらを返すかが並び順で決まる。テストで守っていないので、プラグイン側で ID に種類を混ぜて作る。
- `get_plugin_list` / `get_rep_infos` に出る `data_type` は代表の1種類だけ。1本のプラグインの記録だけを引くときは `query.reps` を使う（[ADR-0311](0311-plugin-list-rep-names-are-always-the-query-values.md)）。
- プラグインの TimeIs の `timeis_end` 行は、この ADR の時点では `IncludeEndTimeIs` を見ずに常に出ていた。プラグインには IncludeEndTimeIs が届いていなかった。[ADR-0318](0318-pass-find-query-to-plugins.md) で検索条件の全体を渡し、本体も IncludeEndTimeIs で絞るようにした。`gkill_plugin_vrchat` はそのとき終了の Kyou も返すようになった。
- 画面の TimeIs の「終了」ボタンはプラグインの rep かどうかを見ていないので、終了なしのプラグインの TimeIs で押すと native の版が書かれる（この ADR では直していない）。`gkill_plugin_vrchat` は終了時刻を必ず埋める。

## Evidence

実測なし — コードの読み取りによる判断。振り分けと絞り込みは下のテストで固定した。

## Related tests

- `src/server/gkill/dao/reps/plugin_typed_adapters_test.go`（`TestPluginTypedAdapters_MixedKindsAreRoutedPerKyou` / `TestPluginTimeIsAdapter_FindKyousFiltersByPlayingTime`）
