# ADR-0706: HTTP ステータスはエラーコードから一元表で決める

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `src/server/gkill/api/message/http_status.go` / `.claude/skills/gkill-go-backend/SKILL.md`「HTTP ステータス（2026-08 導入）」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/message/http_status.go` / `src/server/gkill/api/gkill_server_api/write_response_status.go` |

## Context

`/api/*` の JSON ハンドラは**全部が暗黙の HTTP 200** を返していた。エラーは応答本文の
`errors` 配列にだけ入る設計で、`documents/reverse/error-handling-and-security.md` にも
「業務エラーは HTTP 200 で返し `errors` 配列で判別する」と明文化されていた。

実測すると、`response.Errors = append(...)` のサイトは **574箇所**あり、そのうち
ステータスを変えているものは**ゼロ**だった。ステータスを書いていたのは
`filterLocalOnly` / `handle_file_serve` / `handle_browse_zip_contents` /
`handle_urlog_bookmarklet_address` / `recover_middleware` という
**JSON 契約の外側の4経路だけ**で、`/api/*` の JSON ハンドラ92本の中には1つも無かった。

これが次の壊れ方を生んでいた。

1. **セッション切れが「成功」に見える。** 期限切れセッションも無効アカウントも 200。
   ステータスしか見ない層（監視・プロキシ・アクセスログ・素朴な HTTP クライアント）からは
   全部成功に見える。`gkill_server_api_access_log.go` の `responseRecorder` は
   ステータスを記録するが、書き換える者がいないので障害が運用ログに残らなかった。
2. **同じ失敗が経路ごとに違うステータスになる。** `GetRepositories()` の失敗は
   200 / 403 / 500 の3系統、`GetDevice()` の失敗は 200 / 500、「rep 名が見つからない」は
   200 / 404、「非ローカルからの拒否」は 200 / 403 に分かれていた。
3. **`message.EnsureNotEmpty` の doc コメント自体が被害記録になっていた。**
   2026-08-18 に `/api/get_kyous` と `/api/get_kyous_mcp` で
   「HTTP 200 + `errors: null` + 0件」を踏んでいる（[ADR-0105](0105-chunk-find-query-ids.md)）。
   対処は「errors を必ず1件入れる」で、ステータス側は放置されていた。

さらに、`AccountNotFoundError`(ERR000002) が「自分のセッションのアカウントが消えている」と
「操作対象のアカウントが存在しない」の両方に使われていた。クライアントの `check_auth` は
ERR000002 でログアウトさせるので、**存在しないユーザIDにパスワードリセットを実行した管理者が
その場で締め出されていた。**

## Decision

**エラーコード → HTTP ステータスの一元表**を `api/message/http_status.go` に置き、
応答本文を書く直前にステータスを決めて書く。**本文は1バイトも変えない。**

- 表は `errorCodeHTTPStatus`（414件）。400 / 401 / 403 / 404 / 409 / 429 / 500。
- 複数のエラーが載るときは `HTTPStatusForErrors` が
  `500 > 401 > 403 > 429 > 409 > 404 > 400` の順で最も重いものを選ぶ。
  サーバの障害を利用者側の誤りで覆い隠さないことを最優先にしている。
- ハンドラ側は `writeErrorStatus(w, response.Errors)` を
  `json.NewEncoder(w).Encode(response)` の**直前**に置く（89本すべて同一の1行）。
- ハンドラより手前で打ち切る経路（`auth_middleware.go` / `filter_local_only.go`）は
  `writeGkillErrorResponse` を通し、**ステータスと JSON 本文の両方**を書く。
- `ERR000002` は認証経路専用にし、操作対象が無い場合は
  新設の `TargetAccountNotFoundError`(ERR000413) を使う。

ドリフト防止として3本のテストを置く。

- `message/http_status_test.go` … `error_codes.go` をソース走査し、未分類のコードを落とす。
  ステータスごとの件数も固定するので「とりあえず全部500」も落ちる。
- `gkill_server_api/response_status_guard_test.go` … 全 `handle_*.go` を走査し、
  エンコード行の直前に `writeErrorStatus` があることを確認する。
- `gkill_server_api/response_status_test.go` … httptest で実際に 400/401/403/409/500 を確認し、
  **成功時の本文が変わっていないこと**も確認する。

## Rejected alternatives

- **ハンドラごとに手で `WriteHeader` を書く** — 574箇所。エラーの追加・コピペのたびに
  書き忘れが増え、しかも書き忘れは 200 に戻るだけなので気付けない。
  「同じ失敗が経路ごとに違うステータス」という今の状態がまさにその帰結だった。

- **エラーコードの命名規則からステータスを推論する** — **導けない。**
  `error_codes.go` の語彙には `Forbidden` / `Unauthorized` / `Denied` / `Permission` が
  **1件も無く**、`Invalid*` が 400（リクエスト不正）と 500（応答エンコード失敗）に跨り、
  `NotFound*` が 401（`AccountSessionNotFoundError`）と 404 に跨る。
  `NotFoundTLSCertFileError` に至ってはサーバの設定不備なので 500 が正しい。
  推論にすると、こういう例外が**静かに**間違ったステータスで返り続ける。

- **応答をバッファしてミドルウェアでステータスを決める** — `/api/get_kyous` は
  数十万件・100MB級の応答を返す（`gzipMiddleware` の doc コメント）。
  全部バッファしてから書くのは現実的でない。書き込み点がハンドラの `defer` に
  1箇所しか無いので、そこへ1行入れれば同じ効果が得られる。

- **移行期間中は旧挙動（常に200）へ戻せるスイッチを置く** — 消費者
  （Web クライアント・MCP・Wear OS・CLI・衛星リポジトリ）はすべて同じ管理下にあり、
  同時に直せる。スイッチを置くと契約が2つになり、どちらの経路もテストし続けることになる。

- **業務エラー（重複・見つからない）は 200 のままにし、認証と内部エラーだけ直す** —
  変えるエラーコードが約280件から約270件になるだけで、境界の説明が増えるぶん高くつく。
  「重複追加は 409」は HTTP の意味そのままで、説明が要らない。

## Consequences

- **ボディを変えていないので Web クライアントは1行も触っていない。**
  `gkill-api.ts` は `res.ok` / `res.status` を1箇所も見ておらず、
  `gkill-api.test.ts` が `status: 500` / `status: 403` でも JSON をパースすることを
  積極的にテストしている。
- **本文をステータスより先に読む消費者**は直した ——
  MCP（`gkill-client.mjs` の `post()`。直さないと `callApi` の自動再ログインに到達せず、
  長寿命プロセスなのでセッション期限以降ずっと復旧できなくなる）、
  `add_tag` CLI（当時の名前は auto_tag）、Wear OS companion、E2E ヘルパ。
  衛星リポジトリ（`gkill_autolog` / `gkill_autocomplete`）も同じ理由で直した。
- Service Worker は `service-worker-utils.ts` が既に `!response.ok` を見ているので、
  **非200がキャッシュに焼き付くことはない**（むしろ本文のパースを1回省ける）。
- `/api/*` で panic したとき、以前は **200 + 復号すると空の本文**が返っていた。
  `gzipMiddleware` の `defer gzipWriter.Close()` が panic の巻き戻しで先に走って
  暗黙200を確定させ、外側 `recoverMiddleware` の 500 が捨てられていたため。
  `recoverMiddleware` を最内層にも登録して直した（`serve.go`）。
- `Invalid*ResponseDataError`（応答のエンコード失敗）に割り当てた 500 は**実際には効かない**。
  そのコードが積まれるのはエンコード中で、ステータス行はもう送出済みだから。
  表には網羅性のために入れてあり、`http_status.go` の doc コメントに理由を書いてある。

## Evidence

実測（2026-08-23〜24、このリポジトリのソース走査と再現テスト）。

| 測ったもの | 値 |
|---|---|
| `gkill_server_api` の非テスト `.go` | 110ファイル |
| ルータ登録 | 91ルート（`wrapNoAuth` 16 / `wrapAuth` 19 / `wrapAuthRepos` 58） |
| `response.Errors = append(...)` のサイト | 574箇所 |
| そのうちステータスを変えていたもの | **0** |
| 改修前の `WriteHeader` 呼び出し（テスト除く） | 31（うち実書き込み29）。JSON ハンドラは `handle_urlog_bookmarklet_address.go` の1本のみ |
| 同一の `defer` + `Encode` 形のハンドラ | 89本（1行挿入で全数に効いた） |
| エラーコード | 414件（改修前411 + 新設3） |
| うちどこからも参照されていないもの | 64件（分類だけして残置） |
| `AccountNotFoundError`(ERR000002) の使用箇所 | 11箇所。うち**8箇所が「操作対象アカウント不在」**で、クライアントの `check_auth` が管理者をログアウトさせていた |

**panic 時の応答（再現テスト）**: `Accept-Encoding: gzip` で `/api/*` の panic するルートを叩くと、
改修前は **`status=200` / `Content-Encoding: gzip` / 復号すると本文が空**。
`gzipMiddleware` の `defer gzipWriter.Close()` が panic の巻き戻しで先に走って
空の gzip ストリームを書き、暗黙 200 を確定させていた。
外側 `recoverMiddleware` が書く 500 と JSON はその後ろに生バイトで積まれ、
gzip として読むと最初の空メンバで EOF になるため丸ごと失われていた。

**消費者の実測**: `src/client/classes/api/gkill-api.ts` の `res.ok` / `res.status` は **0件**
（84本の API がすべて無条件に `res.json()` してから `error_code` で分岐）。
よってボディを変えなければクライアントは無改修で通る。
一方で**本文より先にステータスで打ち切る**実装は
MCP（`gkill-client.mjs:108`）、`add_tag` CLI（当時の名前は auto_tag。現 `add_tag.go` の `post`）、
Wear OS companion（`GkillApiClient.kt` の7箇所）、
衛星の `gkill_autolog` / `gkill_autocomplete` にあり、いずれも直した。

## Related tests

- `src/server/gkill/api/message/http_status_test.go`
- `src/server/gkill/api/gkill_server_api/response_status_test.go`
- `src/server/gkill/api/gkill_server_api/response_status_guard_test.go`
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go`（`TestAuthMiddleware_RejectsInvalidSession`）
- `src/server/gkill/api/safefetch/safefetch_test.go`
- `src/mcp/__tests__/client.test.mjs` / `src/mcp/__tests__/write-client.test.mjs`
