# ADR-0112: タグ語彙は「生存する対象を持つタグ」だけを返す — カスケード削除はしない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | MCP の2巡目の指摘（Q-07）。gkill-go-backend スキルの「タグ絞り込みの取得は2経路」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/gkill_repositories.go`（`GetAllTagNames` / `GetAllTagNamesIncludingDeletedTargets`） |

## Context

記録を削除してもタグは削除されない。`gkill_delete_kyou` は Kyou 側へ `is_deleted=true` の版を
追記するだけで、`TAG` テーブルには一切触らない。**これは意図した設計**で、そうでなければ
`gkill_restore_kyou` で復活させたときにタグが失われる。

しかし語彙の列挙（`get_all_tag_names`）はタグ自身の `is_deleted` しか見ていなかったので、
**対象がもう見えないタグが候補に残り続けた**。2巡目の指摘での実測:

| 検索 | 件数 |
|---|---|
| `tags:["点検タグ"]` | **0** |
| `tags:["点検タグ"]` + `include_deleted_data:true` | 1 |
| `get_all_tag_names` | **載っている** |

削除は日常操作なので、使うほど「選んでも0件しか返らない候補」が溜まっていく。
AI から見ると、存在しない切り口を提示され続けることになる。

同じ症状は ADR より前にも1度出ている（存在しない `target_id` へタグを付けられた件、
`5609f9c1` で塞いだ）。あれは「そもそも対象が無かった」経路で、今回は
**「実在する記録に正しく付けたあと、その記録を消した」**経路。後者のほうが日常的に起きる。

## Decision

**語彙を列挙する側で、対象が削除済みのタグを落とす。**

生存判定には既にある最新版アドレス表（`GkillRepositories.latestDataRepositoryAddresses`）を使う。
タグ行は対象 Kyou の ID を `TargetID` に持ち、Kyou 行はその ID をキーに `IsDeleted` を持っているので、
`reader.Get(tag.TargetID)` で対象の生死が直接引ける。

- **追加の I/O はゼロ。** `GetAllTagNames` は今も内部で `GetAllTags`（全 rep 並列・全タグ取得）を
  呼んでおり、`TargetID` は取得済み。増えるのは読み取りロック1回とタグ数ぶんのマップ参照だけ
- **アドレス表に載っていない対象は落とさない。** プラグインや git の記録は表に載らない。
  載っていないことを「削除済み」と読むと語彙が黙って痩せる
- **検証用の一覧は分ける。** `GetAllTagNamesIncludingDeletedTargets` を用意し、
  「そのタグ名は実在するか」の判定（`collectMCPUnknownValueWarnings`）はそちらを使う。
  フィルタ済みの一覧で検証すると、`include_deleted_data:true` で削除済みを開いた
  タグ検索に「未知のタグ」という**誤った警告**が出る

置き場所は `GkillRepositories`。`TagRepositories` は `[]TagRepository` でしかなく
Kyou rep への参照を持たないため、そちらへ入れると値コピー由来の同期問題に触れる。

## Rejected alternatives

- **記録と一緒にタグも削除する（カスケード削除）** — `gkill_restore_kyou` で復活させたときに
  タグが失われる。追記専用（ADR-0201）にも反する。「消したものを戻せる」を売りにしている以上、
  タグが残ること自体は筋が通っている
- **SQL に降ろす（TAG × Kyou の JOIN）** — キャッシュ OFF では TAG と各 Kyou 型が別ファイル DB で、
  キャッシュ ON でも `git_commit_log` だけ別ファイル。11型の UNION も要る。
  ADR-0102 の「横断ロジックを SQL へ降ろさない」に逆行する
- **`TagRepositories.GetAllTagNames` の側に入れる** — この型は Kyou rep への参照を持たない。
  持たせると `GkillRepositories` の値コピーと同期状態の問題に触れる
- **アドレス表に無い対象を落とす** — プラグイン・git の記録が表に載らないので、
  それらに付いたタグが全部消える。「静かに痩せる」ほうが「0件候補が残る」より悪い
- **何もしない（AI 側で0件を確かめてもらう）** — 候補ごとに1往復かかる。
  語彙は「まず候補を出す」ためのものなので、0件が混じると用をなさない

## Consequences

- 削除した記録にしか付いていないタグは候補から消える。**タグそのものは消えない**ので、
  対象を復活させれば候補にも戻る
- 未知タグ検証は対象の生死を問わない一覧を使うので、削除済みを開いた検索でも誤警告は出ない
- 出力を名前順に並べるようにした（以前は map の反復順で不定だった）
- 同じ穴が **Mi の板名**（`MiRepositories.GetBoardNames`）にも残っている。
  設計はそのまま使い回せるが、今回のスコープ外

## Evidence

- 2巡目の指摘での実測（本番 gkill へ stdio 接続、サンドボックス垢 `claude`）:
  `tags:["点検タグ"]` が 0件 / `include_deleted_data:true` を足すと 1件 /
  それでも `get_all_tag_names` には載る
- 最新版アドレス表の `IsDeleted` は素の `bool` 列で、`UpdateCache` の Phase 2 が
  各 rep の `IS_DELETED` 列を最新版の行から読み直して埋める（同じ表の
  `LatestDataRepositoryName` がプレースホルダで壊れていた件（`8f8234b2`）とは別のフィールド）
- 追加 I/O が無いことはコード上で確認: `GetAllTagNames` は元から `GetAllTags` を呼んでおり、
  `Tag.TargetID` は取得済み

## Related tests

- `src/server/gkill/dao/reps/gkill_repositories_test.go`
  - `TestGkillRepositoriesGetAllTagNames_DropsTagsWhoseTargetIsDeleted`
  - `TestGkillRepositoriesGetAllTagNames_KeepsTagsWhoseTargetIsNotInTheAddressTable`
  - `TestGkillRepositoriesGetAllTagNamesIncludingDeletedTargets_KeepsEverything`
