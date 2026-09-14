# ADR-0218: `/api/get_kyou` の版履歴はキャッシュ rep を回して集める（`UnWrap()` は rep 名照合のときだけ）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | `e568c8b8`（`UnWrap()` を入れた ReKyou 修正）/ `.claude/skills/gkill-go-backend/SKILL.md`「rep名の絞り込みは」/ `.claude/skills/gkill-client-foundation/SKILL.md`「Kyou の再読込」/ [ADR-0101](0101-filter-rep-after-cache.md) / [ADR-0605](0605-mcp-version-history-is-a-dedicated-tool.md) |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/reps/repositories.go`（`GetKyouHistoriesByRepName`）・`src/server/gkill/dao/reps/gkill_repositories.go`（`GetKyouHistoriesByRepName`）・`src/client/classes/kyou-reload.ts`（`fetch_refreshed_kyou`） |

## Context

KyouDialog で Mi にチェックを入れると、親の一覧の該当行が引き直され、右上のスピナーが数秒消えない。
チェックやタグ付けを続けるとだんだん重くなる、という報告から調べた。

引き直し（`kyou-reload.ts` の `fetch_refreshed_kyou`）は「SW キャッシュ削除 → `/api/get_kyou` → 型別 → 付随データ」の
約8往復で、時間の大半は `/api/get_kyou` だった。クライアントは `update_time` も `rep_name` も送らないので、
サーバは `Repositories.GetKyouHistoriesByRepName(ctx, id, nil)` に入る。この関数は冒頭で **`UnWrap()`** を呼び、
インメモリキャッシュ rep と `--cache_reps_local` のローカルコピー層を**両方剥がして生の leaf rep** へ戻していた。

実データでは leaf が約850本（IDF の `gkill_id.db` 531 / SQLite 系 約230 / git 84 リポジトリ / プラグイン 6）。
`/api/get_kyou` 1回ごとに、USB 接続 SSD 上の SQLite を約760本 `sql.Open` → 1 SQL → `Close`
（`fullConnect=false` の leaf は接続を持たない）、git 84本は ID がコミットハッシュに当たらず `Log(All:true)` で
約1万コミットを全走査、プラグイン6本へ IPC。これを `threads.Go` の 8 スロットで消化していた。
ADR-0101 が `selectMatchRepsFromQuery` について「`UnWrap()` の戻り値で検索しない」と決めた壊れ方そのものが、
`get_kyou` 経路に残っていた（ADR-0605 の却下案にも「11rep → 約940rep・20.7秒になった経路そのもの」と書かれていた）。

`UnWrap()` が入ったのは `e568c8b8`（ReKyou が画面に表示されない不具合の修正）。`rep_name` 指定時に
leaf の名前で照合するために必要だった変更で、`rep_name` 無しの経路には元から不要だった。

「だんだん重くなる」の正体は蓄積ではなく負荷依存の劣化だった。850本の fan-out が 8 スロットを長く占有すると、
同時に走る `get_mi` / `get_tags_by_id` の fan-out が待ち、飽和を見ると inline 逐次実行に倒れる（`threads.go`）。
プラグインへの `get_kyou` IPC は1プラグイン直列キューで、連続操作で待ち行列が伸び、上限の10秒を超えると
1 leaf の失敗として **`/api/get_kyou` 全体が ERR000101** になる。`gkill_error.log` には `/api/get_kyou` の
ERR000101 が 109 行、1〜2秒に36件のバーストで残っていた。クライアントは失敗時に1回リトライするので、更に倍になる。

クライアント側にも2つの取りこぼしがあった。列ごとの引き直しを直列 `await` で回していたため、
対象が2列に載っていると2列目が先発の決着後に始まり、`kyou-reload.ts` の合流（飛行中の表）に乗れずフルの往復を
もう1本払っていた。もう1つは、一覧の行は実行中 TimeIs を表示しない（`show_attached_timeis=false`）のに、
引き直しでは `load_all(query, true)` が付随データ4種を無条件に強制引き直しし、唯一「検索」である
`/api/get_kyous`（SW キャッシュ対象外）を行の引き直しのたびに撃っていた。

## Decision

`Repositories.GetKyouHistoriesByRepName` は **rep 名の指定が無いとき `UnWrap()` しない**。包まれた rep
（キャッシュ rep）の `GetKyouHistories` をそのまま回す。rep 名の指定があるときだけ剥がし、一致する leaf だけを回る。
`GkillRepositories.GetKyouHistoriesByRepName` を前段に置き、最新版アドレス表に載っている ID ではプラグイン rep を
問い合わせ先から外す。

クライアントは、列・ダイアログの引き直しを同じ tick で全部呼び出してから待ち、
実行中 TimeIs は発生元が読み込み済みのときだけ強制引き直しする。

## Rejected alternatives

- **アドレス表で leaf を1本に絞る（`GkillRepositories.GetKyou` と同じ方式）** — アドレス表が指すのは
  **最新版の** rep だけ。同じ ID の版が端末別の複数 rep に跨っているとき（PC で作って携帯で編集した記録）、
  最新版の rep だけを引くと古い版が履歴から欠ける。キャッシュ rep は種別ごとに全 leaf の全版を1表に持っているので、
  そちらを回せば1 SQL で全版が揃う
- **rep 名指定の経路も `UnWrap()` をやめ、行の `RepName` で後絞りする（ADR-0101 と同じ形）** — できるが、
  `e568c8b8` が固定した `re_kyou_granular_cache_test.go` の意味論（leaf 名で1件に絞る）を触ることになり、
  今回の症状には無関係。rep 名指定の経路は dispatch 前に leaf を1本に絞るので、元から走査は1本程度で済んでいる
- **アドレス表に無い ID でもプラグインを外す** — プラグイン Kyou の ID は表に載らないので、外すとプラグイン Kyou の
  `get_kyou` が黙って空になる。表に無いのはほかに「別プロセスが実 DB へ直接書いた、次回 `UpdateCache` 前の記録」だが、
  それはキャッシュ rep にも無いので、どちらへ聞いても結果は同じ（API の書き込み経路は
  `AddOrUpdateLatestDataRepositoryAddress` で表にも書くので、この経路で追加した直後の記録は載っている）
- **クライアントで `get_kyou` を呼ばず、`update_mi` の応答（`want_response_kyou`）で差し替える** — 付随データ
  （タグ/テキスト/通知）は応答に無いので結局引き直しが要る。サーバ側を直さないと他の `get_kyou`
  （ダイアログを開いたときの引き直し・履歴表示）が同じ850本を払い続ける
- **引き直しで実行中 TimeIs を常に飛ばす** — 詳細ペインとダイアログの KyouView は表示する。飛ばすと
  「ダイアログで保存したら実行中 TimeIs の表示が古いまま」になる。発生元が読んでいたかで分けるのが正しい

## Consequences

- `/api/get_kyou` の版履歴は**キャッシュ rep の内容**になる。別プロセス（携帯からの同期スクリプト等）が
  実 DB へ直接書いた版は、次回 `UpdateCache` までは履歴に出ない。これは型別の `/api/get_mi` 等が元から持っていた
  意味論と同じで、`get_kyou` だけが生 DB を読んで「一覧と履歴で見えている版が違う」状態のほうが不整合だった
- キャッシュ OFF（`cache_in_memory=false`）では `Repositories` の要素が元から leaf なので挙動は変わらない
- プラグイン rep の除外はアドレス表の有無で決まる。アドレス表 DAO を持たない組み立て（テストの素の構造体）では
  絞り込まず全 rep へ聞く
- クライアントの合流は「同じ tick で呼び出す」ことに依存する。列ループへ `await` を1本ずつ戻すと、
  例外もエラーも出ないまま列の数だけ往復が増える。`rykv-view-search-routing.test.ts` / `mi-view-search-routing.test.ts` が
  「2列に載っていても同じ tick で2本 dispatch される」ことで守る
- 一覧の行から始まった引き直しは `is_attached_timeis_loaded=false` のまま返る。詳細ペイン・ダイアログの KyouView は
  `load_attached_infos` の遅延読み込みで必要になったときに取るので、表示は変わらない

## Evidence

- 実環境（2026-09-14）: ローカルコピー済み rep 856 ファイル（IDF 531 / Kmemo 79 / URLog 59 / Tag 59 / Text 32 /
  TimeIs 24 / Nlog 18 / Mi 15 / Lantana 14 / KC 9 / Notification 12 / ReKyou 6 / MiReKyou 4）、git 84 リポジトリ
  約10,165 コミット、Kyou を出すプラグイン 6。`threads` のプールは `NumCPU()` = 8
- `gkill_error.log`（2026-08-30〜09-14）: `/api/get_kyou` の ERR000101 が 109 行。2026-09-12T19:53:51 に 36 件、
  09-13T22:44:23 に 5 件のバースト。`/api/get_mi` / `get_tags_by_id` にはバーストが無い
- 修正前の本番（2026-09-14、`performance.getEntriesByType('resource')` で計測）: mi 画面で行をダブルクリックして
  ダイアログを開いただけで、`open_rykv_dialog` の引き直しが出す `/api/get_kyou` が **16,755 ms**。
  同じ引き直しの他の往復は `/api/get_kyous`（実行中 TimeIs 検索）309 ms、`get_texts_by_id` 225 ms、
  `get_gkill_notifications_by_id` 226 ms、`get_tags_by_id` 60 ms。SW キャッシュ命中は 3〜10 ms。
  `get_kyou` が終わるまで後続の `get_mirekyou` 等が始まらないので、スピナーの長さ ≒ `get_kyou` の時間
- 同じ経路の過去の実測は ADR-0101（git rep だけでプロファイル1窓あたり 20.7 秒）
- 修正後の所要時間は本番へ配布してから同じ操作で再測定する（この ADR を書いた時点では未測定）

## Related tests

- `src/server/gkill/dao/reps/repositories_get_kyou_histories_cache_test.go`
- `src/server/gkill/dao/reps/re_kyou_granular_cache_test.go`（rep 名指定の経路が変わっていないこと）
- `src/client/__tests__/unit/classes/kyou-reload.test.ts`
- `src/client/__tests__/unit/composables/rykv-view-search-routing.test.ts`
- `src/client/__tests__/unit/composables/mi-view-search-routing.test.ts`
