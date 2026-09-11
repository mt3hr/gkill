# ADR-0615: 書き込みに刻む user は、その要求を認証したセッションから決める

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-25 |
| Sources | 2026-08-25 の実利用レビュー（別のクライアントから書いたのに `create_user` が想定と違う／読み取りサーバのアカウント名とレコードの `create_user` が食い違う）と、本番アカウントでの実測 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/mcp/lib/mcp-server-base.mjs`（`handleToolCall` の userId 決定） |

## Context

レビューが2つの症状を報告した。

1. あるクライアントから MCP 経由で書いたのに、`create_user` が別のアカウント名になっている
2. 読み取りサーバの接続アカウント名と、レコードの `create_user` が一致しない

どちらも「特定クライアント専用時代の固定値が残っている」と診断されたが、**そのアカウント名を
ハードコードした箇所は `src/mcp` にも `src/server` にも1つも無い**。1 は接続に使ったアカウントが
実際にその名前だっただけで、正しい値だった。2 も、そのレコードを別端末の別アカウントが書き、
集約アカウント側が読んでいるだけで、正しい。

ただし**症状2を作れるコード経路が別に実在した**。gkill の `/api/add_*` は provenance 3欄
（`create_app` / `create_device` / `create_user`）をクライアントの自己申告のまま保存し、
サーバは一切上書きしない（Web UI も Wear OS も同じ性質）。その値を組み立てる MCP 側で、

```js
const userId = (ctx ? ctx.userId : this.currentUserId) || this.client.userId;
```

と書かれていたため、**「どのアカウントの DB へ書くか」（`sid`）と「レコードに刻む名前」（`userId`）が
別々の出どころ**になっていた。OAuth トークンに `userId` が乗っていないと、`sid` はトークンの
セッション（＝別アカウント）のまま、名前だけ環境変数 `GKILL_USER` へ落ちる。
つまり **別アカウントのセッションへ書いたレコードに、手元の user 名が焼かれる**。

エラーにはならない。書き込みは成功し、`create_user` に嘘が入るだけなので、
「誰が書いたか」で追跡を始めるまで誰も気づかない。

## Decision

**`sid` がある（＝認証済みセッション）ときは、`userId` も同じ認証から取れていなければ書き込みを断る。**
環境変数 `GKILL_USER` へのフォールバックは、`sid` を持たない stdio 接続にだけ残す。

## Rejected alternatives

- **Go 側の `handle_add_*` / `handle_update_*` で `auth.UserID` を強制的に上書きする** —
  自己申告をやめれば根本解決に見えるが、同じ経路を Web UI・Wear OS・他端末からの同期が使っており、
  取り込み系（git / IDF / plugin / add_tag）は**意図的に他人名義で書く**（取り込み元の作者を残す）。
  18本のハンドラすべての意味を変える変更になるので、MCP の穴を塞ぐ対価としては大きすぎる
- **フォールバックを黙って外し、`userId` が空なら空のまま書く** — `create_user:""` のレコードが
  できるだけで、追跡不能になるのは同じ。しかも空文字は検索条件としては「未使用」に見えるので、
  あとから探し出すこともできない
- **stdio でも `userId` を必須にする** — stdio は環境変数（`GKILL_USER`）で接続する形態そのもので、
  `sid` も `ctx.userId` も持たない。必須にすると stdio 接続が全部書けなくなる
- **クライアント名（`initialize` の `clientInfo`）を `create_user` に混ぜる** — 「どのクライアントが
  書いたか」を残したい気持ちは分かるが、`create_user` は gkill のアカウント名で、検索条件にも使われる語彙。
  そこへ別概念を混ぜると、既存レコードとの照合が壊れる。残すなら `create_app` 側
  （自由文字列で allow-list が無い）に `gkill_mcp_readwrite/〈クライアント名〉` のような形で足すのが筋で、
  これはワイヤ契約を変えずにできる。今回は入れていない

## Consequences

- OAuth トークンに `userId` が無いクライアントは、**書き込みだけ**が明示的なエラーで止まる。
  読み取りは従来どおり動くので、再接続すれば直る。**黙って嘘の名前で書くよりは止まるほうがよい**
- `create_user` が接続アカウントと一致することを、MCP の外側（Web UI・同期）には保証できない。
  gkill 全体としては自己申告のままである
- **特定のクライアント名をコードから探しにいかないこと。** 「あのクライアント専用の残骸では」という
  指摘は、この ADR を知らないと何度でも再発する。実体は「接続に使ったアカウント名がそのまま
  出ている」だけで、コードには無い

## Evidence

- provenance に関わるクライアント名のハードコード: 全文検索（`src/mcp` + `src/server`）で **0件**
- 本番の履歴で確認したレコード: `create_app` は `gkill_kftl`、`create_device` は書いた端末名、
  `create_user` はその端末のアカウント ―― 集約アカウントから読んでも、書き手は元のアカウントのまま
- `create_device` は MCP では `"mcp"` 固定（`write-handlers.mjs`）。KFTL 経由だけはサーバ側が
  決めるので、`create_app` が `gkill_kftl`、`create_device` はサーバのデバイス名になる

## Related tests

- `src/mcp/__tests__/write-handlers.test.mjs`（`create_app` は `ctx.appName` から来る＝サーバ種別ごとに違う）
- `src/mcp/__tests__/http-transport.test.mjs`（並行リクエストで user / session が混ざらないこと）
