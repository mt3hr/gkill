# ADR-0510: メモ帳の再送キーは本文の指紋と結果を控え、同じ本文は再生し、別の本文は 409 で断る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-19 |
| Sources | 2026-09-18 の MCP 実利用報告（「同じ再送キーに別の本文を送っても成功扱いになる」「再送で元の作成IDを回収できない」「失敗時 `created` が `null`」「`related_time` だけ小数秒7桁」）。`.claude/skills/gkill-client-kftl/SKILL.md` の冪等キー段落 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/kftl_idempotency.go` / `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text.go`（`kftlSubmissionFingerprint` / `toSubmitKFTLTextCreated`）/ `src/server/gkill/api/req_res/submit_kftl_text_response.go`（`Replayed`） |

## Context

`/api/submit_kftl_text` の `idempotency_key` は Wear のワーカー再送のために入れたもので（監査 S3-wear）、
台帳は「利用者:キー → 成功した時刻」しか持たなかった。2回目以降は本文を見ずに「成功」だけを返し、
`created` は代入されないまま `null` で返っていた。

MCP から使うと次の3つが同時に困る。

- 同じキーで**別の本文**を送っても成功メッセージが返り、その本文は保存されていない（エラーより気づきにくい）
- 1回目の応答を受け取り損ねると、再送しても `created` が空で作成 ID が分からず、後続のタグ付けや取り消しに内容検索が要る
- 失敗時も `created` が `null` で、説明文の「`created[]` が空」と一致しない。`related_time` だけ Windows の時計解像度の小数秒（7桁）が載り、保存値（秒精度）と一致しない

## Decision

台帳の値をキーだけでなく **本文の指紋（`kftl_text` + `"\x00"` + `create_app` の SHA-256）と応答の `created[]`** にする。
同じキーで届いた送信は、指紋が同じなら何も書かずに元の `created[]` を `replayed: true` で返し、
指紋が違えば何も書かずに `ERR000423`（409 `conflict`）で断る。`created` は失敗時も `[]`（非 nil）で返し、
`related_time` は保存層と同じ秒精度へ丸める。TTL（10分）と「成功時だけ記録する」は変えない。

## Rejected alternatives

- **キーを本文のハッシュにする（キーを要求しない）** — 意図的な同一内容の再送（同じ気分値を同じ日に2回）が畳まれて記録できなくなる。
  `gkill-mobile` スキルが前から禁じている形。
- **別の本文が来たら黙って新規登録する（キーを付け替えたとみなす）** — 「キーを使い回した」のは呼び出し側のバグで、
  黙って通すと再送のつもりの送信が二重登録になる事故（本来の再送）と区別が付かない。断って気づかせる。
- **本文そのものを台帳に置いて比較する** — 利用者の本文を10分間メモリに残す理由が無い。指紋で足りる。
- **`created` を `null` のままにして説明文を直す** — 「配列が空」と「欄が無い」を呼び出し側が場合分けする負担が残る。
  `messages` / `errors` を `[]` に揃えた 2026-09-15 の判断（ADR-0710 周辺）と同じ向きで `[]` に揃える。

## Consequences

- `SubmitKFTLTextResponse` に `replayed` が増える（契約は足すだけ。Wear の kotlinx は `ignoreUnknownKeys = true`、
  Web は `created` しか読まない）。MCP の `gkill_submit_kftl` は `replayed` をそのまま返し、要約文にも出す。
- 呼び出し側は「同じキー = 同じ送信」を守る必要がある。新しい内容には新しいキー。守らないと 409 で止まる
  （止まらずに成功扱いになる 2026-09-19 までの挙動より安全）。
- 台帳はプロセス内・インメモリのまま。再起動をまたぐ再送は畳まれない（従来どおり）。
- `related_time` を丸めるので、Web の `saved_kyou_by_kftl`（最大値の比較）は秒精度で動く。保存値と同じなので影響は無い。

## Evidence

- 報告値（2026-09-18）: 本文A＋キーK → 2件作成、同じ本文A＋キーK → 成功・`created:[]`、別の本文B＋キーK → 成功・`created:[]`・本文B未保存。
- 修正後の end-to-end: 同じキー・同じ本文の2回目は `replayed:true` かつ `created[0].id` が1回目と同じで件数は増えない。
  別の本文は 409・`ERR000423`・件数は増えず、元の本文の件数も変わらない。失敗時の生 JSON は `"created":[]`。

## Related tests

- `src/server/gkill/api/gkill_server_api/kftl_idempotency_test.go`（台帳: 指紋と結果の保持・共有しない・TTL・GC・指紋の入力）
- `src/server/gkill/api/gkill_server_api/handle_submit_kftl_text_test.go`（`TestHandleSubmitKFTLText_IdempotencyKey` / `TestHandleSubmitKFTLText_CreatedRecords` / `TestHandleSubmitKFTLText_CreatedIsEmptyArrayOnFailure`）
- `src/server/gkill/api/message/http_status_test.go`（`ERR000423` = 409）
- `src/server/gkill/mcp/write_handlers_test.go`（`replayed` の受け渡しと要約）
