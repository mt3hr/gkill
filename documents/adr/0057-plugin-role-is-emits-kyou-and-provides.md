# ADR-0057: プラグインの役割は emits_kyou / provides をそのまま出して表す — capabilities 語彙を新設しない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | 2026-08-24 の実利用フィードバック（プラグイン一覧→Kyou検索→本文取得→Fitbit→GPS の一周）。[ADR-0056](0056-attached-data-reps-are-a-separate-list.md)「Reps へ混ぜない」。[ADR-0023](0023-plugin-emits-kyou-false.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/handle_get_plugin_list.go` / `src/server/gkill/api/req_res/get_plugin_list_response.go` / `src/server/gkill/api/gkill_plugin/plugin_manifest.go`（`NeedsTypedIndex`） |

## Context

プラグインは3種類ある。Kyou を出すだけのもの（ChatGPT / Claude Code / Codex / Claude.ai）、
型別データも出すもの（Fitbit は `provides:["kc","tag"]`）、
GPSログだけを出すもの（Google ロケーション履歴は `provides:["gpslog"]` / `emits_kyou:false`）。

**この違いがどの API からも読めなかった。** `manifest.json` には `provides` と `emits_kyou` が
あるのに、`get_plugin_list` も `get_rep_infos` もそれを返していなかった。
返っていたのは全プラグイン一律の `data_type` / `rep_name` だけで、
説明文は「この値で `query.reps` / `query.rep_types` を絞れ」と案内していた。

実際に AI がその案内どおり `reps:["GoogleLocation"]` を投げ、警告だけが返る、という迷い方をした。
`rep_name` は manifest の必須項目なので値は入っているが、`emits_kyou:false` の
プラグインは Kyou を1件も出さないので `Reps` に載らず、どの検索値とも一致しない。

同じ応答が自己矛盾してもいた。`get_rep_infos` は `PluginReps` を無条件に列挙していたため、
GoogleLocation が `plugins[]`（「`query.reps` / `data_types` で引ける」）と
`attached_data_reps[]`（「これは `query.reps` の値ではない」）の**両方に出ていた**。

さらに `data_types:["google_location_visit"]` は `knownMCPDataTypes` が manifest の
`DataType` を無条件に既知集合へ入れていたため、**警告すら出ずに必ず0件**だった。

型別索引も誤読を生んでいた。`typedIndex` は `len(manifest.Provides) != 0` で作られるが、
`gpslog` に対応するアダプタは存在しない（`plugin_typed_adapters.go` が全 nil を返す）。
索引の材料が1件も無いので `state:"never_built"` / `record_count:0` のまま永久に固定され、
GPS は普通に取れているのに「索引が壊れている＝位置情報は使えない」と読まれた。

## Decision

- `PluginInfo` に **`emits_kyou` と `provides` をそのまま**足す。**新しい語彙は作らない**
- `emits_kyou:false` のプラグインは `get_rep_infos` の `plugins[]` に**載せない**。
  GPS の供給元としては `attached_data_reps[]` の `data_kind:"gpslog"` にだけ残す
- `knownMCPDataTypes` から `emits_kyou:false` のプラグインの `DataType` を外し、
  rep名 / data_type がそれに一致したときは汎用文ではなく
  「そのプラグインは Kyou を出さない（provides: …）。`get_gps_log` で読め」と**名指しする**
- `PluginManifest.NeedsTypedIndex()`（`gpslog` 以外の provides が1つでもあるか）を新設し、
  **GPS専用プラグインには型別索引を作らない**。`typed_index` はキーごと消える
- 代わりに `gps_index`（`point_count` / `oldest` / `newest` / `fetched_at`）を別枠で足す。
  統計は GPSLog アダプタが取得のたびにプラグインリポジトリへ預け、一覧APIは非ブロッキングで読む

## Rejected alternatives

- **`capabilities:["kyou","gps_log"]` / `kyou_searchable` / `gps_searchable` を新設する** —
  フィードバックの提案はこれだったが、**同じ意味の語彙が既に manifest 側に2つある**
  （`provides` の11値と `emits_kyou`）。3つ目を作ると「どれが正本か」と
  「どれとどれが同期していなければならないか」が増えるだけで、
  静かにズレる面がもう1つ生まれる。`provides` は `FindQuery.RepTypes` と
  語彙を揃えてあるという既存の資産も捨てることになる
- **`plugins[]` に残して説明文で「これは検索できない」と注意書きする** —
  ADR-0056 が `rep_type:"tag"` で却下したのと同じ形。
  「渡せる値の表」に渡せない値が載っていれば、呼び出し側は素直に渡す。
  エラーも警告も出ずに0件になるのが最悪の帰結で、注意書きは防御にならない
- **`data_type` から役割を推測する** — ADR-0023 が既に否決している。
  推測を誤ると作者の記録が黙って消えるので、明示的に宣言してもらう
- **GPS専用プラグインを `PluginReps` からも外す** — 設定画面・死活確認・
  GPSログの受け渡しが全部そこを通っているので、外すとプラグインが使えなくなる
- **`typed_index` を残したまま「GPSには使わない」と説明文で書く** —
  `never_built` は「壊れている」としか読めない。
  出さないことでしか「そもそも索引を持たない」は伝わらない
- **`gps_index` のために `get_plugin_list` を `wrapAuthRepos` へ変える** —
  統計を持つ GPSLog アダプタは `GkillRepositories` 側に居るが、
  そこへ届かせるために認可ラッパを変えると、一覧取得のたびにリポジトリ解決が走る
  （「読み込みが一生終わらない」の原因になった経路。ADR-0056 でも同じ理由で却下した）。
  プラグインリポジトリの実体は `PluginManager` が持つ**同じインスタンス**なので、
  アダプタ側から統計を預ければラッパを変えずに読める

## Consequences

- プラグイン一覧だけで「このプラグインは何を供給し、どこから読むのか」が決まる
- `emits_kyou:false` のプラグインの `data_type` / `rep_name` は
  「渡せる値」を並べたどの一覧にも出なくなった（一覧は `get_plugin_list` が引き受ける）
- `gps_index` は GPS が一度も要求されていなければ出ない。
  「壊れている」ではなく「まだ読んでいない」なので、`get_gps_log` の `count_only` を案内する
- MCP 側は出力を素通ししているので、新フィールドはコード変更なしで流れる。直したのは説明文だけ

## Evidence

- 実利用の報告: `reps:["GoogleLocation"]` が `unknown rep` になり、
  `gkill_get_all_rep_names` にも存在しない。一方 `gkill_get_gps_log` は 2,509件を返した
- `typed_index: {state:"never_built", record_count:0}` を見て
  「位置情報は使えないのか」と解釈された
- `src/plugins/gkill_plugin_google_locationhistory/main.go` の `FindKyous` は常に空を返す

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_plugin_list_test.go`
  - `TestHandleGetPluginList_EmitsKyouAndProvides`（役割が読めること、GPS専用に型別索引が付かないこと）
- `src/server/gkill/api/gkill_server_api/handle_get_rep_infos_mcp_test.go`
  - `TestHandleGetRepInfosMCPExcludesNonKyouPluginsFromPlugins`
- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`
  - `TestHandleGetKyousMCP_NonKyouPluginValuesGetNamedWarning`
- `src/mcp/__tests__/plugin-tools.test.mjs`
