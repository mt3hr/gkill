# ADR-0606: IDF ファイルの経路は優劣ではなく用途で分かれる — 到達できないツールは消し、画像を見せる base64 は残す

| | |
|---|---|
| Status | Accepted |
| Date | 2026-08-24 |
| Sources | `2e478be5` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/server_base.go`（`buildToolResult`）/ `src/server/gkill/mcp/payload.go`（`applyFileLinks`）/ `src/server/gkill/mcp/http_transport.go`（`GET /files/{token}`） |

## Context

MCP から IDF（ファイル・画像・動画）の実体へ届く経路は4つあった。

| 経路 | 効く条件 |
|---|---|
| `file_url` / `file_url_full` | HTTP 接続。`gkill_get_kyous` の IDF ペイロードへ自動注入 |
| `file_path` | stdio 接続 **かつ** gkill への接続元が loopback |
| `gkill_get_idf_file`（base64） | どこでも。`GKILL_MCP_MAX_FILE_BYTES`（既定8MB）で打ち切り |
| `gkill_get_idf_file_path` | rep 名とファイル名から絶対パスを引く |

最後の1つが**固有の到達性を1つも持っていなかった**。
動く条件は `isLocalTransport`（MCP 側）と `isLocalRequest`（Go 側）の AND だが、
`gkill_get_kyous` のペイロードに `file_path` が載る条件も同じ2つの AND
（`handle_get_kyous_mcp.go` が同じ `isLocalRequest` を見る）。
片方だけ真になる組み合わせが存在しないので、**`file_path` の真部分集合**でしかない。

さらに悪いことに、2本の IDF ツールは互いを指していた。
`get_idf_file` はサイズ超過時に「`get_idf_file_path` を使え」と案内するが、
HTTP クライアントではそのツールが必ず失敗する。行き止まりになる。

そして残る3経路の説明が、**全域で優先順位を取り違えていた**。
ツール説明は "PREFER PATH OR URL INSTEAD WHEN AVAILABLE … Use this tool only as a fallback"、
検索スキーマは "prefer in this order: (1) file_path (2) file_url (3) gkill_get_idf_file — base64 fallback"、
README は「AIはこのURLを **Bearer無しでGET** すればバイトを取得できる」、
スキルは "3 paths, preferred order" と書いていた。
どれも `file_url` を base64 の上位互換として扱っている。

## Decision

`gkill_get_idf_file_path`（MCP ツール）と `/api/get_idf_file_path`（Go エンドポイント）を削除する。

**残る3経路は優劣ではなく用途で分かれる**ものとして説明を書き直す。優先順位として書かない。

| クライアント | 画像を**見る**（＝生成の参照に渡せる） | 実体を人間へ渡す |
|---|---|---|
| stdio | `file_path` を直接読む | `file_path` |
| HTTP | **`gkill_get_idf_file` だけ** | `file_url` / `file_url_full` |

`type:"image"` の MCP content ブロックを組み立てるのは `buildToolResult` の1箇所だけで、
その入力は `gkill_get_idf_file` が返す `file_content_base64` に限られる。
このブロックが**画像を会話の中に置く**ものなので、HTTP 接続の AI にとって
`gkill_get_idf_file` は「フォールバック」ではなく**画像を見る唯一の手段**。

`file_url` の価値は消えない。人間へ渡すリンクとして、また 8MB を超える実体の唯一の受け渡し口として残る。
ただし**AI がバイトを得る経路としては書かない**。MCP はツール出力の URL を代わりに取りに行かないし、
クライアントが自前の機能で取れたとしても、得られるのは会話の外側のバイトであって見られる画像にはならない。

`isLocalRequest`（`filter_local_only.go`）は残す。`filterLocalOnly` と
`handle_get_kyous_mcp.go` の `file_path` 可否がこれで決まるため。
判定を固定する8ケースのテストは `filter_local_only_test.go` へ移した。

## Rejected alternatives

- **`gkill_get_idf_file_path` を残して説明だけ直す** — 説明を直しても HTTP クライアントからは
  依然として1回も成功しない。ツール一覧は AI が毎回読む固定コストで、
  到達できないものが載っていること自体が選択を誤らせる
- **`gkill_get_idf_file`（base64）も同時に消す** — **実測で否決**（Evidence 参照）。
  ログ上 `/files/` は0件で「URL 経路が本命」と読めたが、そのログは `http_request` 記録が
  `a510fbc2`（2026-08-19）で入った直後の2日ぶんしかなく、中身はほぼ監査トラフィックだった。
  利用者に実際の操作を1回してもらったところ、増えたのは `gkill_get_idf_file` の3回で
  `/files/` は0件のまま。**消していたら画像生成の参照が丸ごと壊れていた。**
  0件を「使われていない」と読まなかったのが分かれ目
- **`file_url` を消して base64 に一本化する** — 8MB を超える実体の受け渡し口が無くなり、
  人間にリンクを渡す用途も潰れる。用途が違うのでどちらも要る
- **`/api/get_idf_file_path`（Go）だけ残す** — 呼び出し元は削除した MCP ツールだけだった。
  Vue クライアントも Wear OS も使っていない（`gkill-api.test.ts` の
  `NON_WEB_CLIENT_ADDRESSES` に「MCPサーバ専用」として明示的に載っていた）
- **`/files/{repName}/{filePath}`（Go）も整理する** — こちらは Vue 本体の画像・動画・Markdown 表示の
  中核で、共有ページの認可もここ。MCP の `file_url` 配信も内部でこれを叩く。触ってはいけない

## Consequences

- **`gkill_get_idf_file` は消せない。** HTTP 接続の AI が画像を見る唯一の経路であり、
  8MB 上限は事実上「見られる画像の上限」でもある。`GKILL_MCP_MAX_FILE_BYTES` を下げると
  画像生成の参照に使える絵が静かに減る
- **説明文を「URL を優先しろ」へ書き戻さないこと。** 実使用の ChatGPT はこの誤誘導を
  無視して正しく base64 を呼んでいたが、素直に従う AI がいれば画像は届かない
- `file_url` のトークンは1時間で失効する（`GKILL_MCP_FILE_LINK_TTL_MS`）。
  切れたら同じ検索をやり直せば新しい URL が発行される
- `file_url` は Bearer 不要で配信する（画像取得は認証ヘッダを付けられないため、
  トークンが唯一の防御線）。この設計は変えていない
- **エラーコードが3つ消える**（`ERR000387` / `ERR000388` / `ERR000389`）。
  採番に穴が空くが連番性を要求する検査は無く、`errCodeMax` は変わらない。
  消すときは `http_status.go` の表・`http_status_test.go` の分布期待値
  （400/403/500 の3つ）・`verify_docs` の件数を同時に直さないと落ちる
- ツール数は Read 11→10 / ReadWrite 32→31。Write は元から非搭載で 26 のまま。
  `resources/manual_src/` の7言語のうち `verify_docs` が見るのは ja だけなので、
  残り6言語のハードコード値は手で直す（`checkManualParity` は h2/h3/table の個数しか比べない）
- **削除しても `payload.go` の関数は1つも減らない。** `file_url` 経路が使い続けるため

## Evidence

2026-08-24 の実測。

**到達不能ツールの削除（`gkill_get_idf_file_path`）:**

- `http_transport.go` は `isLocalTransport = false` を**無条件**で立てる。
  つまり claude.ai コネクタや ChatGPT のような HTTP クライアントからは、
  このツールは gkill へ問い合わせる前に必ず throw していた
- `$GKILL_HOME/logs/gkill_mcp_read_access.log`（812行、2026-08-22〜24）で
  read サーバの `server_start` は4回とも `transport: http`。
  `gkill_get_kyous` は307回呼ばれているのにこのツールは1回だけ
- Go 側の全参照を走査して、削除後に残る `GetIDFFilePath` / `get_idf_file_path` は0件
- `b68d6bb7` の跡地が示すとおり、このツールは readwrite サーバ（stdio）で
  廃止済み `client.callRead` を呼び続けて**長期間 TypeError で静かに壊れていたのに誰も気付かなかった**

**base64 を消さなかった判断の根拠（1操作で決着させた実測）:**

`gkill_get_idf_file` を消してよいかは既存ログでは決められなかった
（上記のとおり記録が2日ぶんで、ほぼ監査トラフィック）。
利用者に ChatGPT でいつもの操作（キーワード検索 → ヒットしたイラストを参照して新規イラストを生成）を
1回してもらい、増分を数えた。

| 指標 | 操作前 | 操作後 |
|---|---|---|
| `"path":"/files/` | 0 | **0** |
| `gkill_get_idf_file` | 0 | **3**（`response_bytes` 3.3MB / 2.8MB / 2.4MB） |
| `gkill_get_kyous` | 307 | 308（1回、21.8秒） |

クライアント側が申告した経路もログと一致した。検索が145件ヒット →
直近30件から `data_type: idf` を3件選択 → `gkill_get_idf_file` × 3 で
PNG を image ブロックとして取得 → **その3枚を会話内の参照画像として画像生成ツールへ渡す**。
`file_url` は「使う方法もあったが今回は使っていない」との申告。
**当初の想定（URL が本命、base64 はフォールバック）は逆だった。**

## Related tests

- `src/server/gkill/mcp/tool_handlers_test.go`
- `src/server/gkill/mcp/server_test.go`
- `src/server/gkill/mcp/file_link_test.go`
- `src/server/gkill/api/gkill_server_api/filter_local_only_test.go`
- `src/client/__tests__/unit/api/gkill-api.test.ts`
