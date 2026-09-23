# ADR-0634: 利用者ごとのスキル（SKILL.md と付属ファイル）を $GKILL_HOME/skills に置き、gkill_server が読み書きし、MCP と設定画面はその API を使う

| | |
|---|---|
| Status | Accepted |
| Date | 2026-09-23 |
| Sources | 利用者の要求「ChatGPT と Claude 向けに、利用者ごとのスキル（SKILL.md など）を MCP が返す仕組みが欲しい」。壁打ちで決めたこと: 置き場所は `skills/<user_id>/<name>/`、AI は読み書きし即時反映、履歴・未確認バッジ・アップロード衝突の検出は持たない、画面は一覧・表示・zip のダウンロード / アップロード（2段階確認つきの丸ごと置き換え）・スキル丸ごとの削除だけ、MCP の削除ツールは実装だけして公開しない、保存量の上限は設けない |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/server/gkill/dao/skills/`（保存層）/ `src/server/gkill/api/gkill_server_api/handle_*skill*.go` / `src/server/gkill/mcp/skill_handlers.go` / `src/server/gkill/mcp/skill_delete_tool.go` / `src/client/classes/use-manage-skill-list-dialog.ts` |

## Context

ChatGPT には Skills の仕組みが無く、claude.ai のスキルは ChatGPT と共有できない。利用者が「週次まとめの作り方」
「この種の記録にはこのタグを付ける」といった手順を AI に渡すには、会話のたびに貼るしかなかった。
gkill の MCP は両方の AI から繋がるので、手順書を gkill 側に1か所で持てば、どちらの AI も同じ手順で gkill を扱える。

手順書は Agent Skills と同じ形（フォルダ1つ = スキル1つ。`SKILL.md` の frontmatter に `name` / `description`、
本文に手順、必要なら参考資料・スクリプト・素材のサブファイル）にすれば、利用者が AI 製品のあいだで持ち運べる。

gkill の原則として、MCP は起動中の gkill_server の HTTP クライアントで、データのファイルを直接触らない（ADR-0631）。
設定画面から扱うなら、配信するのも gkill_server になる。

## Decision

- 置き場所は `$GKILL_HOME/skills/<user_id>/<skill-name>/`（`gkill_options.SkillsDir`）。スキル名は英小文字・数字・ハイフンで
  1〜64文字（先頭と末尾は英数字）、ファイルのパスは `/` 区切りで各要素が英数字で始まり英数字と `._-` だけ（`dao/skills/path.go`）。
  フォルダは実体を持たずパスの一部として扱う。共通スキルの置き場として `_global/` の名前だけ予約する（今回は作らない）。
- ファイルを読み書きするのは gkill_server だけ（`dao/skills`）。API は `/api/get_skill_list`・`/api/get_skill`・
  `/api/download_skill`・`/api/upload_skill`・`/api/write_skill_file`（MCP 専用）・`/api/delete_skill` の6本。
- MCP は `gkill_get_skill_list` / `gkill_get_skill`（3サーバ）と `gkill_add_skill` / `gkill_update_skill`（write / readwrite）を公開し、
  `gkill_status` にもスキルの名前と説明を載せる。書き込みは revision（中身の SHA-256 先頭16桁）による楽観ロックで、反映は即時。
- AI 向けの削除ツール `gkill_delete_skill`（ファイル単位のみ）は実装するが、ツール一覧と振り分けの行をコメントアウトして公開しない。
- 設定画面（ApplicationConfig の設定ボタン2段目の末尾「スキル」）は一覧・中身の素のテキスト表示・zip のダウンロード・
  zip のアップロード・スキル丸ごとの削除だけ。アップロードは dry_run で「追加・削除・変更・無視されるファイル」を見せてから
  同じ zip で丸ごと置き換える（DELETE_WRITE）。

## Rejected alternatives

- **`$GKILL_HOME/mcp/<user_id>/…`（使い手の名前で切る）** — `$GKILL_HOME` 直下（`lib` / `caches` / `logs` / `configs` / `tls` / `datas`）は
  すべて中身の種類で名付けられている。画面からも触るようになった時点で `mcp` は持ち主ではなくなる。`ai/` や `agents/` も
  何でも入る入れ物になりやすく、別の種類が増えたら別のトップを切ればよい。
- **`datas/<user_id>/skills/`** — rep のパス指定は glob で、ファイルやディレクトリにも当たる。rep の置き場に rep でない
  ディレクトリを置くと、「Git 直下の非 git エントリ1つで全 API が内部エラーになる」と同じ型の事故を招く。
- **キャッシュと同じ「機能 → user_id」の並びに揃える / キャッシュを「user_id → 機能」へ揃え直す** — 並びは操作の単位で決める。
  キャッシュは機能ごとに消す・測る・別ドライブへ逃がす（動画キャッシュだけで数百 GB になる）ので機能が先、スキルは利用者の持ち物で
  `datas/<user_id>/` と同じく「種類 → user_id」。キャッシュを並べ替えるのは数百 GB の移動・`clear_cache` の書き直し・
  配置スクリプトの確認が要るうえ、今回の目的と関係しない。
- **OAuth の状態ファイルも `skills` / `mcp` の側へ移す** — `configs/` はシステムが読み書きするもの（設定・状態・秘密情報）、
  `skills/` は利用者が AI 向けに書くもの。秘密情報を利用者が git 管理・同期したくなる場所に置くと、`git init` した瞬間に
  refresh token をコミットする事故が起こりうる。移すと既存のコネクタが再認可になる。
- **MCP プロセスがファイルを直接読む** — 読むだけなら足りたが、書き込みと画面が加わると検証・ロック・利用者の分離を2か所に
  実装することになる。gkill_server の API に集めれば1実装で済み、利用者もセッションから決まる。
- **MCP の prompts / resources で渡す** — prompts は利用者がメニューから選ぶもの、resources は利用者が添付するもので、ChatGPT の
  コネクタは tools しか使えない。AI が自分の判断で取りに行ける経路は、両方のクライアントで tools だけ。
- **履歴（変更ログと中身のハッシュ置き場）と「AI が書いた・未確認」のバッジ、差分表示** — AI が書いたものを確かめて戻すための
  仕組みとして検討したが、記録アプリの本質ではないので gkill に責務を持たせないと利用者が判断した。
  履歴が欲しくなったら利用者が `skills/<user_id>/` を git で管理すればよい（gkill は関与しない。ドットで始まる項目は一覧に出さない）。
- **アップロード衝突の検出（ダウンロードしてから AI が書き換えた分を、上げ直しが黙って消すのを止める）** — 利用者単位で管理しており、
  今の利用者は1人なので、意識していれば足りると利用者が判断した。利用者が増えたら見直す。
- **AI にスキルの削除を公開する** — 履歴が無いので、AI が消したファイルは戻せない。AI の誤操作や、読み込んだ記録・Web ページに
  紛れ込んだ指示に従った削除を、利用者が後から戻す手段が無い。丸ごとの削除は画面だけに置き、ファイル単位の削除ツールも
  実装だけにとどめた（公開するなら履歴か確認の仕組みとセットで、この ADR を見直す）。
- **保存量の上限（ファイルサイズ・ファイル数・スキル数）** — 個人利用でローカルで完結する運用なので設けない。アップロードの本文は
  既存のアップロード経路の枠（1GB）に載せる（認証付き経路の 32MB 枠では収まらない）。
- **画面でのファイル単位の編集（テキストエディタ・フォルダ操作）** — 手元のエディタで直した zip を上げ直せば足りる。画面の責務を増やさない。
- **Markdown / HTML として描画する** — AI が書いた HTML の中のスクリプトが、ログイン中のセッションで gkill のオリジンで動く（保存型 XSS）。

## Consequences

- **スクリプトはサーバで実行しない。** 本文を返すだけで、動かすのはクライアント側のサンドボックス（Claude.ai のコード実行・
  ChatGPT の Python）。サンドボックスからは gkill に届かないので、スクリプトは「MCP で取ったデータを加工する」形で書く。
- **AI に返すときだけ、`max_file_bytes`（IDF と同じ設定。既定 8MiB）を超えるファイルは中身を省く**（`content_omitted`）。
  保存の上限ではなく返し方の分岐。省かないと数百 MB の base64 で応答が破裂する。
- **置き換えは一時ディレクトリへ展開してから入れ替える**（`_tmp-*` / `_old-*`）。Windows で誰かがファイルを開いていると rename が
  失敗し、既存のスキルは元のまま残る。スキルのフォルダに利用者が置いたドットファイル（`.git` 等）も置き換えで消えるので、
  git で管理するなら `skills/<user_id>/` の単位で置く。
- **利用者IDはアカウント作成時にしか形式を検査していない**ので、パス要素として使う前に検査する。Windows では大文字小文字だけ
  違う利用者IDが同じディレクトリを指すので、親を列挙して完全一致で照合する。
- **「見つからない」は `fs.ErrNotExist` を包まない。** 包むと応答の reason が `storage_unavailable` になり、404 に誤った理由が付く。
- `gkill_delete_skill` はツール一覧に無いまま `case` だけ戻しても呼べない（`Server.HandleToolCall` が `IsWriteToolName` で弾く）。
  名前を help topic や説明文に書かない（`help_topics_test` が落ちるうえ、AI に見えないツールを案内する）。定義は verify_docs が
  数えない `skill_delete_tool.go` に置く（`read_tools.go` / `write_tools.go` に `tool("gkill_…"` を書くとコメントの中でも数えられる）。
- ツールが4本増え、tools/list の予算が read 40,855 → 42,396、write 60,100 → 63,518、readwrite 84,582 → 88,000 になった。
  `schema_revision` が変わるので、ChatGPT と Claude.ai のコネクタは接続し直しが要る。

## Evidence

- tools/list の予算（`gkill_server mcp schema-budget --update` の実測）: read +1,541、write / readwrite +3,418。説明文を
  ADR-0622 の要約にとどめ、1本あたり約 850 バイト（既存ツールの平均は約 2.5KB）。
- golden を再生成した差分は、既存ケースでは initialize（`serverInfo.version` の schema 印）・tools/list・status の4ケース・
  help の目次2ケースだけで、ほかは追加した 14 ケース。
- 利用者の判断以外は実測なし — 脅威モデル（AI の誤操作・紛れ込んだ指示・保存型 XSS）からの判断。

## Related tests

- `src/server/gkill/dao/skills/store_test.go`（名前・パスの規則、frontmatter、テキスト判定のチャンク境界、revision の楽観ロック、
  zip の検証（`..`・絶対パス・シンボリックリンク・大文字小文字の重複・包むフォルダの剥がし）、計画と置き換え、置き換え失敗時に既存が残る、利用者の分離）
- `src/server/gkill/api/gkill_server_api/handle_skill_test.go`（6ルートの一巡とステータス・エラーコード）
- `src/server/gkill/api/gkill_server_api/api_routes_test.go` / `src/server/gkill/api/gkill_server_api/auth_middleware_capped_test.go`（ルート表と、アップロードの枠）
- `src/server/gkill/mcp/skill_handlers_test.go`（送る要求の形、画像の image ブロック、省いたときの warnings、削除ツールが公開されないこと、status の skills）
- `src/server/gkill/mcp/golden_test.go`（`skill *` / `help: skills` のケース）
- `src/client/__tests__/unit/composables/manage-skill-list-dialog.test.ts`（アップロードの2段階が同じ zip を送る、丸ごと削除、ダウンロード、応答の追い越し）
- `src/client/__tests__/e2e/skills.spec.ts`（設定画面からの一巡）
