# ADR-0021: プラグインは provides で型別/付随データを提供でき、アダプタの読み取りは索引から即答する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-11 |
| Sources | `3c3bfade` / `5afbd408` / `documents/reverse/plugin-system.md` 14章 / `.claude/skills/gkill-plugin/SKILL.md`「プラグインの型別/付随データ」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/plugin_typed_index.go` |

## Context

プラグインの記録（`plugin_kyou`）は当初、型別リポジトリにも付随データにも載らなかった。そのため2つの実害があった。

- Fitbit のような数値データを返しても **Dnote の推移グラフで集計できない**
- タグを付けてもタグ一覧（`get_all_tag_names`）に載らないので、**rykv の既定の絞り込み「タグ無し」から漏れて0件になる**

一方で、型別/付随データの読み手は**1件ずつ聞きに来る**（一覧の行ごとにタグを引く）。プラグイン呼び出しは容量1のスロットで直列化されるので（→ ADR-0020）、素直に委譲すると**一覧の行数ぶんの直列 stdio 呼び出し**になり、デッドラインでプロセスが殺され続ける。

## Decision

`manifest.json` の `provides` に種別を書くと、そのプラグインの記録が native と同じ型別リポジトリに載る。

そのうえで、**アダプタの読み取りは決してプラグインへ往復しない**。`PluginTypedIndex` が `find_kyous` 1回ぶんの**不変スナップショット**を持ち、そこから即答する。

## Rejected alternatives

- **アダプタが素直にプラグインへ委譲する** — 一覧の行数ぶんの直列 stdio 呼び出しになる。容量1のスロットで直列化されるので並列化しても解決せず、プロセスが殺され続ける。

- **`provides` を書かせず、返ってきた `data_type` から自動で推測する** — 作者が意図せず型別リポジトリに載る。とくに `emits_kyou` の推測は危険で、推測を誤ると**作者の記録が黙って消える**（→ ADR-0023）。宣言させる。

- **アダプタの `GetLatestDataRepositoryAddress` が実データを返す（型別の場合）** — 返すと `UpdateTime` の揺れで**レコードごと消える**。型別は空を返す。

- **アダプタの `GetLatestDataRepositoryAddress` が空を返す（付随データの場合）** — 返さないと `--cache_in_memory=false` で**タグが全部落ちる**。付随は実データを返す。型別と付随で逆になるので、片方を見て揃えてはいけない。

- **アダプタの `Close` がプラグインプロセスを閉じる** — アダプタは複数あり、1つ閉じると他が死ぬ。`Close` は no-op にし、閉じるのは本体と `PluginManager.CloseAll` だけ。

- **アダプタの登録を `KCReps`→`Reps` コピーループより前に置く** — 前だと**二重検索**になる。後に置く。

## Consequences

登録順の制約（`KCReps`→`Reps` コピーループより**後**）が、別の不具合を生んだ。`GkillDAOManager` はキャッシュrepで `XxxReps` を1個に差し替えた**あと**にアダプタを append するので、**`len(XxxReps) == 1` によるキャッシュrep判定が破れる**（→ ADR-0012）。この2つの決定は順序で結びついていて、片方だけ見て順序を入れ替えると他方が壊れる。

`PluginTypedIndex` は「`find_kyous` 1回ぶんの不変スナップショット」なので、**検索をまたいで使い回してはいけない**。

## Evidence

実測なし — 構造からの判断（呼び出しが容量1のスロットで直列化されるので、1件ずつの委譲は行数に比例した直列 stdio 呼び出しになる）。

症状は実機で観測している（Ryuu の PluginKyou 一覧でプロセスが繰り返し殺される）。

## Related tests

- `src/server/gkill/dao/reps/plugin_typed_adapters_test.go`
- `src/server/gkill/dao/reps/plugin_kyou_rep_name_test.go`
- `src/server/gkill/dao/reps/gps_log_repository_plugin_impl_test.go`
- `src/server/gkill/usecase/write_through_cache_test.go`
