# ADR-0625: 地図条件は MCP 入口で3値を要求し、Go は欠けを警告する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（`map_latitude` だけ指定しても通常検索と同じ 275 件が警告なしで返る）。`.claude/skills/gkill-find-query/SKILL.md` の `HasMapFilter`（3値すべて非 nil）。`.claude/skills/gkill-mcp/SKILL.md` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/normalization.go`（`assertMapFilterComplete`）/ `src/server/gkill/api/gkill_server_api/get_kyous_mcp_helpers.go`（`partialMapFilterWarning`） |

## Context

`FindQuery.HasMapFilter()` は緯度・経度・半径の3つが揃って初めて真で（ADR-0106 の null 意味論）、欠けると
地図条件ごと**黙って無視**される。内部の意味論としては正しいが、AI から見ると「条件を指定したのに効かなかった」
ことが応答のどこにも出ない。半径の単位（メートル。`find_filter.go` で `/1000` して km にする）もどこにも
書かれていなかった。CLI の `add_tag` には同じ不揃いを拒否する検査が前からあった。

## Decision

MCP の入口（`normalizeKyouQuery`）で、3値のうち1〜2個しか無ければ欠けた欄を名指しして拒否する。
範囲（緯度 −90..90、経度 −180..180、半径 > 0。半径 0 以下は Go が黙って素通しする）も入口で弾く。
Go 側の `collectMCPUnknownValueWarnings` にも `partialMapFilterWarning` を足し、API を直接叩く経路には
警告で伝える。単位はスキーマ・ヘルプ・警告文の全部に「メートル」と書く。

## Rejected alternatives

- **欠けた値を既定で補う（半径既定 500m 等）** — 呼び出し側が指定していない条件で結果を絞ることになり、
  「条件を書いたつもりが別の条件で検索された」に変わるだけ。
- **Go の `HasMapFilter` を「1つでも非 nil なら有効」にしてエラーにする** — Web の検索条件 JSON は世代がまばらで
  （`project_find_query_json_generations`）、古い保存条件に半径だけ残っているものが一斉に落ちる。
  意味論は変えず、MCP の入口と警告で扱う。
- **警告だけ（拒否しない）** — 結果は無関係な記録のままで、AI はそれを正常な検索結果として使ってしまう
  （報告のとおり「エラーより気づきにくい」）。

## Consequences

- MCP から地図条件を使うときは3値必須。既存の呼び出しで1〜2値だけを送っていたものはエラーになる
  （それらは元から地図条件が効いていなかった）。
- Go の警告は API 直叩きの網で、MCP 経由では入口が先に止めるので出ない。

## Evidence

- 報告値: `map_latitude` のみで 275 件（キーワード指定なしと同数）、警告なし。
- 修正後: `map_latitude` のみ → `query.map_longitude is required when query.map_latitude is set ... Missing: map_longitude, map_radius`。
  Go 単体: `query.map_latitude is set but query.map_longitude / map_radius is missing ... map_radius is in meters`。

## Related tests

- `src/server/gkill/mcp/normalization_test.go`（「rejects a partial map filter」「rejects out-of-range map values」）
- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_v2_test.go`（`TestPartialMapFilterWarning`）
