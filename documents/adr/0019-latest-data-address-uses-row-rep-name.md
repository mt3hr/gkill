# ADR-0019: 最新版アドレス表の rep 名は行ごとの実 rep 名にする（集約名を焼かない）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `5609f9c1` で顕在化した本番障害の調査 / [ADR-0018](0018-write-through-normalizes-rep-name.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/*_repository_cached_sqlite3_impl.go`（`GetLatestDataRepositoryAddress`）・`src/server/gkill/dao/reps/gkill_repositories.go`（`GetKyou`） |

## Context

最新版アドレス表（`LATEST_DATA_REPOSITORY_ADDRESS`）は「この ID の最新版はどの rep にあるか」を持つ索引で、
`GkillRepositories.GetKyou` はこれを使って問い合わせ先を全 rep から1 rep へ絞る。
突き合わせの相手は `Reps.UnWrap()` が返す **leaf rep の `GetRepName()`** ―― つまり
`Downloads_〈端末名〉_20260712` や `Kmemo_〈端末名〉_20260226` のような、ファイル由来の実名である。

ところが表へ書く側がこの前提を守っていなかった。
キャッシュ rep は集約型（`KmemoRepositories` など）を包んでおり、
`GetLatestDataRepositoryAddress` はその `GetRepName()` を
`? AS LATEST_DATA_REPOSITORY_NAME` で全行に焼いていた。集約型の `GetRepName` は
`"KmemoReps"` / `"IDFKyouReps"` のような**どの rep にも一致しない固定文字列**を返す。

結果、絞り込みの比較は永遠に外れ、全 rep が `continue` され、`GetKyou` は
**エラーも立てずに `(nil, nil)`** を返す。ReKyou 実装だけが `REP_NAME AS LATEST_DATA_REPOSITORY_NAME`
（行ごとの実名）を使っており、そこだけ正しく動いていた。

この欠陥は長く潜伏していた。`GkillRepositories.GetKyou` の本番呼び出し元は3箇所しかなく、
そのうち `usecase/kyou.go` の1つは `GET /api/get_kyou` の `update_time` 指定時だけ通る細い経路で、
壊れても「HTTP 200・`errors` なし・履歴0件」という気付けない形で返っていたからである。

2026-08-24、`5609f9c1`「Fix 存在しない対象IDにタグ/テキストを付けられる」がこの `GetKyou` を
`AddTag` / `AddText` の実在検査に使った。検査そのものは正しいが、土台が壊れていたため
**実在する記録へのタグ/テキスト追加が軒並み `ERR000092`（対象が見つからない）で失敗する**
という、利用者に見える障害になった。

「必ず失敗する」ではなく**条件つき**なのが厄介だった。API の書き込み経路
（`usecase/*.go` の `Add*` / `Update*`）はアドレス表へ**直接**、しかも
`WriteXxxRep.GetRepName()` = leaf 名で1行書く。だから
**同じプロセスで追加した直後の記録にはタグを付けられる**。壊れるのは
UpdateCache の Phase 2 がその ID を索引した後 ―― つまりスキャンで発見された記録すべてと、
再起動後の全記録である（既定 `--cache_in_memory=true` では表がインメモリなので、
起動時 UpdateCache が毎回すべてを集約名で埋め直す）。
この非対称のせいで、調査中に「別のアカウントでは成功する」という反例が出て一度診断が揺れた。

## Decision

キャッシュ rep の `GetLatestDataRepositoryAddress` は、
`LATEST_DATA_REPOSITORY_NAME` に**行の `REP_NAME` 列を射影する**。
`GetRepName()` をバインドしてはいけない。12型（idf_kyou / kc / kmemo / lantana / mi /
nlog / notification / tag / text / time_is / ur_log、および既に正しかった re_kyou）で揃える。

併せて `GkillRepositories.GetKyou` に2つの安全網を置く。

1. アドレス表の rep 名が**空文字**のときは絞り込まない（ライトスルーは
   `GetRepName` に失敗した行を空名で書く。ADR-0018 参照）
2. 絞り込みが**1 rep も選ばなかった**ときは、絞り込みを捨てて全 rep へ問い合わせ直し、
   `warn` を1行残す

## Rejected alternatives

- **`GetKyou` 側で集約名を許容する比較にする**（`"KmemoReps"` を来たら kmemo 系 leaf 全部に一致させる）
  ―― 型ごとの対応表を `GetKyou` に持つことになり、型を足すたびに2箇所を直す羽目になる。
  壊れているのは書き込み側なので書き込み側で直す（ADR-0018 と同じ判断）
- **安全網（全 rep へのフォールバック）だけを入れて本筋を直さない** ―― 名前が全行で食い違っている
  現状では毎回フォールバックが発火し、ID 1件の取得のために leaf rep 全走査が常態になる。
  実測した利用者設定では directory rep のグロブが135行あり leaf は数百 rep になる（ADR-0001 の実測は
  11 rep → 約940 rep・20.7秒）。安全網は「戻ってきたことに気付くための網」であって常用路ではない
- **MiReKyou も同じ形へ揃える** ―― `miReKyouRepositoryCachedSQLite3Impl` は
  `UnWrap()` が**自分自身**を返す。突き合わせの両辺が同じ `GetRepName()` なので現状で自己一貫しており、
  `REP_NAME` 列へ変えるとむしろ一致しなくなる。SQL も leaf 実装と共有ヘルパ
  （`queryMiReKyouLatestDataRepositoryAddress`）で、leaf 側の `?` バインドは正しい
- **アドレス表を使うのをやめて常に全 rep を引く** ―― 索引を捨てることになる。上と同じ理由で不可

## Consequences

- **`?` を1つ減らしたら `QueryContext` の引数も対で落とす。** 残すとバインドが1つずれ、
  例外もエラーも出ずに常に0件になる（2026-08-02 に `notification_repository_cached_sqlite3_impl.go` の
  `GetNotification` で同型の事故があり、「通知の編集・削除が必ず失敗する」として現れた）
- **移行作業は不要。** 既定 `--cache_in_memory=true` ではアドレス表はインメモリ DB 上にあり、
  再起動時の UpdateCache が leaf 名で埋め直す。`git_commit_log` の永続キャッシュが持つのは
  行データであってアドレス表ではなく、かつ委譲先の leaf が実名を書くので元から正しい
- **プラグイン Kyou は影響を受けない。** `plugin_repository_impl.go` の
  `GetLatestDataRepositoryAddress` は空スライスを返すのでアドレス表に載らず、
  `GetKyou` は絞り込み無しの全 rep 走査に落ちる。障害中もプラグイン Kyou へのタグ付けだけは通っていた
- **キャッシュ OFF では再現しない。** leaf 実装の `? AS LATEST_DATA_REPOSITORY_NAME` は
  自分の実名をバインドしており正しい。ADR-0018 と同じく、既定 OFF のテストだけでは見逃す

## Evidence

2026-08-24 の実測。

`dao/reps` の単体テスト（キャッシュ rep を挟んで `UpdateCache` → `GetKyou`）:

```
LatestDataRepositoryName = "KmemoReps", want "Kmemo_TestDevice_20260824"
GetKyou() がキャッシュrep構成で nil を返した
```

検証用サーバ（空ホーム・別ポート・`--cache_reps_local=true`・`--pre_load_users`）で、
「kmemo を1件足す → 再起動して起動時 UpdateCache に索引させる → `tx_id` 無しで
`/api/add_tag` と `/api/add_text`」を通した A/B:

| | 修正前 | 修正後 |
|---|---|---|
| 実在する対象へのタグ追加 | `ERR000092`（HTTP 404） | 成功 |
| 実在する対象へのテキスト追加 | `ERR000092`（HTTP 404） | 成功 |
| 存在しない対象へのタグ追加 | `ERR000092` | `ERR000092`（`5609f9c1` の目的は維持） |

## Related tests

- `src/server/gkill/dao/reps/gkill_repositories_get_kyou_test.go`（`TestGkillRepositoriesGetKyou_FindsKyouThroughCachedRep`）
- `src/server/gkill/dao/reps/latest_data_address_rep_name_scan_test.go`（ソース走査。型を足したときの再発を落とす）
- `src/server/gkill/dao/reps/re_kyou_granular_cache_test.go`（`TestGkillRepositories_UpdateCacheBuildsReKyouOnFirstPass`。元から正しかった側の固定）
