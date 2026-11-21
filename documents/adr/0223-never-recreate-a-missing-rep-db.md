# ADR-0223: 消えた rep のDBファイルを空のDBとして作り直さない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-21 |
| Sources | `src/server/gkill/dao/sqlite3impl/sqlite3impl_util.go` / `src/server/gkill/dao/gkill_dao_manager.go` / `src/server/gkill/dao/reps/local_rep_cache_copy.go` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/missing_rep_db_test.go` / `src/server/gkill/dao/sqlite3impl/sqlite_connection_missing_test.go` |

## Context

rep のDBはすべて `GetSQLiteDBConnection` で開いていた。DSN に `mode=` が無く、SQLite の既定（`rwc`）のまま
**ファイルが無ければ空のDBを作る**。読むだけの rep（`USE_TO_WRITE=0`）は呼ばれるたびにここで開き直し、
コンストラクタは読むだけの rep にも `CREATE TABLE IF NOT EXISTS` を流し、`loadRepIntoRepositories` は
読むだけの rep にも親ディレクトリを `MkdirAll` していた。

本番で、利用者が容量を減らすために端末同期の置き場（`$HOME/Kyou`）を空にしたあと、動いていた gkill_server が
キャッシュ更新で消えた rep のファイルを開き直し、**表だけの空のDBとして作り直した**。端末同期の置き場から
全保存の置き場へのコピー（更新時刻が新しい方で上書きする）が、その空のDBを「新しいファイル」と判定し、
**全保存側にあった中身のある版を空の版で上書きした**。もう使っていない端末の rep（同期ハブに無く、どこからも
降りてこない）が数十本、全保存側で空になった。症状は「終わったはずの TimeIs が実行中に出る」だった
（ある端末で始めた TimeIs を別の端末で終えた記録が、空になった側の rep にあった）。

利用者がファイルを消すこと自体は正当な操作で、壊れたのは **gkill が消えたファイルを黙って作ったこと**である。
空のDBはエラーも警告も出さず、検索は「その rep には何も無い」と答えるだけなので、気付くのは同期で広がった後になる。

## Decision

- `GetSQLiteDBConnection` は**既にあるファイルだけを開く**。先に stat して、無ければ `os.ErrNotExist` を包んだ
  エラーを返し、DSN にも `mode=rw` を付ける
- 作ってよいのは `CreateSQLiteDBConnection`（`mode=rwc`）と `OpenRepositoryDBConnection(allowCreate=true)` だけ。
  rep のコンストラクタは `fullConnect`（= 書き込み rep）を `allowCreate` として渡す。**書き込み rep を初めて置くときだけ作る**。
  書き込み rep のファイルを最初に置くのは新しい利用者の初期化（`initializeNewUserReps` が中身の無いファイルを置く）で、
  表はそのあとの読み込みでコンストラクタが作る。起動時の rep の展開（`expandRepFilePattern`）は、ワイルドカードの無いパスで
  ファイルが無ければその rep を飛ばす（この ADR の前から同じ。go-zglob の時代も同じ）ので、**消えた書き込み rep のファイルも
  起動時には作り直されない**。作り直されるのは、書き込みのディレクトリ rep の `.gkill/gkill_id.db`（ディレクトリが有るとき）だけ
- `commit_tx` の `OpenSQLiteDBConnectionForTx` と `ATTACH` も作らない（`mode=rw`）。書き込み rep が起動後に消えていたら、
  空のDBへ書いて書けたことにせず、エラーで止める
- `loadRepIntoRepositories` は、書き込み rep のときだけ親ディレクトリを作る。読むだけのディレクトリ rep は、
  対象ディレクトリも `.gkill/gkill_id.db` も作らず、ID表が無ければエラーにする（[ADR-0216](0216-detach-a-broken-rep-but-never-silently.md) の
  切り離しと警告に乗る）
- ローカルキャッシュ版（`*_local_cached.go`）の `UpdateCache` は、元のファイルが有ることを確かめてから
  ローカルキャッシュを閉じて消す。元が無ければエラーを返し、使えていたキャッシュは残す

## Rejected alternatives

- **読むだけの rep を `mode=ro` で開く** — 読むだけの rep もコンストラクタで `CREATE INDEX IF NOT EXISTS`・
  スキーマ版の記録（`GKILL_META_INFO`）・`EnsureUnixepochIndex` を書くので、`ro` では古いDBを開けなくなる。
  守りたいのは「書かない」ではなく「**無いファイルを作らない**」なので、`rw`（作らないが書ける）にした。

- **コンストラクタで1回だけ有無を見て、以後の開き直しはそのまま** — 今回の事故は読み込みの**後**に消えた
  ファイルが、検索とキャッシュ更新の開き直しで作られて起きた。読むだけの rep は接続を持ち続けない
  （`fullConnect=false` で開き直す）ので、開き直すたびの経路を塞がないと同じことが起きる。共通の入口で塞いだ。

- **作り直した空のDBを、同期の側（コピーのスクリプト）で弾く** — 空のDBを作る主体は gkill で、同期の作り方は
  利用者ごとに違う（hbg・robocopy・rsync・手作業）。同期側の手当ては別にあってよいが、gkill が作らなければ
  どの同期でも起きない。

- **書き込み rep も初回作成を含めて作らない** — 新しい利用者の初期化が置いた中身の無いファイルに、表を作れなくなる。
  書き込み rep は gkill が持ち主なので、初めて置くときに作るのは正しい。起動後に消えた書き込み rep は
  接続を持ち続けている（Windows では消せない）か、`commit_tx` の `mode=rw` で止まる。

- **起動時の展開で、無い書き込み rep のパスも通してコンストラクタに作らせる** — 今はワイルドカードの無いパスで
  ファイルが無いと、その rep は飛ばされる（`rep_file_glob_test.go` が go-zglob と同じ結果であることを固定している）。
  通すようにすると、利用者が消した書き込み rep が起動のたびに空のDBで復活し、この ADR が止めた形がそのまま戻る。
  新しく置く経路は新しい利用者の初期化に任せる。

- **ディレクトリ rep で `IS_EXECUTE_IDF_WHEN_RELOAD` のときだけID表を作る** — ID表が消えたディレクトリで作り直すと、
  ファイルに新しい ID が振り直され、元の ID に付いていたタグやテキストが宙に浮く。そのID表が同期で全保存側へ
  広がるのは DB と同じ形。読むだけのディレクトリ rep のID表は端末同期で運ばれてくるものとして扱う。
  ID表を新しく作るのは `gkill_server idf <ディレクトリ>`（書き込みとして開く）だけ。

## Consequences

- 読むだけの rep のファイルが消えると、次の再読み込みで切り離され、検索の応答に警告が出る
  （[ADR-0216](0216-detach-a-broken-rep-but-never-silently.md)）。読み込み済みのまま消えたときは、
  検索とキャッシュ更新がその rep の「ファイルがありません」エラーを返す。黙って空になるよりも気付ける。
- 端末同期の置き場を空にしたいときは、gkill を止めてから消し、起動し直して rep の一覧を作り直させる
  （glob は起動時に展開されるので、消えたファイルはそもそも読まれない）。
- `GetSQLiteDBConnection` を新しい用途（gkill 自身が持つDB）に使うときは、作る必要があるなら
  `CreateSQLiteDBConnection` を使うこと。設定DB・アカウントDB・キャッシュDBは元から別の開き方なので変わらない。

## Evidence

実障害: 端末同期の置き場を空にしたあとのキャッシュ更新で、消えた rep のファイルが表だけの空のDBとして
作られ（作成時刻が数秒の幅に集まり、大きさは種類ごとの表だけの値）、続く同期のコピーで全保存側の
中身のある版がその空の版に置き換わった。中身のある版は、同期ハブからだけ受け取る別の控えに残っていた。
gkill のログには、その直前から消えた rep のファイルの「ファイルが見つからない」が出ていた。

## Related tests

- `src/server/gkill/dao/reps/missing_rep_db_test.go`（12種のコンストラクタが読むだけでは作らず書き込みでは作れること・
  読むだけのディレクトリ rep がID表を作らないこと・読み込み後に消えたファイルの検索がエラーになり作り直さないこと・
  ローカルキャッシュ版が元の消失でキャッシュを壊さないこと）
- `src/server/gkill/dao/sqlite3impl/sqlite_connection_missing_test.go`（`GetSQLiteDBConnection` / `OpenSQLiteDBConnectionForTx` が
  作らないこと・`OpenRepositoryDBConnection` が `allowCreate` のときだけ作ること）
- `src/server/gkill/dao/sqlite3impl/sqlite_connection_test.go`（接続のプラグマ。作る側で置いてから開く形に変更）
