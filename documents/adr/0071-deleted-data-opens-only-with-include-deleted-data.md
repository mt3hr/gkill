# ADR-0071: 削除済みの Kyou は include_deleted_data だけが開ける（is_deleted は使わない）

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `ae969ce8` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find_filter.go`（削除済みの除外） |

## Context

`FindFilter` は「最新版が `IsDeleted` の ID」を**フラグ分岐なしに無条件で**結果から落としていた。
そのため「今日なにを消したか」を検索で数える手段が存在しなかった。

`FindQuery` には `IsDeleted` フィールドが定義されているが、`api/find/README.md` の
「未実装フィールド」表に載っており、Kyou 検索では読まれない。
にもかかわらず MCP のスキーマは `is_deleted: "Include soft-deleted entries."` と説明しており、
**事実に反する記述**になっていた。

ADR-0054 で1件ずつ読み返す手段（`gkill_get_kyou_history`）は用意したが、
それは ID を知っている場合の経路で、列挙はできない。

## Decision

`find_filter.go` の削除済み除外を `IncludeDeletedData`（JSON `include_deleted_data`）でゲートする。
既定（false）は従来どおり除外。`IsDeleted` は Kyou 検索では今後も読まない。
`KyouMCPDTO` に `is_deleted` と `update_time` を足し、返ってきた行を判別できるようにする。

## Rejected alternatives

- **`IsDeleted` を実装する** — 唯一の読み手である
  `git_commit_log_repository_local_dir_impl.go` が `if query.IsDeleted { continue }` と書き、
  コメントで「gitコミットに削除の概念はないため、削除済み検索(IsDeleted=true)には該当しない」と
  明言している。つまりそこでは `IsDeleted=true` が **「削除済みのみを探す」という逆の意味**。
  「含める」として実装すると、同じ旗が層によって反対の意味を持つ
- **三値（`*bool`）の新フィールドを足す** — `FindQuery` のフィールド数は `verify_docs` が
  機械検査しており（現在41）、資料側の連鎖が動く。`IncludeDeletedData` は既に
  `tag_repositories.go` / `text_repositories.go` / `notification_repositories.go` で
  「削除済みも含める」の意味を持っているので、新設せずに Kyou 検索へ広げるほうが
  1つの旗が全層で1つの意味を保てる
- **削除済み専用の API を足す** — 検索条件（期間・タグ・語）は生きている記録と同じものを使いたい。
  専用 API にすると `FindQuery` の語彙をもう一組維持することになる
- **MCP の `is_deleted` プロパティを削除する** — 送っているクライアントが未知キーで
  throw するようになる。「受理するが無視される」と説明を直すに留める

## Consequences

- **`find_filter.go` は gkill で最も熱い関数**で、rykv / mi / dashboard / Web UI / 全 MCP が通る。
  コードは `if` 1つだが影響範囲は全画面。既定 false を厳格に担保する必要がある
- 既定 false の担保: クライアントの `FindKyouQuery` に該当フィールドが無いので JSON から落ち、
  Go のゼロ値になる。Go 側に Kyou 検索でこれを立てる呼び出し元も無い
- **旗を立ててもタグ/テキストは漏れない。** `find_filter.go` は利用者のクエリを
  `TagReps.FindTags` / `TextReps.FindTexts` へ渡しておらず、呼び出しごとに新しい `FindQuery` を組む
- **rekyou / mirekyou はこの旗の対象外。** rep の内部で削除済みを弾いており、
  削除済みを指す rekyou も落とす。`git_commit_log` には削除の概念が無い
- `KyouMCPDTO` の `is_deleted` / `update_time` に `omitempty` を付けてはいけない。
  false / ゼロ値で消えると「生きている」と「フィールドが無い」の区別が付かなくなる
  （`IsZip` / `Addition` / `Deletion` と同じ理由）
- **キャッシュ ON / OFF の両方でテストすること。** 削除の判定はどちらの経路でも同じでなければならない

## Evidence

2026-08-23 の readwrite MCP 監査（実測）:

- D-1 の件数が、通常検索でも `is_deleted:true` を付けても **どちらも20件**（増えない）
- 削除した mi をボード検索 + `mi_check_state:"all"` + `is_deleted:true` で引いても出ない

修正の検証（実測）: ゲートを `if true` に潰すと
「include_deleted_data を立てても削除済みのKyouが出ない」がキャッシュ ON / OFF の両方で落ちる。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_kyous_mcp_test.go`
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go`
