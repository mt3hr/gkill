# ADR-0023: 記録を返さないプラグインは emits_kyou: false で明示する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-11 |
| Sources | `3c3bfade` / `documents/reverse/plugin-system.md`「emits_kyou — 記録を返さないプラグイン」 / `CLAUDE.md` の `src/plugins` 構造コメント |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_plugin/plugin_manifest.go` |

## Context

Google ロケーション履歴プラグインは **GPSLog だけを提供し、Kyou を1件も返さない**。ところがプラグインは一律に `Repositories.Reps` へ登録されるので、

- 「記録保管場所」の一覧に**選んでも0件になる項目**が並ぶ
- 検索のたびに**空振りの stdio 往復**が出る

## Decision

`manifest.json` の `emits_kyou` を `false` にすると `Repositories.Reps` に登録しない（省略時 `true`）。

`PluginReps` と `GPSLogReps` には従来どおり入るので、**GPSログは rep の選択状態と無関係に常に効く**。

## Rejected alternatives

- **`provides` の内容から自動で推測する（`gpslog` だけなら Kyou を返さないとみなす）** — **推測を誤ると作者の記録が黙って消える。** 「GPSログも Kyou も返す」プラグインは正当に書けるので、`provides` から `emits_kyou` は導けない。判定できない事実を推測しない。

- **`FindKyous` が0件を返したら自動で外す** — 初回呼び出しは取り込み前なので**必ず空で返る**。1回の結果で判断すると、正常なプラグインが起動直後に一覧から消える。

- **rep 一覧の表示側で0件のものを隠す** — 空振りの往復は残る。しかも「本当に0件なのか、まだ取り込んでいないのか」を表示側は区別できない。

- **既定を `false` にして、Kyou を返すプラグインに宣言させる** — 既存プラグインが全部壊れる。既定は従来どおり `true`。

## Consequences

マニフェストに宣言が1つ増える。プラグイン作者は「Kyou を返すか」を明示することになるが、これは作者だけが決定できる事実なので、宣言させるのが正しい。

`emits_kyou: false` のプラグインでも `PluginReps` / `GPSLogReps` には入る。**「Reps に居ない ＝ 何も提供しない」ではない**ので、rep 一覧を根拠にプラグインの有無を判断しないこと。

## Evidence

実測なし — 決定可能性からの判断（`provides` から `emits_kyou` は導けず、初回呼び出しは必ず空で返る）。

## Related tests

- `src/server/gkill/api/gkill_plugin/plugin_manifest_test.go`
- `src/plugins/gkill_plugin_google_locationhistory/manifest_test.go`
- `src/server/gkill/dao/reps/gps_log_repository_plugin_impl_test.go`
