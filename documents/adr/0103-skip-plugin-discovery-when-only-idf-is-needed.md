# ADR-0103: IDF のリポジトリしか要らない経路ではプラグインを探索しない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-29 |
| Sources | `.claude/skills/gkill-cli-ops/SKILL.md`「CLI サブコマンドと運用の約束」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/gkill_dao_manager.go` |

## Context

`GetRepositories` は、リポジトリを組み立てたあとにプラグインを探索する。探索は**1プラグインにつき1つサブプロセスを起動して**マニフェストを問い合わせ、提供する型に応じたアダプタを各 Reps へ登録する。

一方 `generate_thumb_cache` / `generate_video_cache` / `clear_cache` は `gkill_options.LoadIDFRepOnly` を立て、`IDFKyouReps` しか使わない。それでもプラグインの探索は毎回走っていた。

派生キャッシュの生成そのものを速くしたあと（[ADR-0100](0100-expand-rep-patterns-without-walking.md) / [ADR-0101](0101-derived-cache-scan-lists-directories.md)）、残った所要時間の大半がこの探索だった。プラグインを複数入れている利用者ほど不利になる。

## Decision

`gkill_options.LoadIDFRepOnly` が立っているときは、プラグインの探索とアダプタの登録を丸ごと飛ばす。

## Rejected alternatives

- **常に探索する（現状維持）** — 「一貫性のため」は、使わないものを毎回サブプロセスとして起動する理由にならない。しかも費用はプラグインの数に比例して増える。

- **探索を遅延させ、初回利用時に走らせる** — サーバ側には効くが、IDF しか使わない経路では「初回利用」が来ないので結果は同じで、そのくせエラーの出る場所が構築時から利用時へ動く。サーバ側の遅延化は別の判断として切り離すべきで、この決定と混ぜない。

- **探索を並列化する** — プロセス起動そのものは消えないうえ、この経路では1つも要らない。要らないものを速く起動しても仕方がない。

- **サブコマンドごとに「プラグインを読まない」旗を新設する** — `LoadIDFRepOnly` と同じ意図に2つ目の旗を立てることになり、片方だけ立て忘れる余地を作る。

## Consequences

- **この決定は「プラグインのアダプタが `IDFKyouReps` へ入らない」ことに依存している。** 登録先は Kmemo / KC / URLog / Nlog / Lantana / TimeIs / Mi / Tag / Text / Notification / GPSLog の各 Reps だけで、IDF は含まれない。将来プラグインが IDF 型のデータを提供して `IDFKyouReps` へ append するようになったら、この分岐は**そのプラグインぶんのサムネイルと互換動画がエラーも警告も出ないまま作られない**という壊れ方をする。append 先を増やすときは必ずここも見ること。
- 速くなるのは `LoadIDFRepOnly` を立てる CLI だけ。認証つきの HTTP リクエストは従来どおりプラグインを読むので、サーバ起動後の初回リクエストの待ちは変わらない。
- 回帰は `PluginManager` が作られていないことで見る。「探索へ入っていない」を直接観測できるのがここだけで、`PluginReps` が空なだけでは「プラグインが1つも入っていない環境」と区別がつかない。

## Evidence

複数のプラグインを入れている利用者で `GetRepositories` の内訳を測ると、リポジトリの構築とキャッシュ rep のラップを合わせても全体の1割に満たず、**残りはすべてプラグインの探索**だった。構築の内訳（ディレクトリ作成・SQLite の開閉・`IF NOT EXISTS` の DDL・スキーマ版の読み出し）を本体と同じ DSN・同じ文で別途測っても、リポジトリ数ぶん合計して数秒にしかならない。

つまり「リポジトリが多いから遅い」ではなく「プラグインを起動しているから遅い」だった。この切り分けをしないまま構築側を並列化しても、ほとんど何も速くならない。

## Related tests

- `src/server/gkill/dao/gkill_dao_manager_load_idf_rep_only_test.go`
