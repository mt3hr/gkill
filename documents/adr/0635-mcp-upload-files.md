# ADR-0635: MCP からのファイルのアップロードは画面の「ファイル」タブの写しで、file_path は stdio だけ受ける

| | |
|---|---|
| Status | Accepted |
| Date | 2026-10-18 |
| Sources | 利用者の要求「gkill_write の MCP に gkill_upload_file のような名前のツールを足してほしい。オプションは画面と同じような感じにしたい」。壁打ちで決めたこと: 名前は `gkill_upload_files`、今回は画面の「ファイル」タブだけ（GPS ログのタブは必要になったら別ツール） |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/mcp/upload_files.go` / `src/server/gkill/mcp/write_tools.go` / `src/server/gkill/api/gkill_server_api/handle_upload_files.go` / `src/server/gkill/api/req_res/upload_files_request.go` |

## Context

MCP はファイル（idf の記録）を読めるが（`gkill_get_idf_file`。経路は ADR-0606）、置く手段が無かった。
画面には「アップロード」（`upload-file-view.vue`）があり、「ファイル」タブで保存先の file rep（rep_type `directory`）・
名前が重なったときの扱い（名前を変える / 上書き）を選んでファイルを落とすと `/api/upload_files` が base64 の本文を受けて
ディレクトリへ書き、idf の記録として登録する。関連時刻はブラウザの `File.lastModified` で、あとからダイアログで直せる。

MCP は起動中の gkill_server の HTTP クライアントで、データのファイルを直接触らない（ADR-0631）。
クライアントには stdio（同じ端末）と HTTP（リモート。OAuth）の2種類があり、読み取りの経路はこの種別で分けている（ADR-0606）。

## Decision

- ツール名は `gkill_upload_files`。write / readwrite に載せる（`WriteTools` へ1本）。叩くのは画面と同じ `/api/upload_files`。
- 引数は画面の「ファイル」タブの写し: `files[]`（1〜100件）・`target_rep_name`（省くと `/api/get_rep_infos_mcp` の書き込み先の
  file rep。画面が最初に選んでいるもの）・`conflict_behavior`（`rename` 既定 / `override`）・各ファイルの `related_time`。
  `merge` は GPS ログ用で、ファイルでは `override` と同じ動きになるので出さない（画面も出していない）。
- 各ファイルは `file_name` + `data_base64` か `file_path` のちょうど一方。**`file_path` は stdio のときだけ受ける。**
  stdio では MCP サーバが自分でファイルを読み、`file_name` の既定は基底名、関連時刻の既定は更新日時（画面の `lastModified` と同じ）。
  `data_base64` の関連時刻の既定は現在時刻。
- 本体の `UploadFilesRequest` に `create_app` を足し（空なら従来どおり `"gkill"`）、MCP は自分のアプリ名
  （`gkill_mcp_write` / `gkill_mcp_readwrite`）を送る。
- 本体の `HandleUploadFiles` がキャッシュ rep へ書き戻す記録に `ContentPath` を入れる。キャッシュの検索は `CONTENT_PATH` が空の行を
  「繋がっていない rep の ID」として読み飛ばすので、空のままだと上げたファイルが次の UpdateCache（再起動を含む）まで検索に出なかった。
  画面からのアップロードも同じだった（画面はアップロード直後の一覧を応答の記録で出すので気付きにくい）。dev 環境で MCP から上げた直後の
  `gkill_get_kyous`（`query.ids`）が 0 件になったことで見つかった。キャッシュ OFF では再現しない。
- 入口で断るもの: base64 として読めない値（Strict。data URL の前置きと改行は剥がす）、`/` `\` を含む `file_name`、
  1回の呼び出しの中での同名（大小無視）、gkill の本文上限（1GB）を超える見積もり。

## Rejected alternatives

- **`gkill_upload_file`（単数・1回1ファイル）** — 画面は複数のファイルを一度に落とせ、API も配列で受ける。単数にすると
  AI がファイル数だけ往復する。
- **`gkill_add_idf_file`（読み取りの `gkill_get_idf_file` と対の名前）** — `add_*` は新しい記録を足すだけで既存を上書きしないが、
  このツールは `override` でディスク上のファイルを置き換えうる。画面と API の語（アップロード）に揃えたほうが、利用者の言い方
  「この画像を上げて」からも引きやすい。
- **HTTP でも `file_path` を受ける** — HTTP のクライアントはリモートなので、パスはクライアントの手元ではなく MCP サーバの手元の
  ファイルを指す。受けると、リモートの AI が MCP サーバの端末の任意のファイルを gkill へ取り込み、`gkill_get_idf_file` で
  読み出せる。読み取り側がローカルパスを HTTP のクライアントへ見せないのと同じ理由で、種別をゲートにする。
- **作成アプリ名を `"gkill"` のままにする** — MCP で上げた記録が画面からの分と区別できず、`create_apps:["gkill_mcp_readwrite"]`
  で見つからない（メモ帳経由の書き込みと同じ穴。help topic で注意書きしている）。本体の要求に欄を1つ足すだけで、Web は送らないので
  挙動は変わらない。
- **GPS ログのタブも同じツールにまとめる（種別の引数で切り替える）** — 保存先の rep の種類も、衝突の選択肢（`merge` が意味を持つ）も、
  登録する記録の種類も違う。必要になったら `/api/upload_gpslog_files` を呼ぶ別ツールにする。
- **アップロード直後にタグを付ける引数** — 画面も付けない。応答の id を `gkill_add_tag` へ渡せば足り、タグの規則を2か所に持たない。

## Consequences

- **トランザクションではない**（ADR-0217。画面と同じ）。1件でも失敗すると gkill はエラーだけ返して何も登録しないが、書けたファイルは
  ディレクトリに残り、次の索引更新で idf の記録として現れる。`rename` のまま盲目的にやり直すと2つ目の複製ができるので、help topic `idf` で
  先に確かめるよう案内する。
- 同じ呼び出しの中の同名を断るのは、gkill が衝突の判定をファイルごとに書き込み前に行い、同じ呼び出しの2つが両方「まだ無い」と
  判定されて同じパスへ書かれるため。呼び出しを分ければ `conflict_behavior` が効く。
- HTTP では MCP の本文上限（`maxMcpBodyBytes`）がそのまま効き、超えると MCP の手前で 413 になる。大きなファイルは stdio の
  `file_path` か、利用者が rep のディレクトリへ置いて索引を更新する。
- ツールが1本増え、`schema_revision` が write / readwrite で変わるので、ChatGPT と Claude.ai のコネクタは接続し直しが要る。

## Evidence

- tools/list の予算（`tool_schema_budget.json`、`schema-budget` の実測）: write 63,602 → 65,728、readwrite 88,157 → 90,283（どちらも +2,126）。
  説明文は ADR-0622 の要約にとどめ、詳細は help topic `idf` に置いた。
- golden を再生成した差分は、既存ケースでは initialize（`serverInfo.version` の schema 印）・tools/list・status（`tool_count` と `schema_revision`）・
  help の目次 / `data_types` / `idf` だけで、ほかは追加した 9 ケース（ゴールデンの要求コーパス由来の件数）。
- 利用者の判断以外は実測なし — `file_path` のゲートは脅威モデル（リモートの AI が MCP サーバの手元のファイルを読む）からの判断。

## Related tests

- `src/server/gkill/mcp/upload_files_test.go`（既定の rep 解決、ERR000151 の包み直し、`create_app`、Strict base64、`file_path` は stdio だけ
  —— 手で組んだ文脈だけでなく `Server.HandleMessage` を通して、書き込み側の文脈にもトランスポートが渡ること、
  更新日時が関連時刻の既定になること、合計の上限、rep 名の列挙の打ち切り）
- `src/server/gkill/mcp/golden_test.go`（`upload files: *` / `help: idf` のケース）
- `src/server/gkill/mcp/schema_contract_test.go`（全プロパティ指定のスモーク。`files[0].x` の形のフィールドも落として呼び直す）
- `src/server/gkill/api/gkill_server_api/gkill_server_api_test.go` の `TestHandleUploadFiles_CreateApp`（`create_app` の有無と刻まれる値）と
  `TestHandleUploadFiles_UploadedEntryIsSearchableRightAway`（上げた直後に `/api/get_kyous` で引ける。キャッシュ ON / OFF の両方）
