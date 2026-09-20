# ADR-0111: 実装されなかった検索条件は消す — 受理して無視するより未知キーとして弾く

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | ADR-0110 の却下案を1つ覆す |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/find/find_query.go` / `src/server/gkill/mcp/constants.go`（`KYOUS_QUERY_*_FIELDS`） |

## Context

`FindQuery` には「定義だけあって検索処理が一度も参照しないフィールド」が2つあった。
`api/find/README.md` にも「未実装フィールド」という節を作って明記していた。

| フィールド | 実態 |
|---|---|
| `HideTimeIsTags` | 読み手が1つも無い。定義とテスト literal だけ |
| `IsDeleted` | Kyou 検索では読まれない。唯一の読み手 `git_commit_log_repository_local_dir_impl.go` が **「削除済みのみを探す」という逆の意味**で読み、0件を返す |

MCP のスキーマは `is_deleted` を "Deprecated and ignored for Kyou searches" と説明し、
`hide_timeis_tags` は "Explicit TimeIs tag exclusion list" と、**効くかのように**説明していた。

ADR-0110 で削除済みの列挙は `IncludeDeletedData` に一本化したが、
紛らわしい `IsDeleted` はそのまま残していた。
残した理由は「送っているクライアントが未知キーで throw するようになる」だったが、
これは**送っているクライアントがいるかどうかを確かめていない**推測だった。

## Decision

`FindQuery` から `IsDeleted` と `HideTimeIsTags` を削除する（41→39 フィールド）。
MCP の語彙（`KYOUS_QUERY_BOOLEAN_FIELDS` / `KYOUS_QUERY_STRING_ARRAY_FIELDS` /
`find_query_schema.go`）からも外す。送られたら `normalizeKyouQuery` が未知キーとして throw し、
`detail.allowed` に正しい候補（`include_deleted_data` を含む）を添えて返す。

`git_commit_log_repository_local_dir_impl.go` の逆意味の分岐は削除する。
契約コメントを「git コミットに削除の概念が無いため、削除に関する検索条件は一切効かない」へ改める。

**プラグインプロトコルの `IsDeleted` は残す。** `findQueryToPluginQuery` が常に `false` を渡す形にし、
`plugin_protocol.go` の doc に「gkill 本体からは常に false が渡る」と明記する。

## Rejected alternatives

- **受理して黙って無視する現状のまま残す（ADR-0110 の判断）** — その判断の前提
  「送っているクライアントがいるかもしれない」が実測で崩れた（Evidence 参照）。
  さらに、黙って無視するのは**監査が問題にした「静かに壊れる」そのもの**で、
  AI は「フィルタを掛けたつもり」で結果を読んでしまう。
  未知キーとして弾けば「そのフィルタは存在しない」と即座に学べる。**うるさいほうが正しい**
- **`LEGACY_USE_FLAG_KEYS` のような「受理して捨てる」バケツへ入れる** — あのバケツは
  旧 `use_X` フラグを**実際に送っている古いクライアントが存在した**ための移行装置。
  送信元が実在しないものを同じ装置に載せると、装置の意味が
  「後方互換のため」から「なんとなく残す場所」へ薄まる
- **`IsDeleted` を「削除済みのみ」の意味で実装する** — `IncludeDeletedData` と意味が重なり、
  1つの旗が層によって反対の意味を持つ状態が固定される（ADR-0110 の議論と同じ）
- **プラグイン SDK の `sdk.Query.IsDeleted` も消す** — サードパーティ製プラグインが見る公開 API で、
  `src/plugins/` の4モジュール（それぞれ独立 go.mod、`replace` 参照）の tidy も要る破壊的変更になる。
  常に `false` を渡す1行で済むものに、外部への破壊を払う理由が無い

## Consequences

- **スキーマをキャッシュしている MCP クライアントが `is_deleted` を送るとエラーになる。**
  これは意図した挙動だが、旧スキーマはセッション寿命で固定されるので
  （`project-mcp-stale-schema-cache` の事情）、切り替え直後に一度だけ起きうる
- `FindQuery` のフィールド数は `verify_docs` が機械検査する。41→39 にすると
  `api/find/README.md` と `api/README.md` の3行が連動する
- **git の rep には削除に関する条件が一切効かなくなった**（従来 `IsDeleted=true` で0件を返していたのが、
  条件そのものが無くなった）。git コミットに削除の概念が無いので実害は無いが、
  契約コメントを読む側には「効かない」と明示されている必要がある
- プラグインプロトコルの `IsDeleted` は永久に `false` になる。
  プラグイン側でこの値に分岐を書いても意味が無いことを `src/plugins/README.md` にも書いた

## Evidence

2026-08-24 の実測。「送信元が実在しない」を3方向から確かめた。

- **コード**: `src/client/classes/api/find_query/` に `hide_timeis_tags` / `is_deleted` が
  **全 git 履歴で一度も存在しない**。出荷済みバンドルにも0件
- **永続化されたクエリ**: 稼働中インストールの `$GKILL_HOME/configs/` を読み取りのみで確認。
  `share_kyou_info.db` と `user_config.db` のどちらにも両キーが **0件**。
  対照として `timeis_tags` はそれぞれ 2件 / 4件ヒットする。
  `hide_timeis_tags` は `timeis_tags` を部分文字列に含むので、存在すれば対照カウントに必ず出る
- **読み手**: `query.IsDeleted` を読む非テストコードは4箇所だけで、うち2つは
  `&find.FindQuery{IsDeleted: false, ...}` というゼロ値の明示（挙動に影響しない）。
  `HideTimeIsTags` は定義とテスト literal の2箇所のみで、読み手ゼロ

## Related tests

- `src/server/gkill/api/find/find_query_test.go`
- `src/server/gkill/dao/reps/git_commit_log_repository_local_dir_impl_test.go`
- `src/server/gkill/mcp/normalization_test.go`
- `src/server/gkill/mcp/constants_test.go`
