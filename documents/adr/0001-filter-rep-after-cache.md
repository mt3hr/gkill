# ADR-0001: rep名の絞り込みは「検索するrep」ではなく「検索結果」で行う

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-19 |
| Sources | `06fcdec4` / `44b2be68` / `df9ceeb8` / `.claude/skills/gkill-go-backend/SKILL.md`「rep名の絞り込みは『検索するrep』ではなく『検索結果』でやる」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_filter.go` |

## Context

本番プロファイル（2026-08-19）で、rykv の検索に残っていた最大の項目が **git rep の 20.7秒＝実質CPUの60%** だった。go-git の `Repository.Log()` が全コミットをディスクから読み直していた。

「キャッシュ構築中だけ」「repの配線ミス」の2説は一時ログと info ログでどちらも否定された（`Reps` に入っているのはキャッシュrep、`isCacheBuilding` は偽）。

真因は `selectMatchRepsFromQuery` の Step4 だった。クエリが rep名を指定していると `matchRep.UnWrap()` でラッパーを剥がし、**生のディスクrepを `MatchReps` に登録して検索していた**。すぐ上の Step3 には「ここで `UnWrap()` してはいけない」と理由まで書いてあるのに、Step4 には適用されていなかった。

これが常時経路だったことが効いた。**GUI は常に非nullの `reps` を送る**（`FindKyouQuery` のコンストラクタ既定が `[]` で、`apply_rep_summary_sets_to_detaul` が必ず配列を入れる）ので、rykv/mi の検索は毎回ここを通る。rykv は `rep_types` を送らないため、11個のキャッシュrepが**約940個の生rep**に展開されていた（実データの一例では 312rep 中 263個が端末別の重複登録）。`--cache_reps_local` のローカルコピー層も同時に剥がれ、クエリが外付けUSB上の元DBへ戻っていた。影響は git だけでなく全型に及び、キャッシュ版でない `idfKyouRepositorySQLite3Impl.FindKyous` が 3.96秒でプロファイルに出ていた。

## Decision

rep名の絞り込みを「**どのrepを検索するか**」から「**どの結果を残すか**」へ移す。`selectMatchRepsFromQuery` はラッパーのまま `MatchReps` へ入れ、実際の絞り込みは `findKyous` の `filterKyousByRepName` が `Kyou.RepName` に対して行う。

`UnWrap()` を使ってよいのは「そのラッパに選ばれた実repが1つでもあるか」の**枝刈り判定だけ**。

## Rejected alternatives

- **`UnWrap()` した生repを `MatchReps` に入れたまま、検索する rep を絞る（改修前の実装）** — キャッシュを丸ごとバイパスする。実データで11キャッシュrep→約940生rep、git だけでプロファイル1窓あたり20.7秒。`--cache_reps_local` のローカルコピー層も剥がれ、外付けUSB上の元DBへクエリが戻る。

- **枝刈りの `UnWrap()` も省いて、そもそも候補repを絞らない** — 省くと「キャッシュOFF」「1種別だけチェック」「一致0件」の3ケースがむしろ悪化する。枝刈りは残す。

- **`dao/reps` 側に絞り込みを置く** — そこを通るのは利用者のクエリだけではない。ReKyou / MiReKyou は参照先を解決するために `target_resolution_memo.go` で**利用者のクエリをそのまま** `FindKyousSequential` へ渡す。ここで rep名で絞ると、チェックしていないrepに参照先を持つリポストが「参照先が見つからない」扱いになり、**語句検索に黙って当たらなくなる**（エラーも0件表示も出ず、ヒット数が減るだけ）。ソース走査 `TestNoRepNameFilterInDaoReps` が見張っている。

- **SQL の WHERE へ押し込む** — 別起案。→ ADR-0002

## Consequences

意味論は変わらない。`OnlyLatestData` は本番の全経路で true 固定で、ID X について M＝全rep横断の最大 `UpdateTime`、I＝指定rep とすると、旧は「各leafの自表内MAXの行」、新は「横断MAX(=M)の行のうち `RepName`∈I」。`replaceLatestKyouInfos` が全rep由来のアドレス表と突き合わせて「最大がMに届かなければレコードごと落とし、届けば `UpdateTime`==M の行だけ残す」ので、どちらも同じ集合に収束する。キャッシュOFFでは `UnWrap()` が自分自身を返すので完全一致。

結果側で絞ることの代償として、**5つの落とし穴が生まれた**。どれも例外もエラーも出さずに壊れる。規則そのものは `.claude/skills/gkill-go-backend/SKILL.md` にあるが、なぜ必要かはここに残す。

1. 本文ヒット由来の2本目の検索（`matchTextFindByIDQuery`）にも同じ絞り込みが要る。片方だけだと本文で当たった記録が rep 絞り込みをすり抜ける
2. 全部落ちたIDは**キーごと消す**。空スライスを残すと `kyous[0]` を見る `filterLocationKyous` / `filterMiForMi` / `overrideKyous` が **panic** する
3. `Reps == nil` は「未指定」。`len()` で判定すると**全件消える**
4. **`RepName` が空の行は残す。** キャッシュrepへの write-through は呼び出し側の値をそのまま INSERT するので、追加直後の行は `REP_NAME` が空（実rep名が入るのは次の `UpdateCache`）。落とすと**いま追加した記録が最大1分間一覧から消える**。残しすぎても「チェックを外したrepの、たった今書いた記録が1分だけ残る」で済むので、非対称に安全側へ倒す
5. 書き込み側に**実在しないrep名を入れさせない**。非空のrep名は「実在するが選ばれていないrep」として落とされるので、合成した名前を渡すと記録が黙って消える。実例: `commit_tx`（KFTLの送信経路）は一時リポジトリから読み直した記録をそのまま write-through していたが、`GetXxxByTXID` は `? AS REP_NAME` に temp rep の名前を差し込んで返すため、**KFTLで書いた記録だけが一覧から丸ごと消えていた**

落とし穴4の裏返しとして、フィルタ側に `*Temp` の例外を足してはいけない（temp rep 名は流儀がばらばらで、一覧を2箇所で維持することになる）。直すのは常に**書き込み側**。

## Evidence

- 改修前: git rep 20.7秒／プロファイル1窓（実質CPUの60%）、11キャッシュrep → 約940生rep（312rep 中 263個が端末別の重複登録）、キャッシュ版でない `idfKyouRepositorySQLite3Impl.FindKyous` が 3.96秒
- 実データ（2026-08-19 実測、本番プロファイル）

## Related tests

キャッシュONでの rep 絞り込みは改修前まで**未テスト**だった（既存の `TestHandleGetKyous_RepFilter` はキャッシュOFFかつ「存在しないrep名で0件」しか見ていなかった）。以下はすべてキャッシュON/OFFの両方で回る。

- `src/server/gkill/api/select_match_reps_cache_test.go`
- `src/server/gkill/api/find_kyou_rep_name_filter_test.go`
- `src/server/gkill/api/gkill_server_api/get_kyous_rep_filter_test.go`
- `src/server/gkill/api/gkill_server_api/get_kyous_tx_rep_filter_test.go`
- `src/server/gkill/usecase/source_conventions_scan_test.go`（`TestNoRepNameFilterInDaoReps`）
