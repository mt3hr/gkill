# ADR-0040: パスワードは Argon2id で保存し、ワイヤ形式（password_sha256）は変えない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-03 |
| Sources | `67e2f0c9` / `75367089` / `525037e3` / `ed72dc6d` / `documents/reverse/design-philosophy.md`「パスワードを Argon2id で保存する判断」 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/account/account.go` |

## Context

パスワードは**無塩SHA-256をそのまま保存**していた。`account.db` を読めた者は、レインボーテーブルで元のパスワードを引くまでもなく、**そのハッシュをそのまま送れば全アカウントになりすませる**（ワイヤ形式が同じなので）。

一方でワイヤ形式（クライアントが `password_sha256` を送る）は Web / MCP / Wear OS / autolog の4系統に浸透していた。

## Decision

保存を **Argon2id** にする。**ワイヤ形式（`password_sha256`）は変えない。**

既存のハッシュは包み直さず、**全員に再設定させる**。

## Rejected alternatives

- **ワイヤ形式も変えて、生パスワードを送らせる（サーバ側でハッシュ）** — Web / MCP / Wear OS / autolog の4系統すべてを同時に更新することになる。Wear OS は2つのAPKにまたがるプロトコル、autolog は別リポジトリ。**保存の安全性はワイヤ形式を変えなくても得られる**ので、変更範囲を保存側に閉じた。

- **既存の無塩SHA-256を Argon2id で包み直す（`argon2id(sha256(pw))` として移行）** — 移行は無停止でできるが、**包む前のハッシュが漏れていた場合の危険がそのまま残る**（漏れたハッシュをワイヤに載せればログインできる状態が続く）。全員に再設定させるほうが確実。

- **ワイヤ形式が SHA-256 のままなら安全性は上がらない、として何もしない** — 上がる。守っているのは「**`account.db` を読めた者がなりすませないこと**」で、ワイヤ形式とは別の脅威。

## Consequences

**ワイヤ形式が `password_sha256` のままなので、通信路の安全性は TLS に依存する。** これは変更前と同じ。ここを「Argon2id にしたから安全」と読み違えないこと。

既存ユーザは全員パスワードの再設定が必要になった。`reset_password` サブコマンドが URL を発行する。

古いスキーマの `account.db` を新しいコードで開くと**全パスワードが無効化されうる**ので、スキーマ移行のテストを置いてある（`account_schema_migration_test.go`）。

ログインは非存在ユーザとパスワード誤りを**同じ error_code ＋ 文言**に統一し、非存在時も**ダミーの Argon2id を実行する**（ユーザ列挙対策）。Argon2id は意図的に遅いので、実行しないと応答時間でユーザの存在が分かる。

## Evidence

実測なし — 脅威モデルからの判断（`account.db` の読み取り ＝ 全アカウントのなりすまし、という状態を解消する）。

## Related tests

- `src/server/gkill/dao/account/password_hash_test.go`
- `src/server/gkill/dao/account/account_schema_migration_test.go`
- `src/server/gkill/api/gkill_server_api/handle_set_new_password_test.go`
- `src/server/gkill/api/gkill_server_api/handle_reset_password_test.go`
