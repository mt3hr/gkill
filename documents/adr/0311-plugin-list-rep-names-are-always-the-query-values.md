# ADR-0311: プラグイン一覧の rep_names は常に「query.reps に渡せる値」

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（「`plugin_list` の `rep_name:"ArchivedGit"` は `query.reps` に使えない（名前が罠）」）。ADR-0308（プラグインは複数の rep 名を申告できる） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/req_res/get_plugin_list_response.go`（`RepNames`）/ `src/server/gkill/api/gkill_server_api/handle_get_plugin_list.go` / `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go`（`pluginManifestRepNameHint`） |

## Context

manifest の `rep_name` はプラグインの名札で、zip の Git リポジトリを束ねるプラグインの Kyou はリポジトリ名を
名乗る（ADR-0308）。`get_plugin_list` の `rep_names` は「申告があるときだけ」載せる `omitempty` で、索引が
未構築で申告が空だとキーごと消え、応答には検索に使えない `"ArchivedGit"` しか見えなかった。
`query.reps` にそれを渡しても Go は「未知の rep」としか言わず、綴りは合っているので直しようが無かった。

## Decision

`rep_names` は Kyou を出すプラグインでは**常に**載せ、値は「そのプラグインの記録が名乗る rep 名」＝
`query.reps` に渡せる値にする（申告があればその値、索引未構築なら `[]`、申告しない・取得に失敗したプラグインは
`[manifest の rep_name]`）。Kyou を出さないプラグインは `null`。Go の rep 未知警告は、値が emits_kyou な
プラグインの manifest 名に一致したとき「それは名札で、使う値は `rep_names` にある」と名指しする。

## Rejected alternatives

- **manifest の `rep_name` も申告値に含めて検索できるようにする** — Kyou の `rep_name` を書き換えることになり、
  名前で rep を選ぶ Web の設定と食い違う（ADR-0308 で否決済みの向き）。
- **`rep_names` は据え置き、説明文だけ直す** — 索引未構築のときに応答から値が消える構造が残る。
- **`rep_name_searchable` のような3つ目の語彙を足す** — manifest の語彙（`emits_kyou` / `provides`）に
  無い語を増やさない（ADR-0608）。

## Consequences

- `rep_names` を見れば `query.reps` の値が分かる。`[]` は「索引未構築でまだ値が無い」。
- 単純なプラグイン（申告なし）でも `rep_names` が `[manifest の rep_name]` として載る。

## Evidence

- 報告値: `query.reps:["ArchivedGit"]` が 0 件・警告は汎用文。
- 修正後: 申告しないテストプラグインで `rep_names:["KyouOnlyTestRep"]`、GPS 専用は `null`。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_plugin_list_test.go`（`TestHandleGetPluginList_EmitsKyouAndProvides`）
- `src/server/gkill/api/req_res/req_res_test.go`（`PluginInfo` の `rep_names`）
