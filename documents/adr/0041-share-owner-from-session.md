# ADR-0041: 共有情報の所有者はリクエスト本文ではなくセッションから決める

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-03 |
| Sources | `f944c566` / `src/server/gkill/api/gkill_server_api/handle_update_application_config.go` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/handle_get_shared_kyous.go` |

## Context

共有情報（共有URLの実体）を保存するとき、**リクエスト本文の `user_id` をそのまま保存していた**。

共有ページは `share_id` だけで認証される公開エンドポイントなので、**一般アカウント1つあれば、他人を指す共有を作るだけで、そのユーザーのライフログ全体を認証なしで読み出せた**。

## Decision

共有情報の所有者は**セッションから決める**。リクエスト本文の `user_id` は所有者の決定に使わない。

## Rejected alternatives

- **本文の `user_id` がセッションのユーザーと一致するか検証する** — 検証を1箇所書き忘れれば同じ穴が開く。**そもそも本文から取らない**ほうが、書き忘れが起きる場所自体が無くなる。

- **共有ページ側で所有者を再検証する** — 共有ページは `share_id` しか持たない（セッションが無いのが仕様）。再検証する材料が無い。

- **管理者だけが他人の共有を作れるようにする** — 用途が無い。共有は自分の記録を見せるための機能。

## Consequences

「利用者が指定した識別子を、権限の根拠に使わない」が原則として立つ。同じ形の判断が派生キャッシュのパス（→ ADR-0044）や共有ファイル配信（→ ADR-0042）にも要る。

## Evidence

実測なし — 脅威モデルからの判断（一般アカウント1つで他人の全記録が読める状態だった）。

## Related tests

- `src/server/gkill/api/gkill_server_api/handle_get_shared_kyous_test.go`
