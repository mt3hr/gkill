# ADR-0064: 付随データの「その瞬間に走っていたか」は Kyou 検索と同じ規則で判定する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-25 |
| Sources | 2026-08-25 の実利用レビュー（`is_include_timeis` が削除済みの打刻を添付する）と、本番アカウントでの実測 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go`（`livePlaingTimeIsCandidates` / `timeIsCoversMoment`）/ `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp.go`（付随 TimeIs の取得と添付） |

## Context

`gkill_get_kyous` に `is_include_timeis:true` を付けると、1件の記録に**削除済みの打刻が混ざって**返っていた。

本番実測（2026-08-25）:

| | 件数 |
|---|---|
| `is_include_timeis:true` が付けた打刻 | **16件** |
| 同じ瞬間を `plaing_time` で引いた打刻 | **2件** |

差の14件はすべて削除済み。最古のものは**1年前（2025-08-14）に開始し、終了記録ごと消された**打刻で、
それが「今日 17:30 に書いたメモ」へ *実行中* として付いていた。

原因は、同じ「その瞬間に走っていた打刻」の判定が**3つ別々に実装されていた**こと。

| 経路 | 実装 | 削除済み |
|---|---|---|
| Web の列 | クライアントが `plaing_time` 検索を投げる（`generate-plaing-timeis-query.ts`） | 落ちる |
| 共有ページ | Kyou 側は `FindFilter.FindKyous`、打刻の実体は `TimeIsReps.FindTimeIs` 直叩き | **Kyou 側だけ落ちる** |
| **MCP** | `TimeIsReps.FindTimeIs` を直叩きして Go 側で総当たり | **落ちない** |

gkill で削除済みを落としているのは `find_filter.go` の Kyou 集約だけで、SQL にもリポジトリにも無い
（`GenerateFindSQLCommon` に `IS_DELETED` は一度も現れない）。MCP の独自走査はそこを通らない。
`reps.TimeIs.IsDeleted` は値としてその場にあり、ループが読んでいないだけだった。

終了していない打刻は「開始時刻より後をすべて覆う」という意味論なので、
**削除された未終了の打刻は永久に走り続ける幽霊になる**。これが14件の正体。

## Decision

**付随データの判定は、Kyou 検索が使っているのと同じ規則を通す。**

- 削除済みの除外を `livePlaingTimeIsCandidates` に、覆っているかの判定を `timeIsCoversMoment` に切り出し、
  ハンドラのループから追い出す。どちらも純関数なので単体で固定できる
- 判定の作法は `find_filter.go` の削除済み除外と同じ「ID ごとの最新版の `IsDeleted`」。
  `FindTimeIs(OnlyLatestData:true)` が既に ID ごと1件へ畳んでいるので、受け取った行の `IsDeleted` が最新版のもの
- 付随打刻のタグは**打刻IDでメモ化**する（ADR-0007 と同じ、リクエスト単位のメモ化）

**`find_filter.go` の「IsDeleted は使わないこと」は、レコードの `.IsDeleted` を読むなという意味ではない。**
あれは *`FindQuery` の旗として* 使うなという話（`git_commit_log_repository_local_dir_impl.go` が
`IsDeleted=true` を「削除済みのみ検索」という**逆の意味**で読むため）で、
レコードの `.IsDeleted` を読むのは同ファイルの削除済み除外が現にやっている。この区別をコメントに残す。

## Rejected alternatives

- **`is_deleted` を付随打刻の DTO に足して、判別は呼び出し側に任せる** — 「記録時に何が走っていたか」を
  知る機能なので、削除済みが混じった時点で答えが壊れている。呼び出し側の全員が必ず見る保証も無い。
  実際レビューは16件をそのまま「その時の状況」として読んでいた
- **`FindTimeIs` の側で削除済みを落とす** — `FindTimeIs` は削除済みの検索にも使われる共有経路で、
  ここで落とすと `include_deleted_data` 系の呼び出しが壊れる。除外はいつも呼び出し側の責務
  （ADR-0071 が `find_filter.go` に置いたのと同じ理由）
- **付随打刻も `FindFilter.FindKyous` を通す（Web と同じ経路に寄せる）** — 1ページの Kyou それぞれの
  `related_time` について検索を投げることになり、20件のページで20回の全文検索になる。
  現在の「一度だけ引いて Go 側で照合」のほうが桁で速い。**寄せるのは規則であって経路ではない**
- **取得時に期間で絞って全件走査をやめる** — 走査対象は本番で28,435件あり、
  `Kyou数 × 28,435` 回の比較になる。しかし絞るには「ページの時間範囲を覆う打刻」を SQL で表す必要があり、
  `FindQuery` にあるのは点の `PlaingTime` だけで範囲版が無い。期間で素直に絞ると
  **窓より前に始まった打刻が落ちる**（それこそが「走っている」打刻）。SQL の新設が要るので今回は見送り、
  削除済みの除外（16→2）とタグのメモ化で実害を消した
- **付随打刻を共有テーブル（`timeis_ids[]` + 応答末尾の表）にする** — 重複の解消としては正しいが、
  削除済みが落ちれば件数が16→2になり重複の実害がほぼ消える。効果を測ってから判断する

## Consequences

- **付随打刻の件数が実測で 16 → 2 に減る。** 応答サイズと `max_size_mb` の消費も同じだけ減る
- タグ引きの呼び出しが `Σ(Kyouごとの一致件数)` から「一致した打刻の種類数」になる
  （1ページ20件・1件16打刻なら320回 → 数回）
- `TimeIsMCPDTO` に `is_deleted` は**足さない**。削除済みは返さないので、あれば嘘になる
- **共有ページの実体側（`handle_get_shared_kyous.go` の `AttachedTimeIss`）は、当初の調査で
  「落ちる」と書いたが実際には落ちていなかった。** Kyou リストは `FindFilter` 経由なので落ちる一方、
  打刻の実体は `TimeIsReps.FindTimeIs` を直叩きしており、MCP とまったく同じ欠落を持っていた。
  同日中に `livePlaingTimeIsCandidates` を通す形へ寄せた（同時に、先に無条件代入してから
  自分自身と `UpdateTime` を比べていた版選択のデッドコードも除いた）
- 判定が純関数2つになったので、規則を戻すと必ずテストが落ちる（下記）

## Evidence

本番実測（2026-08-25、記録数の多い実アカウント）:

- `is_include_timeis:true` で1件の kmemo に付いた打刻が16件。`plaing_time` に同じ瞬間を渡すと2件
- 混ざっていた打刻のひとつを `gkill_get_kyou_history` で引くと `latest_is_deleted: true`、
  `end_time: null`、開始は2025-08-14（1年前）
- 生存する `timeis_start` の全期間件数は28,435件。`is_include_timeis:true` は毎回この全行を読む

修正の検証: `livePlaingTimeIsCandidates` の `IsDeleted` スキップを外すと
`TestLivePlaingTimeIsCandidates_DropsDeleted` と `..._AllDeleted` が落ちる（実施済み）。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`
  - `TestLivePlaingTimeIsCandidates_DropsDeleted`（幽霊が今日の記録を覆うことも同時に固定）
  - `TestLivePlaingTimeIsCandidates_AllDeleted`
  - `TestTimeIsCoversMoment`（`plaing_time` の SQL と同じ意味であること）
