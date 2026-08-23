# ADR-0056: タグ・テキスト・通知・GPSログの格納先は別枠で返す — Reps へ混ぜない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | 2026-08-24 の MCP 再監査（P-01）。gkill-go-backend スキルの「rep名の絞り込みは『検索するrep』ではなく『検索結果』でやる」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/handle_get_rep_infos_mcp.go` / `src/server/gkill/api/req_res/get_rep_infos_mcp_response.go`（`AttachedDataRepInfoMCPDTO`） |

## Context

`gkill_add_tag` / `gkill_add_text` は書き込み先を引数に取らず、サーバが決める。
その決まった先（`Tag.db` / `Text.db` など）が**どの一覧にも出てこなかった**。

- `get_all_rep_names` は 10件（Kyou を生む rep）しか返さない
- `get_rep_infos` の `rep_infos[]` も同じ集合
- `ApplicationConfig.rep_struct` は null（設定画面で「適用」を押すまで書かれない）

書き込み応答の `rep_name` に `"Tag"` が入るので**事後には分かる**が、
「これから書くとどこへ行くのか」は書いてみるまで分からなかった（2026-08-24 の再監査 P-01）。

`Reps` は「Kyou 検索の対象 ＝ 利用者が選べる記録保管場所」という一本の意味に固定されている。
`gkill_dao_manager.go` に明示のコメントがあり、**Kyou を1件も返さないプラグインを
`Reps` に入れない**理由として「一覧に並び、選んでも0件の項目になる。検索のたびに
空振りの往復も1回発生する」と書かれている。

## Decision

**`get_all_rep_names` と `rep_infos[]` は触らない。`get_rep_infos` の応答に
別枠の `attached_data_reps[] {rep_name, data_kind}` を足す**
（`data_kind` は `tag` / `text` / `notification` / `gpslog`）。

説明文に「**これは `query.reps` の値ではない**」を明記する。
収集は各コレクションを `UnWrapTyped()` して leaf の名前を集めるだけで、
ADR-0001 が許容する「名前を列挙するためだけの `UnWrap`」に収まる。

## Rejected alternatives

- **`find.KyouRepTypes` に tag / text / notification / gpslog を足す** —
  `api/rep_types_coverage_test.go` が
  「`RepsOfKyouRepType("tag"/"text"/"notification"/"gpslog")` は0件」を要求しており即座に落ちる。
  この設計は明示的に固定されている
- **`Reps` へアダプタ経由で入れて `get_all_rep_names` に出す** —
  Web の `application-config.ts` が `get_all_rep_names` の結果から
  `rep_struct` と `rep_type_struct` を組み立て、**「適用」で DB へ永続化する**。
  `"Tag"` が入り込むと `query.rep_types` に載り、以後の検索で毎回
  「未知の rep_type」警告が出続け、**自動では消えない**。
  さらに「選んでも0件」の項目が rykv のツリーに並ぶ ——
  emits_kyou=false のプラグインで既に避けた事象そのもの
- **`rep_infos[]` に `rep_type:"tag"` として混ぜる** — 呼び出し側が素直に
  `query.reps` へ渡す。Kyou の `RepName` が `"Tag"` になることは無いので、
  **エラーも警告も無く0件**になる
- **`/api/get_repositories` を使わせる** — `RepName` が空で返るので使えない
  （`repository.File` を消してから返しており、`RepName` はその `File` 由来）
- **`ApplicationConfig.rep_struct` をサーバ側で合成して返す** —
  `get_application_config` は `wrapAuth` で登録されており rep 一覧を持たない。
  `wrapAuthRepos` へ変えると、設定取得のたびにリポジトリ解決が走る
  （「読み込みが一生終わらない」の原因になった経路）。
  発見の口は本 ADR の別枠フィールドで開くので、そちらは説明文で
  「null は設定画面で一度も適用していないだけ」と書くに留める

## Consequences

- 「タグはどこへ書かれるか」が**書く前に**分かる
- `rep_infos[]` の意味は変わらない（絞り込みに使える値だけが入る）
- Web は `get_rep_infos` を使っていないので影響を受けない
- `rep_struct` は null のまま。MCP のツール説明にその条件を書いた

## Evidence

- `GetAllRepNames` は `g.Reps.UnWrap()` だけを歩く。Tag / Text / Notification / GPSLog は
  `TagReps` / `TextReps` / `NotificationReps` / `GPSLogReps` という別のフィールドに居り、
  インタフェース型も別系統（`TagRepository` は `FindTags` / `UnWrapTyped` を持つ）
- `handle_get_rep_infos_mcp.go` は `find.KyouRepTypes` を回して
  `api.RepsOfKyouRepType()` が返す rep だけを歩くので、原理的に付随データは出てこない
- `add_tag` の応答には `rep_name:"Tag"` が入る（`WriteTagRep.GetRepName()` 由来）

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_rep_infos_mcp_test.go`
  - `TestHandleGetRepInfosMCPListsAttachedDataReps`（別枠で返ること、`rep_infos` と混ざっていないこと）
- `src/server/gkill/api/rep_types_coverage_test.go`（Kyou rep 種別の語彙が変わっていないこと）
