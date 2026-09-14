# ADR-0219: commit_tx は書き込み rep のファイルを ATTACH した1接続の SQLite トランザクションで確定する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-15 |
| Sources | 2026-09-13 外部レビュー #4「トランザクションがユーザの期待するトランザクションではない」 / `.claude/skills/gkill-go-backend/SKILL.md`「commit_tx は1つの SQLite トランザクション」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/commit_tx.go` / `src/server/gkill/dao/sqlite3impl/sqlite3impl_util.go`（`Preparer`） |

## Context

gkill は種別ごとに別の SQLite ファイルへ書く（kmemo.db / tag.db / text.db / ...）。1回の「保存」は
複数ファイルへの書き込みで、SQLite の1接続1ファイルのトランザクションでは束ねられない —— という前提で、
`/api/commit_tx` は temp rep（`tx_id` で積んだ一時リポジトリ）から13種別を**固定順に逐次追記**していた。
途中の種別で失敗すると `return` し、それより前の種別は残る。`handle_commit_tx.go` の doc コメントも
「名前に反して DB トランザクションではありません」と自己申告していた。

Wear OS / MCP の `/api/submit_kftl_text` は temp すら使わず実 rep へ直書きし、失敗したら
`created[]` に「そこまでに書けたもの」を載せて**利用者に後始末させる**設計だった。Web の連鎖削除は
タグ・テキスト・通知・ReKyou・MiReKyou を1件ずつ `update_*` で消し、失敗しても続行して「再実行で収束する」
という説明で運用していた。

2026-09-13 の外部レビューがこれを「利用者視点では『保存した』『削除した』は1操作。それが部分成功するなら
データ整合性モデルとしてかなり難しい。SQLite 間トランザクションが困難なら operation journal / saga /
commit marker のような上位概念で操作単位の原子性を復元する価値がある」と指摘した。

調査で副次的に見つかったもの: usecase の `Add*` / `Update*` は `tx_id` 指定時（temp に積むだけ）でも
最新版アドレス表を新しい版の時刻で進めていた。失敗 → `discard_tx` のあと表だけが未来を指し、
`find_filter.go` の「表より古い版は除外」で**既存の TimeIs が検索から消える**。保存に失敗したのに
副作用が残る実例で、例外もエラーも出ない。

## Decision

**`commit_tx` の中身を、行がある種別の書き込み rep のファイルを1接続に ATTACH した、1つの SQLite
トランザクションにする。** temp rep への積み方・`/api/commit_tx` / `/api/discard_tx` の API・クライアントの
`tx_id` の流れは既存のまま。失敗したら ROLLBACK で何も書かれない。Go 側 KFTL も temp に積んで同じ
`CommitTx` で確定する。usecase は `tx_id` 指定時に最新版アドレス表を進めない（commit が書く）。
commit が成功したら temp の行は消す（commit は tx を消費する）。

成立の根拠: 書き込み rep は常に素の leaf（ローカルキャッシュ層は非書き込み rep だけ）でファイルパスが取れる。
rep ファイルの DSN は `journal_mode(DELETE)` + `synchronous(FULL)`（ADR-0204 / ADR-0215）で、SQLite の
複数ファイル原子コミット（super-journal）の条件「main がファイルで WAL でない」を満たす。13 型の
`AddXxxInfo` はすべて単一テーブルへの INSERT 1本で、テーブル名（KMEMO / TAG / ... / IDF）が重複しないので、
ATTACH した接続で非修飾名がそのまま目当てのファイルに落ち、既存の INSERT SQL を `*sql.Tx` に対して
そのまま流用できる（`insertXxxRow` に切り出し、`AddXxxInfo` と commit の両方がそれを呼ぶ）。

## Rejected alternatives

- **saga（論理補償）で戻す** — 最初の計画。書いた版と直前の版を控え、失敗したら逆順に「直前の版の再追記」
  「無ければ `is_deleted=true` 版の追記」で打ち消す。追記専用 DAO（ADR-0201）とは整合するが、
  (1) `UPDATE_TIME` が秒精度なので補償版は元の版の +1 秒にしないと同秒で黙って負ける、(2) 補償版が履歴に残り
  `include_deleted_data` で見える、(3) 補償自体が失敗する二重障害の応答形が要る、(4) 既存の temp_db と
  commit の仕組みと二重になる。ユーザーが「新設せず既存の temp_db と commit ロジックで解けないか」と差し戻し、
  ATTACH で解けることが分かった時点で不要になった。
- **物理 DELETE で戻す** — ADR-0201 が同期衝突（削除した端末と未削除端末が同期すると行が復活する）で
  却下済み。加えて 13 型 × leaf / cached の新 DAO メソッドとアドレス表の巻き戻しが要る。
- **永続 operation journal（プロセス断の復旧）** — 利用者別ローカル SQLite に「操作 ID・状態・適用した各版と
  直前の版」を同期書きし、次回起動時に未完了操作を補償する案。ATTACH のトランザクションなら SQLite が
  super-journal でプロセス断も含めて all-or-nothing にするので不要。temp はインメモリなので未 commit 分は消えるが、
  実 rep に半端は残らない。
- **サーバに `/api/delete_kyou`（連鎖削除 + saga）を新設する** — 削除だけ別の原子性機構になる。既存の
  `update_*` が `tx_id` を受けるので、クライアントが束ねて `commit_tx` すれば同じ仕組みで足りる（ADR-0410）。
- **何もせず受容する（ADR-0217 と同じ型）** — アップロードは「残ったファイルは次の IDF 走査で取り込まれる」
  という回収経路があったが、保存の部分確定には無い（Kyou 本体だけ残り、タグと本文が無い）。
- **ATTACH 数の上限（modernc は `SQLITE_MAX_ATTACHED = 10`）を定数と事前判定で守る** — 実装したがユーザーが
  「要らない、他と同じように」と差し戻した。超えれば ATTACH 自体が失敗して何も書かずに返るので静かには壊れない。
  既存の操作は KFTL 最大でも 10 種別、連鎖削除 6 種別、追加画面 3 種別。

## Consequences

- 1回の commit に載る種別数は SQLite のコンパイル時上限（main + 10 ファイル）に従う。超える操作を将来足すなら
  tx を分ける。超えたときは `CommitTxRolledBackError`（ERR000419）で何も書かれない。
- commit は temp を消費する。同じ `tx_id` をもう一度 commit しても何も書かれない（以前は丸ごと二重登録だった）。
  失敗時は temp を残す（再 commit / discard できる）。クライアントは失敗したら discard する。
- Go 側 KFTL の `created[]` は「確定したもの」になり、失敗時は空。「そこまでに書けたもの」の意味は消えた。
  Wear / MCP の再送案内は「同じ内容をそのまま送り直せばよい」で足りる（冪等キーは従来どおり）。
- 打刻の終了が「同じテキスト内で開始した打刻」を見つけられないのは以前からで（実 rep を引く）、以前は
  開始だけ残っていたが今は何も残らない。
- ATTACH した接続は rep 自身の接続プールとは別で、rep 単位の mutex を取らない。競合は SQLite のファイルロックと
  `busy_timeout(6000)` に任せる。BEGIN IMMEDIATE で先頭ファイルのロックを先に取る。
- 各 leaf の INSERT は `insertXxxRow` にだけ置く。`AddXxxInfo` へ複製すると、tx 経由の追記だけが静かにずれる。
- キャッシュへのライトスルーと最新版アドレス表の更新はトランザクションの**外**（commit 後）で、失敗はログのみ。
  次の UpdateCache で直る（以前と同じ）。

## Evidence

実測なし — 構造からの判断（SQLite の複数ファイル原子コミットの仕様と、書き込み rep が常に leaf であること）。
部分確定の事故は `handle_submit_kftl_text_test.go` の旧テスト「途中失敗でも書けたぶんは created に載る」が
仕様として固定していたもので、今回反転させた。

## Related tests

- `src/server/gkill/dao/reps/commit_tx_test.go`（ROLLBACK で何も残らない・temp の消費・書き込み rep 未設定は書く前に返す・テーブル名の重複なし）
- `src/server/gkill/api/gkill_server_api/handle_commit_tx_atomic_test.go`（ERR000419・`committed[]`・再 commit で版が増えない・tx 更新の discard 後に既存の TimeIs が残る）
- `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text_test.go`（「途中失敗では何も残らず created は空」）
- `src/server/gkill/usecase/source_conventions_scan_test.go`（`TestCommitTxSetsRealRepNameBeforeWriteThrough` / `TestCommitTxRestoresIDFTargetRepNameBeforeRealWrite`）
