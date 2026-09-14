# ADR-0410: 複数書き込みになる画面の操作は tx_id で束ねて commit_tx で確定する

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-15 |
| Sources | 2026-09-13 外部レビュー #4 / `.claude/skills/gkill-client-tags/SKILL.md`「Kyou の追加/編集画面のタグ欄」節 / `.claude/skills/gkill-client-foundation/SKILL.md` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/client/classes/gkill-tx.ts` / `src/client/classes/kyou-tags.ts` / `src/client/classes/cascade-delete-kyou.ts` |

## Context

Web の追加画面（kc / lantana / nlog / timeis / urlog / mi / mirekyou の7本）と編集画面（10本）は
「本体を `add_*` / `update_*` → タグを `add_tag` / `update_tag`（Mi は通知も）」と複数の API を順に投げていた。
連鎖削除（`cascade-delete-kyou.ts`）はタグ・テキスト・通知・ReKyou・MiReKyou・本体を1件ずつ `update_*` で
消していた。どれも途中で失敗すると一部だけが書かれる。

`tx_id` で束ねる仕組み（temp rep + `commit_tx`）は KFTL のために最初からあったが、ADR-0403 は
「TXID 指定時の `add_tag` は `added_tag` を返せず `registered_tag` を上げられない。しかも `commit_tx` は
DB トランザクションではなく部分確定しうるので、束ねても原子性は買えない」として却下していた。
2026-09-15 に `commit_tx` が1つの SQLite トランザクションになり（ADR-0219）、後者の理由が消えた。
ユーザーの指示は「フロント側の Add や Delete に tx 考慮を追加できるならそれがいい」。

## Decision

**複数の書き込みになる画面の操作は、全部を1つの `tx_id` で temp rep に積み、`commit_tx` で確定する。**
共通ヘルパは `gkill-tx.ts` の `run_in_tx`（work が失敗／throw → `discard_tx`、通れば `commit_tx`、commit が
失敗しても `discard_tx`）。tx 中の応答には Kyou もタグも載らないので、タグは**クライアントで組み立てた Tag**を
そのまま `registered_tag` / `deleted_tag` に使い、Kyou は commit 後に `fetch_committed_kyou`（SW キャッシュ削除 →
`get_kyou`）で引き直してから `registered_kyou` / `updated_kyou` を上げる。ADR-0403 の「add_tag が終わってから
registered_kyou」は「commit が終わってから」に読み替える。

18 画面は書き込みが1件でも常に tx を通す。単発の1 API 操作（コンテキストメニューのタグ追加、テキスト・通知の
単独の追加/編集/削除）は tx にしない。連鎖削除は探索に失敗したら1本も積まず、失敗時の `deleted_ids` は空
（1つのトランザクションなので部分成功が無い）。

## Rejected alternatives

- **サーバに `/api/delete_kyou`（連鎖削除）を新設する** — 削除だけ別の原子性機構になる。`update_*` は
  すでに `tx_id` を受けるので、クライアントで束ねれば同じ仕組みで足りる。
- **書き込みが1件だけのときは tx を通さない分岐を残す** — 「1件なら直接、2件なら tx」の2経路になり、
  テストと約束が倍になる。往復は commit と引き直しの分だけ増えるが、保存操作なので許容。
- **tx 中の `add_tag` の応答から `added_tag` を取る** — 一時リポジトリにしか無いので取れない（`handle_add_tag.go`）。
  サーバは渡した内容をそのまま書くので、クライアントの Tag で足りる。`registered_tag` の消費側は `tag.tag`（名前）と id しか見ない。
- **タグ履歴（前回のタグ・履歴チップ）を `add_tag` の成功時に積む** — tx 中はまだ付いていない。commit 後に
  `record_added_tag_history` で積む。積んでから失敗すると履歴だけが動く。
- **連鎖削除で失敗した行だけを画面に残し、消せた行を `deleted_ids` に入れる** — 以前の設計。トランザクションなので
  部分成功が無く、失敗したら全行を残す。
- **連鎖削除の探索エラーを警告扱いで続行する** — 見えていない参照元を残したまま本体だけ消すと、参照先の無い
  リポストが残る。探索に失敗したら1本も積まない。

## Consequences

- 保存の往復が commit と引き直しの分だけ増える（追加: `add_*` + `add_tag`×N + `commit_tx` + `get_kyou`）。
- `commit_tx` の失敗は `FAILED_SAVE_NOTHING_SAVED_MESSAGE`（何も保存されていない）、連鎖削除の失敗は
  `FAILED_DELETE_KYOU_NOTHING_DELETED_MESSAGE`（何も削除されていない）で、文言が事実と一致する。
- KFTL の `do_submit` も commit 失敗時に `discard_tx` する（以前は呼んでおらず temp に残っていた。commit が
  temp を消費するようになったので、成功時の discard は要らない）。
- 単体テストの gkill_api モックには `generate_uuid` / `commit_tx` / `discard_tx` / `get_kyou`（`kyou_histories`）が要る。
  無いと `run_in_tx` が throw し、画面は「保存に失敗」の経路に落ちる（テストは静かに通らない）。
- `want_response_kyou` は tx 中は送らない（実 rep を読み直すだけで tx の内容は載らない。往復の無駄）。

## Evidence

実測なし — 構造からの判断（ADR-0219 で commit が原子になったこと、`update_*` / `add_*` の 26 リクエストが
Go / TS ともに `tx_id` を持つこと）。

## Related tests

- `src/client/__tests__/unit/classes/gkill-tx.test.ts`
- `src/client/__tests__/unit/classes/cascade-delete-kyou.test.ts`（「全件を同じ tx_id で積み、commit_tx を1回だけ呼ぶ」ほか）
- `src/client/__tests__/unit/composables/add-views.test.ts`（「registered_kyou は add_tag と commit_tx が終わってから emit される」「タグの追加が失敗したら discard_tx して…」）
- `src/client/__tests__/unit/composables/edit-views.test.ts`（「タグの追加が失敗したら discard_tx して…」）
- `src/client/__tests__/unit/classes/use-confirm-delete-kyou-view.test.ts`
