---
name: gkill-mcp
description: "gkill の MCP サーバ（src/mcp/、read/write/readwrite の3種、stdio + HTTP OAuth 2.1）の約束。1リクエスト文脈は不変の requestContext を引数で流し server.current* に書かない、プラグイン本文は gkill_get_kyous の include_plugin_content でまとめて埋め込み同一プラグインへ並列に投げない、IDF ファイルの3経路（file_path / file_url / base64）とトランスポート種別ゲート、環境変数一覧を扱う。src/mcp/ を編集するとき、MCP ツールを追加・変更するとき必読。「他人のセッションに紐づく file_url が発行される」「ErrPluginBusy になる」の調査でも必読。"
---

# MCP サーバの不変条件

対象: `src/mcp/**`

**このファイルは全文が、実際に起きた事故の再発防止である。該当作業では飛ばさずに読むこと。**
多くは「例外もエラーも出さずに静かに壊れる」種類で、破っても目の前ではエラーにならない。

### MCP Server — `src/mcp/`

3つのMCPサーバーを提供。いずれもstdio (ローカル) / HTTP (OAuth 2.1認証付きリモート) の2モード対応。

| Server | Tools | stdio | HTTP | Port |
|---|---|---|---|---|
| Read | 10 (get_kyous, get_mi_board_list, get_all_tag_names, get_all_rep_names, get_gps_log, get_application_config, get_rep_infos, get_idf_file, get_kyou_history + plugin 1) | `npm run mcp:gkill-read` | `npm run mcp:gkill-read-http` | 8808 |
| Write | 27 (9 add + 1 submit_kftl + 1 delete + 1 restore + 9 update + 5 read convenience + plugin 1) | `npm run mcp:gkill-write` | `npm run mcp:gkill-write-http` | 8809 |
| ReadWrite | 31 (read 9 + write 21 + plugin 1) | `npm run mcp:gkill-readwrite` | `npm run mcp:gkill-readwrite-http` | 8810 |

**GPS専用プラグインの `rep_name` / `data_type` は検索値ではない。** manifest の必須項目なので値は入っているが、`emits_kyou:false` のプラグインは Kyou を1件も出さないので `query.reps` にも `data_types` にも一致しない。`get_plugin_list` は `emits_kyou` / `provides` をそのまま返す（`capabilities` のような3つ目の語彙を作らないこと。正本は manifest の語彙）。`get_rep_infos` の `plugins[]` は「渡せる値」の表なので `emits_kyou:false` は載せず、`attached_data_reps[]` の `data_kind:"gpslog"` にだけ残す（ADR-0607 / ADR-0608）。Go 側の未知値警告も、一致したときは汎用文ではなく「そのプラグインは Kyou を出さない。`get_gps_log` で読め」と名指しする。**`typed_index` は Kyou の索引**なので `provides` が `gpslog` だけのプラグインには付けない（付けると材料が無く `never_built`/0 で固定され「索引が壊れている」と誤読される）。GPS の取り込み状況は別枠の `gps_index`。

**プラグインの診断文はAIへ返さない。** `last_error`（プラグインプロセスの生stderr）と `typed_index.last_build_error` には、利用者の端末のディレクトリ構成がそのまま入る。プラグインは別リポジトリの成果物なので「実パスを書くな」を書き手側の約束にはできない（実際 uguisu のプラグインは、ホームが `systemprofile` へ化ける事故を診断するために意図してホームを書いている）。`handlePluginToolCall` が `plugins[]` からこの2つを落とし、非空だったときだけ `has_last_error` / `has_last_build_error` を立て、**実際に落としたときだけ** `warnings` へ1行足す（落としていないのに出すと常時ノイズになる。ADR-0609 と同じ判断）。**`has_*` をやめて中身を返す形へ戻さないこと** —— 「is_alive=true なのに0件」の診断（外部監査 D2）は「何か書かれている」ことだけで成立する。Go 側も出口で端末固有の情報を伏せるが（`message.RedactEnvironmentSpecific`）、あれは HTTP API を直に叩く経路のための最後の網であって、AI へ中身を渡してよい理由にはならない（[ADR-0707](../../../documents/adr/0707-redact-environment-specific-strings.md)）。

**`data_type` には2つの語彙がある。** 検索結果と `add_*` / `update_*` の応答が返すのは**射影名**（`mi_create` / `mi_check` / `mi_limit` / `mi_start` / `mi_end` / `mirekyou_*` / `timeis_start` / `timeis_end`）、delete / restore / history が受理するのは**エンティティ種別**（`ENTITY_TARGETS` のキー。`mi` / `timeis` …）。対応表は `lib/constants.mjs` の `PROJECTION_TO_ENTITY_DATA_TYPE` 1箇所で、3ツールの入口が `toEntityDataType` を通すので**応答をそのまま次のツールへ渡せる**（以前は `gkill_add_mi` の応答 `mi_create` を `gkill_delete_kyou` へ渡すと enum 違反で落ち、KFTL の `created[]` だけが通る三者三様だった）。**さらに `data_type` は `for_mi` の関数でもある** —— `for_mi` を立てないと Mi の5射影は代表1件へ潰れ（`find_filter.go` の `isMiData` gate）、`mi_check` がほぼ常に勝つので `mi_create` は素の検索に出てこない。`data_types` に射影名を渡して `for_mi` が無いときは Go 側が警告する。潰し込みを外す・`for_mi` を自動で立てる、はどちらも否決（[ADR-0610](../../../documents/adr/0610-mi-projection-depends-on-for-mi.md)）。

**接続先は `gkill_get_application_config` の `fields:["user_id","device"]` で分かる。** read / write / readwrite が別アカウントを向いていることがあり、AI からは区別できずに「同じAPIなのに件数が違う」「`query.ids` が壊れている」と誤診された。gkill は元から返しており、MCP の射影が捨てていただけ。**`GKILL_BASE_URL` / `GKILL_USER` の値そのものは出さない**（端末の情報。ADR-0707）。

**同じ処理を2形態で持たない。** 手順が同じで型ごとに違うのは「正規化関数」と「patch する欄」だけ、という所は表で持つ。`gkill_update_*` 9本は24行のブロックが9本並んでおり、取得先・更新先・応答キーの対応表（`ENTITY_TARGETS`）が既にあって `softDeleteOne` と `gkill_get_kyou_history` はそちらを使っている、という「表と直書きの併存」状態だった（2026-08-25 に `UPDATE_TARGETS` + `runUpdate` へ集約）。併存していた間、**「見つからない」の文言が3種類に割れていた** —— `entityNotFoundMessage`（`lib/payload.mjs`）が唯一の正本で、read / write / update の全部がここを通る。型を取り違えたのか ID が無いのかは型別エンドポイントの応答からは区別できないので、**区別できないことを言う**（「ID が存在しない」と断定しない）。

**書き込みに刻む user は、その要求を認証したセッションから決める。** `create_user` / `update_user` は
MCP が本文へ載せる自己申告値で、gkill 側は上書きしない（Web UI も Wear も同じ）。`sid`（どのアカウントの
DBへ書くか）と `userId`（レコードに刻む名前）は**出どころが別**なので、`|| this.client.userId`
（環境変数 `GKILL_USER`）へのフォールバックを無条件に残すと、**別アカウントのセッションへ書いた
レコードに手元の名前が焼かれる**。エラーは出ず、書き込みは成功する。`mcp-server-base.mjs` は
sid がある（＝認証済み）ときだけ userId を必須にし、sid を持たない stdio だけ環境変数を使う。
**特定のクライアント名をコードから探さないこと** —— 「あのクライアント専用の残骸では」という指摘は
毎回ここへ戻ってくるが、ハードコードは1箇所も無く、実体は接続に使ったアカウント名がそのまま
出ているだけ（[ADR-0615](../../../documents/adr/0615-write-user-comes-from-the-authenticated-session.md)）。

**更新は「変わる欄が1つも無い」なら書かずに断る。** `runUpdate` は9ツール共通なので、ガード1つで全部に効く。
追記型なので no-op でも版が1つ増え、あとから読む側には「何が変わったのか」が区別できない
（delete / restore の `already deleted` / `already active` と同じ理由。判定は `undefined` のみを未指定と
みなす —— `null` は `timeis.end_time` で「消す」の意思表示になる）。**`update_time` は `nextUpdateTime` を
通すこと** —— 素の `new Date()` だと1秒解像度の丸めで、同じ秒の中の2回目が最新版と見なされず黙って消える
（[ADR-0616](../../../documents/adr/0616-update-rejects-an-empty-patch.md)）。

**説明文が「これを呼べ」と名指しするツールは、そのサーバに載っていること。** read の履歴説明が
`gkill_restore_kyou` を、write の「見つからない」が `gkill_get_kyous` を案内していて、どちらもそのサーバには
無かった（AI は載っていないツールを探しにいく）。`tool-handlers.test.mjs` が read / readwrite の全 description と
`entityNotFoundMessage` を走査して検査する。**参照情報としての言及**（「この id は `gkill_update_text` へ
渡すためのもの」）だけが `CROSS_SERVER_TOOL_MENTIONS` の例外で、**案内を足したくなったら表ではなく
ツールの搭載側を直すこと**（接続先確認は書き込み前こそ要るので、`gkill_get_application_config` は
write 専用サーバにも載せた）。write 専用サーバは read を数本に絞る設計なので、共有された説明文が
検索の口に触れるのは避けられず、そちらは検査対象から外してある。
**古スキーマ警告は読み書き両方に掛ける。** `handleReadToolCall` / `handleWriteToolCall` のどちらもディスパッチ本体を包んで `appendStaleSchemaWarning` を1箇所で足す。片側だけだと「同じ古さなのに読み取りでしか知らされない」ことになる（`gkill_delete_kyou` / `gkill_restore_kyou` の `targets` が非string型の後付け引数）。

**OAuth の `authenticateUser` は `makeOAuthAuthenticateUser`（`lib/mcp-server-base.mjs`）1本。**3サーバへ逐語コピーされていた18行で、片方だけ直すと静かにずれる。

**OAuth の scope はサーバ種別ごとに1値で、正本は各エントリスクリプトの `START_SPEC` 1箇所。** metadata の `scopes_supported`・authorize の既定値と不一致拒否・トークン発行・Bearer 受理検証（不一致は 403 `insufficient_scope`）の全部が bootstrap 経由でこの1値から生成される。**テストで scope を検証するときは自前の値で `OAuthServer` を組み立てず、`START_SPEC` を import する** —— ReadWrite サーバが `gkill:read` を広告し read トークンで書き込めた事故は、全テストが自前の値を使っていて宣言値を誰も読んでいなかったために出荷まで漏れた（`start-spec.test.mjs` が宣言値と3サーバ間の重複禁止を固定する）。`scopes_supported` へ複数値を載せる・既存トークンを移行救済で通す、はどちらも否決（[ADR-0617](../../../documents/adr/0617-oauth-scope-is-one-value-per-server-kind.md)）。

**GPSのカーソルは `gkill_get_kyous` のものと別方式。** Kyou 側は Go 製の複合カーソル `{RFC3339Nano}::{ID}`、GPS 側は Node 製の base64url。コーデックの正本は `lib/gps-cursor.mjs` 1本だけで、発行（`paginateGpsLogs`）と受理（`normalizeGpsArgs`）の**両方がここを import する**。以前は受理側に Kyou 用の RFC3339 検証がコピペされており、説明文どおり `next_cursor` を verbatim で渡すと 100% 弾かれていた。片側だけのテストでは検出できないので、境界をまたぐ往復テストを消さないこと。

**Plugin tools** — 3サーバ共通は `lib/plugin-tools.mjs` の `gkill_get_plugin_list` 1つだけ（読み取り専用。`post_plugin_config` は公開しない）。プラグインKyouの本文は gkill に保存されておらず、`get_kyous` が返すのはメタデータと `rep_name`/`kyou_id` を持つ `payload.kind="plugin"` だけ。本文は **`gkill_get_kyous` に `include_plugin_content:true` を渡して同じレスポンスに埋め込ませる**（1件ずつ取る `gkill_get_plugin_content` ツールは廃止。AIの往復が N+1 回になり非効率だったため）。インライン化は `plugin-tools.mjs` の `inlinePluginContents` が担当し、`/api/get_plugin_content_html` を rep ごとに直列・rep 間は並列（既定4）で叩き、`lib/html-text.mjs` でHTML→テキスト変換して `payload.content_text` に入れる。各ペイロードには `content_status`（`ok`/`truncated`/`skipped`/`error`）が付き、`ok` 以外なら本文は不完全。上限は1件4000文字（`plugin_content_max_text_length` で最大200000まで）・合計200000文字・20件・30秒。`plugin_content_format` で `text`（既定）/`html`/`both` を選べる。 同一プラグインへ並列に投げても速くならない理由は [ADR-0602](../../../documents/adr/0602-mcp-inline-plugin-content.md)。

> **同一プラグインへ並列に投げてはいけない。** プラグインの stdio は1本しかなく呼び出しは直列化されるので、同時発行しても速くならず順番待ちが伸びるだけ。待ちきれなかったぶんは `ErrPluginBusy` になる。インライン取得が実行中のリクエストを abort しないのも同じ配慮（abort 自体は現在プロセスに影響しないが、MCPサーバは古い gkill にも接続しうる）。デッドラインは「新しいリクエストを始めない」だけで実現している。Write専用サーバには `gkill_get_kyous` が無いため、プラグイン本文を読むには ReadWrite サーバを使う。
>
> （2026-08-06以前の gkill は30秒デッドラインを排他ロックの**前**に張っていたため、同時発行するとロック待ちで期限を食い潰し、期限切れ時の `Process.Kill()` でプラグインプロセスが回収されていた。現在は期限をスロット取得後に張るのでこの誤射はない。）

**削除と過去版は `gkill_get_kyous` からは絶対に見えない。** gkill は追記型で、update は新しい版の追加、delete は `is_deleted=true` の版の追加。検索は最新版かつ未削除しか返さず、**`query.only_latest_data:false` は受理されるが無視される**（旧版を落としているゲートは `handle_get_kyous_mcp.go` の2箇所ではなく最低6箇所あり、うち `find_filter.go` の `replaceLatestKyouInfos` はフラグで分岐しないので、既知の2箇所を外しても件数は1件も変わらない。`dao/reps/repositories.go` がバケットキーを `ID+UpdateTime` に変える一方 `filterTagsKyous` は素のIDで引くため、開けるとタグ絞り込みが全部0件になる）。**代わりに使うもの**: 1件の全版は `gkill_get_kyou_history`（`data_type` 必須。型非依存の `/api/get_kyou` は `UnWrap()` でキャッシュを丸ごとバイパスするので使わないこと）、削除の取り消しは `gkill_restore_kyou`、削除済みの列挙は `query.include_deleted_data:true`（かつて併存した `query.is_deleted` は一度も実装されず、2026-08-24 に語彙ごと削除した。送ると未知キーとしてエラーになる）。**`update_time` は1秒解像度**なので、同じ秒の中で delete→restore すると新しい版が最新と見なされず黙って何も起きない。`write-handlers.mjs` の `nextUpdateTime` が必ず1秒以上進める。 却下案と6箇所のゲートの棚卸しは [ADR-0605](../../../documents/adr/0605-mcp-version-history-is-a-dedicated-tool.md)、削除の旗の意味論は [ADR-0110](../../../documents/adr/0110-deleted-data-opens-only-with-include-deleted-data.md)。

**`gkill_get_kyous` は `partial` と独立に `warnings` を必ず確認する。** `partial=false` はページング上の打ち切りが無いことだけを示し、記録保管場所の読み込み失敗で結果が欠けていても false のままになりうる。
その場合は利用可能な場所の結果を使いながら利用者へ警告し、警告された名前を `query.reps` へ指定し直さないこと。
指定すると利用可能な場所まで絞り落として、0件に見える。通常検索・`count_only`・`group_by` の全経路で
既存の `warnings` を保持し、新しい部分成功フラグを足さない（[ADR-0216](../../../documents/adr/0216-detach-a-broken-rep-but-never-silently.md)）。

**ディスパッチと要約の正本は `lib/read-handlers.mjs` と `lib/write-handlers.mjs` の2本だけ。** サーバファイル（`gkill-*-server.mjs`）に case を書き足さないこと。以前は read が read/readwrite へ、write が write/readwrite へ逐語コピーされており、readwrite 側だけが廃止済み `callRead` を呼び続けて、IDF ファイルパス取得のツールが静かに TypeError で落ち続けた（そのツールは 2026-08-24 に削除済み）。`buildToolResult` と3段の要約（plugin→read→write）も `lib/mcp-server-base.mjs` に1本化してある（write サーバの版だけ file-link 注入と IDF 画像ブロックを欠いた劣化コピーだった）。**ただし `const TOOLS = [...]` の配列は各サーバファイルに残すこと** ―― `src/tools/verify_docs.mjs` がサーバファイル中のこのリテラルを走査してツール数を数えており、lib へ移すとツール数が黙って0になる。削除・復活・履歴が共有する型別エンドポイント表は `lib/constants.mjs` の `ENTITY_TARGETS` 1箇所（以前は enum・`DELETE_DATA_TYPES`・対応表2つの4箇所に散っていた）。

**Environment variables:** `GKILL_BASE_URL`, `GKILL_USER`, `GKILL_PASSWORD_SHA256`, `MCP_LOG` (default `info`), `GKILL_MCP_MAX_FILE_BYTES` (default 8MB — cap for `get_idf_file` base64), `GKILL_MCP_FILE_LINK_TTL_MS` (default 1h — HTTP-mode file URL token TTL), and for HTTP mode: `MCP_TRANSPORT`, `MCP_PORT`, `MCP_BIND_ADDR` (listen address; default `0.0.0.0`, set `127.0.0.1` behind a reverse proxy/tunnel), `MCP_OAUTH_ISSUER`. Also read: `GKILL_PASSWORD` (plaintext alternative to the SHA256), `GKILL_SESSION_ID`, `GKILL_INSECURE`, `GKILL_LOCALE`, `GKILL_FETCH_TIMEOUT_MS`, `GKILL_HOME`.

**ツールスキーマはクライアントのセッション寿命で固定される。** クライアントは接続時に `tools/list` を1回取るだけで、サーバを再起動しても**生きているセッションには新しいスキーマが届かない**（2026-08-24 の再監査で、削除済みの `gkill_get_idf_file_path` を配り続ける live コネクタから実測）。そのため後から足したトップレベル引数は**正規JSON文字列**として届く（`"true"` / `'["nlog"]'`）。救済は `lib/normalization.mjs` の `reviveStaleSchemaArgs` で、`KYOUS_` / `GPS_` / `APP_CONFIG_` / `IDF_` / `REP_NAMES_` / `KYOU_HISTORY_STALE_SCHEMA_ARG_KINDS` の6表に載っている引数だけが型を復元される。**引数をスキーマへ足したら必ず同じ表と `STALE_SCHEMA_ARG_KINDS_BY_TOOL` へも足すこと。** 忘れると boolean / number / 配列の新引数は既存の全クライアントから型エラーになり、「新しいセッションでだけ動く」状態で出荷される（`is_video` が実際にそうなった）。string 型（`group_by` / `cursor` / `thumb` / `contains`）は素通しするので表に載せなくてよい。**古さが証明できたときは黙って直さず伝える**: 表の引数が文字列で届いた／廃止済み引数（`include_id` 等）が届いたときだけ `detectStaleSchemaSignals` が検出し、`handleReadToolCall` が payload の `warnings` へ「接続し直せ」を1行足す（証明できないときは出さない。推測で出すと常時ノイズになる）。2026-08-24 の実利用報告では、指摘9件のうち4件が「既に直っているのに古いスキーマから見えていなかった」だった。**同じ理由で、ツールを消したら `lib/constants.mjs` の `REMOVED_TOOL_HINTS` へ1行足す** —— 旧セッションはそのツールを一覧に持ったまま呼び続けるので、`Unknown tool` だけ返すと代替が分からない。

**サーバ3種は「どのツールを載せるか」しか違わない。** `handleToolCall`（plugin → read → write）は `lib/mcp-server-base.mjs`、起動処理は `lib/mcp-server-bootstrap.mjs` が正本で、`gkill-read-server.mjs` / `gkill-write-server.mjs` / `gkill-readwrite-server.mjs` が持つのは `{tools, readToolNames, writeAppName}` と起動 spec の表1行だけ。**readwrite は read + write そのもの**で、read サーバと同じ `READ_TOOLS` 配列を spread する（名前集合は `WRITE_TOOLS` と重ならない）。**サーバ側へディスパッチや起動処理を書き戻さないこと。** 「read と readwrite でスキーマが違う」という報告は 2026-08-25 に実際に来たが、実体は**昨日起動したままの MCP プロセスが昨日の定義を配っていた**だけだった（6項目すべてが `git show HEAD:` 版の文言と一致）。コードを疑う前にプロセスの起動時刻を見ること。詳細と却下案は [ADR-0613](../../../documents/adr/0613-add-and-update-share-one-field-table.md)。

**追加と更新の引数検証は `ENTITY_FIELD_SPECS`（`lib/write-normalization.mjs`）が正本。** 型ごとに手書きすると片方だけ厳しくなって静かにずれる —— 実際 `gkill_add_urlog` だけが `assertUrlWithScheme` を通し、`gkill_update_urlog` は素通しで、スキームの無いURLで更新すると gkill がページ取得を試みず **title が空のままエラーも出さずに保存**されていた。1行要約も同じ理由で `ENTITY_SUMMARIZERS`（`lib/write-handlers.mjs`）から作る。**`normalizeXxxArgs` の中身を手書きへ戻さないこと。**

**書き込みの後付けフラグはリクエストの修飾子であって、エンティティの列ではない。** `gkill_add_urlog` の `fetch_metadata` / `fetch_favicon`（既定 true = 外向き取得する。false で対象サイト・favicon サービスへの取得を項目別に抑止。Go 側へは `skip_fetch_*` へ**反転**して写す）と `gkill_add_mi` / `gkill_update_mi` の `allow_create_board`（既定 true = 未知の板名は従来どおり新しい板になる。false のときだけ板一覧と**完全一致**照合して typo を弾く）が該当し、どれも実体オブジェクトへは載せない（update_mi の `patchFields` へ足すと「変更なし」判定まで壊れる）。**update_urlog に抑止フラグは無い** —— Go は `re_get_urlog_content:true` を明示されたときだけ再取得し、MCP はそれを送らないので、update 経路に抑止すべき取得が無い（説明文にもそう書いてあり、`write-handlers.test.mjs` が非送信を固定する）。**板名未指定時の既定板への補完値は照合しない**（新規アカウントは板が0件で、照合すると既定板すら弾いてタスクが作れない）。既定値では照合の往復も発生させない。**後付けの boolean 引数は `revivesStaleBoolean` と `STALE_SCHEMA_ARG_KINDS_BY_TOOL` の両方へ載せる**（`write-tool-handlers.test.mjs` の表駆動メタテストが機械強制。片方を忘れると既存の全クライアントから型エラーになる）。既定を「取得しない」へ倒す案・2フラグを1本に束ねる案の却下理由は [ADR-0618](../../../documents/adr/0618-urlog-outbound-fetch-is-default-on-with-opt-out.md)。

**「cursor と併用できない」の検査は `assertAggregationNotCombinedWithCursor`（`lib/normalization.mjs`）1本。** `count_only` / `group_by` と `cursor` の併用は gkill 側でも弾かれるが、返るのは `ERR000352: 記録の取得に失敗しました` という汎用文で**理由が本文に一切乗らない**（実測 2026-08-25: 検索失敗と区別が付かなかった）。GPS 側（`paginateGpsLogs`）は前から MCP 層で弾いており、`get_kyous` だけが素通しだった。

**プラグインの説明文を Kyou ごとに焼き込まない。** 130〜150字あり、20件取ると同じ文が20回並ぶ（`include_plugin_content` とは無関係に常に載っていた）。`PluginPayloadMCPDTO` は `rep_name` / `plugin_name` だけを持ち、説明は応答トップレベルの `plugins[]` へ rep 名ごと1回。**`provides` を宣言しないプラグインには `typed_index` が付かない**ので、`gkill_get_plugin_list` からは「取り込み0件」と「正常」の区別が付かない（`is_alive:true` のまま全期間0件が実際に起きた）。件数はここでは数えない（全プラグインへ直列に往復することになる。ADR-0301）。代わりに warnings で名指しし、`count_only` + `data_types` での数え方を案内する。

**IDF file access**: 経路は「優先順位」ではなく**用途で分かれる**（2026-08-24 実測、[ADR-0606](../../../documents/adr/0606-idf-file-reaches-ai-through-payload.md)）。stdio クライアントには `get_kyous` が `file_path`（絶対パス）を載せるので、それを直接読むのが最良（base64 もサイズ上限も無い）。HTTP クライアントには代わりに `file_url` / `file_url_full`（`FileLinkStore` が発行する1ファイル束縛・期限付き・推測不能な公開URL。`GET /files/{token}` が Bearer 無しで配信し、画像は `file_url` がサムネ・`file_url_full` が原寸）が載るが、**MCP はそのURLを代わりに取りに行かない**し、クライアントが自前の機能で取れても会話の外側のバイトになるだけなので、これは人間へ渡すリンクと考える。**HTTP 接続の AI が画像を「見る」唯一の経路は `gkill_get_idf_file`**（`buildToolResult` が `type:"image"` の content ブロックを作る唯一の入力）で、画像生成ツールへ参照画像として渡す場合もこれ。上限（`GKILL_MCP_MAX_FILE_BYTES`、既定8MB）に当たる画像・動画は `thumb="WxH"`（一辺最大1024 = Go 側 `thumbFileServer.maxSize`。動画は `is_video:true` 併用）で縮小取得する。上限超えの `thumb` を渡すと Go は**黙って原本を返す**ので MCP 側で先に弾いている。**「URL を優先しろ」と書き戻さないこと** —— 実使用の ChatGPT はその誤誘導を無視して正しく base64 を呼んでいた。Local paths never go to remote clients; the transport type (stdio vs HTTP) is the gate. gkill itself stays private.

**HTTPモードの1リクエスト文脈は `server.current*` 共有フィールドに書かず、不変の `requestContext={sessionId,userId,remoteAddr}` を `handlePayload→handleMessage→handleToolCall/buildToolResult` へ引数で流す**（2026-08-21、監査 C-02）。以前は `_lastTokenUserId` と `server.currentSessionId/currentUserId/currentRemoteAddr` に書いて await をまたいで読んでいたため、並行リクエストで別要求の user/session が混線し、他人のセッションに紐づく file-link URL 発行や書き込みレコードの作成者誤記が起きた。`mcp-server-base.mjs` の入口で `requestContext ?? Object.freeze({...this.current*})` にフォールバックするので stdio と既存の直接呼び出しテストは無改修。**http-transport 側から `server.current*` への書き込みを復活させないこと**（渡し忘れは sessionId=null→gkill側401で顕在化し、静かな混線には戻らない）。OAuth は S256 必須・未登録 client_id は認可拒否（`oauth-server.mjs` の `_validateAuthorizeParams`）、公開ファイル配信は nosniff + CSP sandbox（Go 側 `withUserContentSecurityHeaders` のミラー）、`oauth-store.mjs` の保存は temp+rename の 0600。守るテストは `src/mcp/__tests__/http-transport.test.mjs`（Bearer 401 = C-01 回帰・並行分離 = C-02 回帰・M-06）。 却下案（AsyncLocalStorage 等）と渡し忘れが顕在化する理由は [ADR-0601](../../../documents/adr/0601-mcp-request-context-immutable.md)。

## 関連スキル

- [gkill-plugin](../gkill-plugin/SKILL.md) — プラグインの stdio 直列化（並列に投げても速くならない理由）
- [gkill-find-query](../gkill-find-query/SKILL.md) — 旧形式JSONキーの移行（`LEGACY_USE_FLAG_KEYS`）
- [gkill-go-backend](../gkill-go-backend/SKILL.md) — サーバ側の認可・セキュリティヘッダ

## 詳しい設計と却下案（ADR）

- [ADR-0707 端末固有の文字列は出口で伏せ、診断文はAIへ返さない](../../../documents/adr/0707-redact-environment-specific-strings.md)
- [ADR-0601 リクエスト文脈は不変で引数渡し](../../../documents/adr/0601-mcp-request-context-immutable.md)
- [ADR-0602 プラグイン本文はインライン埋め込み](../../../documents/adr/0602-mcp-inline-plugin-content.md)
- [ADR-0603 カーソルは期間の端を進める](../../../documents/adr/0603-mcp-cursor-pushes-period-end.md)（Superseded）
- [ADR-0604 複合カーソルと厳密上限・count_only/group_by](../../../documents/adr/0604-mcp-composite-cursor-strict-limits.md)
- [ADR-0605 版履歴と削除の取り消しは専用ツール](../../../documents/adr/0605-mcp-version-history-is-a-dedicated-tool.md)
- [ADR-0110 削除済みは include_deleted_data だけが開ける](../../../documents/adr/0110-deleted-data-opens-only-with-include-deleted-data.md)
- [ADR-0611 同じ対応表を「表」と「直書き」の2形態で持たない](../../../documents/adr/0611-one-table-not-two-forms.md)
- [ADR-0612 付随データは Kyou 検索と同じ規則で判定する](../../../documents/adr/0612-attached-data-follows-kyou-search-rules.md)
- [ADR-0613 追加と更新は1つのフィールド表から作る](../../../documents/adr/0613-add-and-update-share-one-field-table.md)
- [ADR-0617 OAuth の scope はサーバ種別ごとに1値](../../../documents/adr/0617-oauth-scope-is-one-value-per-server-kind.md)
- [ADR-0618 urlog の外向き取得は既定で行い、引数で項目別に抑止する](../../../documents/adr/0618-urlog-outbound-fetch-is-default-on-with-opt-out.md)
