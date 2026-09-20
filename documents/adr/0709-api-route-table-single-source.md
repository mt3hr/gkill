# ADR-0709: HTTP API のルート表は Go 側の1つの表を正本にし、OpenAPI からの生成は採らない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-14 |
| Sources | `src/server/gkill/api/gkill_server_api/gkill_server_api_address.go` / `.claude/skills/gkill-go-backend/SKILL.md`「HTTP API のルート表（2026-09 導入）」節 |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/gkill_server_api/gkill_server_api_address.go` |

## Context

外部レビュー（2026-09-13）が「API 契約に歴史のゴミが残っている」と指摘した。
`/api/get_kftl_template` と `/api/get_gkill_info` はアドレス定義だけがあってハンドラが登録されておらず、
実行すると 404。Web クライアント側にも同じ名前の残骸がある。根因は
「API 定義・route・handler・client が同じ正本から作られていない」ことで、
**OpenAPI / JSON Schema を正本にして server route/type → TS client を生成する**方向が提案された。

実態を数えると、同じ表が **4 箇所に手書き**されていた。

| 場所 | 中身 | 状態 |
|---|---|---|
| `gkill_server_api_address.go` | `XxxAddress` / `XxxMethod` フィールド 92 組 | 未登録の 2 組が残っていた |
| `serve.go` | `router.HandleFunc(...)` 90 行 | 正 |
| `gkill_server_api_test.go` の `setupTestRouter` | 76 行の**部分コピー** | 本番と違うラッパーが 4 本（下記） |
| `gkill-api.ts` | `xxx_address` 89 / `xxx_method` 95 フィールド | 未登録の 2 組＋相方の無い `_method` 6 個 |

テストハーネスの部分コピーが本番とずれていた 4 本は、`upload_files` / `upload_gpslog_files` が
本番の `wrapNoAuthCapped`（上限つき）ではなく素の `wrapNoAuth`、`get_kyous_mcp` / `get_rep_infos_mcp` が
本番の `wrapNoAuthCapped` ではなく `wrapAuthRepos`。つまり**統合テストは本番の認証ラッパーを一度も通っていなかった**。

既存の防御は 2 つあったが、どちらも通り抜けていた。

- `gkill-api.test.ts` は TS の `_address` と Go の `XxxAddress = "..."` を突き合わせるが、
  **両側に同じゴミが揃っていた**ので緑だった。
- `verify_docs` は「92 件定義・90 件登録」を**別々の数として資料に書かせ**、二重の数が一致しているかを
  検査していた。ドリフトの存在そのものを機械検査で制度化していた形になる。

さらに `message/error_codes.go` には、これら 2 本と、Web クライアントの孤児 `_method` が指す
かつてのエンドポイント（`get_git_commit_logs` / `update_tag_struct` / `update_rep_struct` /
`update_device_struct` / `update_rep_type_struct` / `update_kftl_template`）と、`get_*_struct` 4 本の
コードが計 37 件（メッセージコード 7 件）、どこからも参照されずに残っていた。

## Decision

**Go 側の1つの表を正本にする。** `gkill_server_api_address.go` の `apiRoutes()` が
「パス・HTTP メソッド・認証区分・無認証ボディ上限・ハンドラ」を 1 行 1 ルートで持ち、
本番の `serve.go` とテストハーネスの `setupTestRouter` は同じ `registerAPIRoutes` でその表を登録する。
「定義があるのに登録が無い」は構造的に起きない。

**Web クライアントは生成せず、表との突き合わせで守る。** `gkill-api.test.ts` が表をソース走査し、
パスの集合・HTTP メソッド・`_address`↔`_method` の対・`gkill_fetch` の配線を検査する。

存在しないエンドポイントの残骸（Go の定義 2 組、TS のフィールド 10 個、エラーコード 37 件、
メッセージコード 7 件）はすべて削除し、資料の「92 件定義・90 件登録」という二重の数は
ルート表の行数 1 つに戻す。

## Rejected alternatives

- **OpenAPI / JSON Schema を正本にして Go のルート・型と TS クライアントを生成する（レビューの提案）** —
  gkill の TS クライアント（`gkill-api.ts`、約 3,400 行）は薄い転送層ではない。
  `get_kyous` の応答を 5,000 件ずつ `Kyou` クラスへ実体化して `AbortController` で中断できるようにする、
  `add_tag` / `update_tag` の後に付随データキャッシュを捨てる、`get_server_configs` の生 JSON を
  `ServerConfig` に詰め直す、`check_auth` でログイン画面へ戻す —— といった振る舞いがメソッドごとに
  手書きされている。生成物が担えるのは**パスとメソッドの 2 文字列**だけで、それ以外は生成された
  薄いクライアントの上に今の 3,400 行を載せ替える改修になる。Go 側も `req_res` 186 型が `find.FindQuery` や
  `reps.Kyou` などドメイン型をそのまま埋め込んでおり、スキーマから生成した型へ置き換えるには変換層が要る。
  MCP（`src/server/gkill/mcp`）・Wear OS・CLI（`add_tag` / `update_cache`）はパスをリテラルで持っていて、生成しても
  これらの突き合わせテストは残る。今回観測されたドリフトは「パス・メソッド・登録・認証区分」の層で、
  そこは表 1 つで閉じる。将来 gkill 外の消費者が現れて仕様書が要るなら、
  **Go の表 → OpenAPI の書き出し**（逆向き）が安い。

- **構造は変えず、ソース走査のテストだけ足す** — 「アドレス定義があるのに `serve.go` に無い」は
  落とせるが、テストハーネスの部分コピーは残る。本番と違うラッパーで登録された 4 本は
  そのままで、統合テストが本番の認証経路を通らない状態が続く。

- **Go の表から TS のルート定数を生成する** — 2 文字列 × 86 本のためにビルド工程を 1 つ増やし、
  生成物の鮮度検査も要る。突き合わせテストは同じ保証をビルド工程なしで与える。
  `gkill-api.ts` の `xxx_address` / `xxx_method` は残るが、対応関係はテストが固定する。

- **`HandleXxx` のメソッド名から反射でルートを組む** — パスが名前から導けない
  （`AddAccount` → `/api/add_user`、`GetTagsByTargetID` → `/api/get_tags_by_id`、
  `AddNotification` → `/api/add_gkill_notification`）。認証区分とボディ上限の選択も
  名前には無く、規約で隠すと退行が黙って通る。

- **未参照のエラーコードは残す（番号の連続性のため）** — 存在しないエンドポイントのコードが
  「エラーコード体系」の資料に載り続けるのがこのレビューの指摘そのもの。欠番は既に 4 つあり
  （`ERR000243` / `ERR000387` 〜 `ERR000389`）、連番であることに依存する箇所は無い
  （`message_test.go` は形式と一意性だけを見る）。実在するエンドポイントの未参照コード
  （約 27 件）は「将来のエラー分岐」でありうるので今回は触らない。

## Consequences

- **エンドポイントを足す手順は「ハンドラ + 表の 1 行 + golden の 1 行 + TS の対」。**
  表へ足し忘れるとハンドラがあっても 404 だが、`TestAPIRoutes_EveryHandlerIsRouted` が
  `HandleXxx` の反射列挙と表を双方向に突き合わせるので `go test` で落ちる。
- **認証区分の変更は 2 箇所を意図して直す作業になる。** 表だけ変えると
  `TestAPIRoutes_AuthKindGolden` が落ちる。`wrapAuthRepos → wrapNoAuth` のような退行が
  テストを 1 本も落とさずに本番へ届くことを防ぐための、意図的な二重化。
- **doc コメントの `// POST /api/xxx（wrapXxx）` 行は表と一致していなければならない**
  （`TestAPIRoutes_DocCommentMatchesTable`）。資料の網羅率検査が頼りにしている唯一の
  「ハンドラ → パス・認証区分」の記述なので、ずれると読む人だけが騙される。
- **テストハーネスが本番の認証ラッパーを通るようになった。** アップロード 2 本は上限つき、
  MCP 2 本はハンドラ内の自己認証経路で検査される。`TestAuthMiddleware_RejectsInvalidSession` は
  表から機械的に対象を組むので、表に足した認証つき経路は自動で検査に入る。
- **表の行は 1 行で書く。** `verify_docs` と `gkill-api.test.ts` が
  `{Path: "...", Method: "...", Auth: ..., Body: ..., Handler: g.HandleXxx},` を正規表現で読む。
  複数行に割ると両方が 0 件になり、`verify_docs` はそこで落ちるが TS 側は「0 件ガード」で落ちる。
- **Web 以外の消費者（MCP / Wear OS / CLI）のリテラルパスは今回も手書きのまま。**
  同じ突き合わせテストをそれぞれに足すのが次の改善候補。

## Evidence

実測（2026-09-14、このリポジトリのソース走査）。

| 測ったもの | 値 |
|---|---|
| ルート表の手書き箇所 | 4（Go 定義 / `serve.go` / テストハーネス / `gkill-api.ts`） |
| 未登録のアドレス定義 | 2（`get_kftl_template` / `get_gkill_info`。Go と TS の両方） |
| テストハーネスの登録 | 76 / 90 本。うち本番と違うラッパー 4 本 |
| TS の相方が無い `_method` | 6（`get_git_commit_logs` / `update_tag_struct` / `update_rep_struct` / `update_device_struct` / `update_rep_type_struct` / `update_kftl_template`）＋名前のずれ 1 組（`update_server_configs_address` ↔ `update_server_config_method`） |
| 存在しないエンドポイントのエラーコード | 37 件（414 → 377）。メッセージコード 7 件（90 → 83） |
| 表の内訳 | 90 ルート = `authNone` 13（うち上限つき 12）/ `authSession` 19 / `authSessionRepos` 58 |
| doc コメント `// POST /api/xxx（wrapXxx）` | 90 / 90 が同一書式（表との突き合わせに使える） |

変異確認: 表から 1 行消す → 3 テスト（双方向突き合わせ・doc コメント・golden）が落ちる。
認証区分を 1 本変える → doc コメントと golden の 2 テストが落ちる。
TS に相方の無い `_method` を足す／メソッドを GET に変える → それぞれ 1 テストが落ちる。

## Related tests

- `src/server/gkill/api/gkill_server_api/api_routes_test.go`
- `src/server/gkill/api/gkill_server_api/auth_middleware_capped_test.go`（`TestAPIRoutesNoAuthBodyRoutesAreCapped`）
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go`（`TestAuthMiddleware_RejectsInvalidSession`）
- `src/client/__tests__/unit/api/gkill-api.test.ts`（`endpoint address parity with Go`）
- `src/server/gkill/api/message/http_status_test.go`
