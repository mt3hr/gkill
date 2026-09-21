# ADR-0216: 読めない rep は切り離して動き続ける。ただし黙って落とさない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-30 |
| Sources | `9ae607be` / `3f22c954` / `src/server/gkill/dao/gkill_dao_manager.go` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/gkill_dao_manager_broken_rep_test.go` |

## Context

`GetRepositories` は rep 定義を1本ずつ組み立てるが、**どこか1本でも失敗すると `return nil, err` で
丸ごと落ちる**。auth middleware（`auth_middleware.go`）は全リクエストでこれを呼ぶので、
rep 1本の障害が **そのユーザの全 API の 500** になる。

2026-08-30、USB接続ディスク上の directory rep の索引DB（`〈外部ストレージ〉/〈記録保管場所〉/.gkill/gkill_id.db`）が
SQLITE_CORRUPT になり、実際にそうなった。`/api/get_kyous` も `/api/get_gps_log` も一律 500。
**設定画面も開けないので、壊れた rep を無効化して自力で復旧する手段が残らない。**
`/api/reload_repositories` すら先頭で `GetRepositories` を呼ぶため使えない。

同型の事故は 2026-08-15 にも起きている（`$HOME/Git` 直下の Git Bash のクラッシュダンプ1ファイル）。
そのときは `git_commit_log` の case にだけ「gitリポジトリでないものはスキップする」を入れて塞いだが、
**種別ごとに同じ判断を書く形**なので、次に別種別が壊れたときには効かなかった。

## Decision

読み込みに失敗した rep は、**書き込み先でない限り**その1本だけ切り離して残りで動き続ける。

- 切り離しは**必ず利用者へ返す**（検索の応答に `MSG000090` の警告として載る。MCP は `warnings[]`）
- 運用者向けには `gkill_error.log` へも残す（切り離した1本ごとの行と、構築完了時のまとめ1行）。
  レベルを Warn ではなく Error にしているのは、rep が1本消えるのは検索結果が黙って痩せる異常で、
  `--log warn` 運用のときに他の警告へ埋もれさせないため
- `IsEnable` は書き換えない（設定DBへ書き戻さない）
- **失敗の理由では分岐しない**。どんな理由でも切り離し、必ず言う
- **書き込み先 rep が読めないときは従来どおり全体を失敗させる**（下記 Rejected alternatives 参照）
- 続行/中止の判定は `GetRepositories` の1箇所に集約する。
  組み立て本体は `loadRepIntoRepositories` へ切り出し、そちらには判定を書かない

## Rejected alternatives

- **従来どおり全体を失敗させる** — rep 1本の物理障害でそのユーザの全 API が `ERR000018` になり、
  設定画面すら開けないので**自力復旧の手段が残らない**。`/api/reload_repositories` 自身が
  `GetRepositories` を呼ぶので再読込 API も使えない。同型の自前呼び出しがハンドラ側に多数ある。

- **SQLite のエラーコード（SQLITE_CORRUPT=11）を見て、壊れているときだけ切り離す** —
  [ADR-0208](0208-git-repo-detect-by-os-stat.md) とまったく同じ罠。エラーの型・コードはドライバと OS の
  実装詳細で、次のバージョンで変われば**そのときだけ静かに全滅へ戻る**。
  方針を「どんな理由でも切り離すが、必ず言う」にして、判定を実装詳細から切り離した。

- **壊れた rep を黙ってスキップする（ログだけ）** — [ADR-0208](0208-git-repo-detect-by-os-stat.md) が
  「権限エラーや壊れたリポジトリまでスキップに化けて、**1件も出ないことに気付けなくなる**」として
  却下済み。あの却下が禁じているのは**黙る**ことなので、返す形での切り離しはその判断と矛盾しない。
  警告を出すことが、この決定が成立する条件そのものになっている。

- **壊れた rep を `IsEnable=false` にして設定へ書き戻す** — USB を挿し直せば直る種類の障害まで
  恒久的な設定変更になる。利用者は**自分が消していない rep が消えたこと**に気付けない。

- **書き込み先 rep も同じように切り離す** — `WriteXxxRep` が nil のまま
  `usecase/` と `handle_commit_tx.go` の書き込み経路へ入り、nil ポインタ参照で落ちる。
  `commit_tx` は DB トランザクションではなく決まった順の逐次書き込みなので、途中で落ちると
  **Kyou 本体だけ書けてタグと本文が落ちた**状態が残る（`usecase/mirekyou.go` が同じ事故を記録している）。
  500 で止まる方がまだましなので、先に書き込み経路の取得口（`WriteXxxRep` の直接参照をやめて
  nil を弾く形）を通すこと。順序を逆にすると 500 が panic に化けるだけになる。

- **`directory` 型だけを切り離しの対象にする** — 障害の原因（ディスクの瞬断 → SQLITE_CORRUPT）は
  種別と無関係で、次に `kmemo` が壊れたときに同じ修正をもう一度書くことになる。
  組み立て本体を切り出した後は判定が1関数なので、全種別にしても差分は増えない。

- **警告を `errors` に載せる** — `get_kyous` の `errors` はクライアントが**検索結果ごと捨てる**。
  `MSG000088`（プラグイン検索の失敗）が `messages` 側にいるのと同じ理由。

- **切り離しの有無をフラグで選べるようにする** — 「全滅する側」を選べる意味がない。

- **`get_rep_infos` の `rep_infos[]` に「読み込めなかった rep」を混ぜる** — 壊れた rep は leaf rep に
  ならないので、あの一覧を作る走査（rep 群を歩いて `UnWrap` → `GetRepName`）に**そもそも現れない**。
  載せるには架空の行を合成することになり、そうすると `rep_infos[]` の
  「`query.reps` へそのまま渡せる実 rep 名」という契約が壊れる。渡されたら
  `filterKyousByRepName` が非空の rep 名を「実在するが選ばれていない」として落とし、
  **エラーも警告も無く0件**になる。別枠（`unavailable_reps[]`）にする案もあるが、
  MCP の DTO・JS 側の射影許可リスト・tool description まで触ることになるので、
  まずは `gkill_error.log` に出す形で足りるかを見る。

## Consequences

**保存済みの検索条件に、切り離された rep 名が残る。** 列の `Reps` に名前が残っていると
`filterKyousByRepName` が「実在するが選ばれていない」として全件落とし、
**エラーも警告も無いまま0件**になる。`GetAllRepNames` からも消えるので UI の候補にすら出ない。
**この決定で警告を必須にしているのは、ここに気付けるようにするため。**

最新版アドレス表（`persistLatestDataRepositoryAddresses`）は AddOrUpdate のみで DELETE しないので、
欠けた rep を指す行が残る。`GetKyou` の「1 rep も選ばなかったら絞り込みを捨てて全 rep へ問い合わせ直す」
救済（[ADR-0210](0210-latest-data-address-uses-row-rep-name.md)）が受けるので誤動作にはならないが、
その warn が常態化する。

キャッシュ rep のフルリビルド（[ADR-0202](0202-rebuild-cache-only-on-db-change.md)）は、
欠けた rep のぶんをキャッシュから落とす。`--cache_in_memory=true`（既定）ならメモリDBなので再起動で戻る。
`git_commit_log` のキャッシュだけは常に永続ファイルなので、ディスク上のキャッシュから消える。
実データそのものは追記専用（[ADR-0201](0201-append-only-dao.md)）なので失われない。

rep によっては、生成に成功して `repositories` へ足したあとの段階（watch 登録など）で失敗しうる。
その場合 rep 自体は使えるまま警告だけが出る。黙るより良いので、そのまま警告する。

`UpdateCache` の途中で露見した破損は、この決定では救えない（`GetRepositories` の末尾で
全体を失敗させる経路が残っている）。SQLITE_CORRUPT は開いた後の読み取りで初めて出ることも多いので、
ここは今後の課題として残る。

## Evidence

実障害（2026-08-30）: USB接続ディスク上の `gkill_id.db` が SQLITE_CORRUPT
（`Tree 3 page 3 right child: invalid page number 66`）。`GetRepositories` が丸ごと失敗し、
`/api/get_kyous` `/api/get_gps_log` `/api/get_updated_datas_by_time` `/api/get_all_rep_names` が
すべて 500。復旧は実機へ ADB で接続し、6本の索引を1本ずつ `integrity_check` にかけて特定した。

同型の先行事例（2026-08-15）: `$HOME/Git` 直下のクラッシュダンプ1ファイルで全 API が 500。
`TestGetRepositoriesSkipsNonGitEntriesInGitCommitLogGlob` がその再発を止めている。

切り出しの安全性: `loadRepIntoRepositories` へ移した607行は、
git の判定1箇所を除いて元コードと1文字も違わないことを、元コードを機械変換した結果との
差分で確認した（コミット `c1706862`）。

## Related tests

- `src/server/gkill/dao/gkill_dao_manager_broken_rep_test.go`（切り離し・書き込み先の除外・`IsEnable` の不変・破損DBの再現・個別/集約 Error ログ・正常時に集約ログが出ないこと）
- `src/server/gkill/api/gkill_server_api/broken_rep_warning_test.go`（Web警告が `errors` でなく `messages` に載ること・パスが漏れないこと、MCPの通常/件数のみ/グループ化で `warnings` が残り `partial=false` と独立であること）
- `src/server/gkill/api/gkill_server_api/response_status_test.go`（認証時の保管場所取得失敗が `ERR000018`・HTTP 500・Error ログへ一貫して伝播すること）
- `src/server/gkill/dao/gkill_dao_manager_git_rep_test.go`（gitの非repスキップが警告に昇格していないこと）
