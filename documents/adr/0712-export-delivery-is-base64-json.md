# ADR-0712: 出力した HTML は JSON に載せて返し、ファイルそのものを body で返さない

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-22 |
| Sources | `src/client/classes/api/gkill-api.ts` の `gkill_fetch` / `src/server/gkill/api/req_res/download_skill_response.go` / `src/client/classes/file-base64.ts` / [ADR-0415](0415-exported-view-is-a-separate-flag.md)（出力モードと通信の1点止め） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/api/req_res/export_kyou_html_response.go` の doc コメント、`src/client/classes/api/gkill-api.ts` の `export_kyou_html` の doc コメント |

## Context

「HTML で出力」は、HTML ファイル1枚（single）か、HTML と同名フォルダをまとめた ZIP（sidecar）を
利用者へ渡す。素直に考えれば `Content-Disposition: attachment` で body にファイルを載せたい。

しかし gkill ではそれができない。

1. **`Content-Disposition` は `src/server` 全体に1件も存在しない。** 採った前例が無い
2. **`gkill_fetch` は JSON 以外の応答を `bad_response`（ERR900100）の合成応答に変える**
   （`gkill-api-bad-response.test.ts` が固定している）。ファイルを直接返すと、
   クライアントの共通層がそれを**エラーとして扱う**
3. 唯一の ZIP 配布（スキル）が同じ理由で base64 を選んでいる。
   `req_res/download_skill_response.go` の doc コメントが「クライアントは JSON 以外の応答を受け付けないため」と書いている

## Decision

**出力の結果は JSON のフィールドに載せる。** single は `html`（文字列）、sidecar は
`zip_base64`（ZIP を base64 にしたもの）。受け口の `base64_to_blob` は
`src/client/classes/file-base64.ts` に既にある。

両方が同じ応答型になるので、`defer` の `writeErrorStatus` → JSON という
全ハンドラ共通の形がそのまま成立する。

## Rejected alternatives

- **`Content-Disposition` で body にファイルを載せる** — `gkill_fetch` が JSON 以外を `bad_response` に変える。この1点だけのために共通層へ分岐を入れると、以後どの応答が JSON なのか読み手に分からなくなる。
- **出力だけ `fetch` を直に呼ぶ（`gkill_fetch` を通さない）** — セッション・再認証・エラーの合成・中断の扱いが共通層にあり、1本だけ別経路にすると、ログインが切れたときの振る舞いがこの API だけ違う。
- **`showSaveFilePicker()` でブラウザに保存させる** — `file-system-access.d.ts` は `showSaveFilePicker` だけを宣言していて `showDirectoryPicker` の型が無く、ZIP を受け取ってから JS で展開する新規依存も要る。対応ブラウザも限られる。
- **出力した HTML をサーバに置いて URL を返す** — 置き場の寿命・掃除・認可（誰が取れるか）がまるごと増える。出力は「渡して終わり」のものなので、サーバに残す理由が無い。

## Consequences

- **base64 は素のバイト数の 4/3 になる。** サーバのメモリは ZIP 実体のおよそ 2.3 倍（実体 + base64）を要する。サイズ予算は展開後の合計ではなく **ZIP 実体に対して**掛ける。
- 応答が大きくなるので、メディアの埋め込みは出力時に選べるようにしてある（既定はサムネイルのみ）。
- クライアントは `html` か `zip_base64` のどちらか一方だけを見る。`output_format` が応答にも入っているので、要求と応答が食い違ったときに気づける。

## Evidence

実測なし —— `src/server` に `Content-Disposition` が1件も無いこと、`gkill_fetch` が
JSON 以外を `bad_response` に変えること（`gkill-api-bad-response.test.ts`）、
スキルの ZIP 配布が同じ理由で base64 を選んでいることからの判断。

## Related tests

- `src/client/__tests__/unit/api/gkill-api-bad-response.test.ts` — JSON 以外の応答が `bad_response` の合成応答になること（この ADR の前提）
- `src/server/gkill/api/gkill_server_api/handle_export_kyou_html_test.go` — 出力が JSON の `html` に入って返ること、弾いたときは `html` が空のままであること
