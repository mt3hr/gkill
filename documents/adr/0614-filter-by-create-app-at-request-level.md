# ADR-0614: 「どのアプリが書いたか」の絞り込みは FindQuery ではなく MCP リクエストに置く

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | 2026-08-24 の MCP 再監査（P-39）。ADR-0101 / ADR-0102 が否決した「rep名を SQL へ降ろす」と同じ失敗クラス |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go`（`applyMCPCreateAppsFilter`） / `src/server/gkill/api/req_res/get_kyous_mcp_request.go` |

## Context

`create_app` は**全13テーブルに列があり、全 rep が値を埋めている**。
`gkill` / `gkill_kftl` / `gkill_mcp_readwrite` / `gkill_mcp_write` / `urlog_bookmarklet` / `git` /
プラグインの自称名が実際に入っている。にもかかわらず**引く手段が1つも無かった** ——
検索条件38キーに作成アプリ系が存在しない（2026-08-24 の再監査 P-39）。

「MCP で入れた記録だけ見直したい」「メモ帳から入れたぶんを数えたい」が、
データはあるのにできない状態だった。

## Decision

**`FindQuery` には足さず、`GetKyousMCPRequest` に `create_apps` / `update_apps` を置く。**
絞り込みは `applyMCPDataTypesFilter` の隣に並べた Go 側のポストフィルタで行う。
`nil`=未使用 / 非nil空=0件という `FindQuery` の null 意味論に揃える。

**同時に `KyouMCPDTO` へ `create_app` / `update_app` を出す。**
絞れるのに結果から読めないと、呼び出し側は絞り込みが効いたのかを検証できない。

## Rejected alternatives

- **`FindQuery` に足して `GenerateFindSQLCommon` で SQL へ降ろす** — 一見きれいで、
  SQL の組み立ては1箇所に集約されているので27の rep 実装がタダで対応する。
  だが `target_resolution_memo.go` の `findWordMatchTargetIDs` が、ReKyou / MiReKyou の
  ワード委譲で**利用者のクエリをそのまま下位検索へ渡す**。ここに作成アプリの条件が乗ると
  **「MCP で作ったリポストだが、リポスト先の記録はブラウザで作った」がエラーも警告も無く消える**。
  ADR-0101 / ADR-0102 が rep 名について否決したのとまったく同じ失敗クラス
- **同上（コスト面）** — フィールド数が動くと `verify_docs` の機械検査が連鎖して
  資料8〜10本・クライアント TS・プラグインプロトコル・SDK まで波及する。
  実績として、フィールドを2つ削った `a95648a7` は52ファイルに触れている
- **プラグイン索引側にも実装する** — プラグイン索引のインメモリ照合は元から
  `FindQuery` の一部条件しか実装していない。足しても rep 種別によって
  効いたり効かなかったりする状態が増えるだけ
- **`create_app` を DTO に出すだけで絞り込みは付けない** — 呼び出し側が全件取って
  自分で数えることになり、件数もページングも壊れる
- **`update_app` は付けない** — 「編集したクライアント」で引きたい場面は
  「作成したクライアント」と同じくらいあり、実装コストはほぼゼロ

## Consequences

- 絞り込みは MCP 経由の検索でだけ効く。Web の検索には出ない（Web は自分が何で書いたか知っている）
- ポストフィルタなので、`limit` や `max_size_mb` の適用より前に絞られる位置に置く必要がある
  （`applyMCPDataTypesFilter` の直後に並べてある）
- `KyouMCPDTO` に2フィールド増えるが `omitempty` は付けない
  （空と「フィールドが無い」が区別できなくなる。外部監査 C4 / C5 と同じ理由）

## Evidence

- `CREATE_APP` 列は実データ13テーブル・キャッシュ14テーブル・temp すべてに `NOT NULL` で存在する
- `reps.Kyou.CreateApp` は全 rep が埋めている（git は `"git"`、
  アップロードは `"gkill"`、ブックマークレットは `"urlog_bookmarklet"`、
  KFTL は `"gkill_kftl"`、MCP は `"gkill_mcp_write"` / `"gkill_mcp_readwrite"`）
- 先例と明文の理由: `get_kyous_mcp_helpers.go` の `applyMCPDataTypesFilter` に
  「FindQuery に足さずリクエストレベルなのは、和集合仕様の rep_types（Web の Mi 画面が依存）を
  触らずに Mi/MiReKyou/プラグインを直接絞る口を作るため（外部監査 S5/A2）」とある

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`
  - `TestApplyMCPCreateAppsFilter` / `TestApplyMCPUpdateAppsFilter`
- `src/mcp/__tests__/normalization.test.mjs`（null 意味論と旧スキーマ救済）
