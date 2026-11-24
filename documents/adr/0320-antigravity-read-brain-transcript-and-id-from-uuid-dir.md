# ADR-0320: Antigravity の会話は brain の transcript_full.jsonl から読み、会話IDは直近の UUID フォルダ名から採る

| | |
|---|---|
| Status | Accepted |
| Date | 2026-11-23 |
| Sources | `.claude/skills/gkill-plugin/SKILL.md` / `src/plugins/gkill_plugin_antigravity/README.md` |
| Supersedes | なし |
| Superseded-by | なし |
| Anchors | `src/plugins/gkill_plugin_antigravity/scan.go`（`classifyFileName` / `conversationIDOf` / `pickLargestPerConversation`） / `src/plugins/gkill_plugin_antigravity/loader.go`（`buildMessages` / `kyouIDOf`） / `src/plugins/gkill_plugin_antigravity/summaries.go`（`readConversationSummaries`） |

## Context

Google Antigravity（Gemini ベースのエージェント IDE）での作業がライフログに残らない。
ChatGPT / Claude.ai / Claude Code / Codex は同梱プラグインがあるのに、Antigravity だけが抜けていた。

Antigravity は `~/.gemini/antigravity` に会話を置く。同じ会話が4つの形で書かれている。

| パス | 中身 |
|---|---|
| `conversations/<会話ID>.db` | SQLite。`steps.step_payload` が **protobuf バイナリ**（`trajectory` / `cascade_id` という Windsurf 系の語彙） |
| `brain/<会話ID>/.system_generated/logs/transcript_full.jsonl` | **1行1ステップの JSONL。全会話にある** |
| 同 `logs/transcript.jsonl` | 同じ行数の**省略版**（`truncated_fields` 付き） |
| 同 `logs/chunks/transcript[_full]/<8桁連番>.jsonl` | 同内容を一定サイズで割った写し |
| `conversation_summaries.db` | SQLite。会話一覧（平文の `title` / `preview` / `workspace_uris`）と fts5 の検索索引 |

`transcript_full.jsonl` の1行は `step_index` / `type` / `source` / `status` / `created_at` を必ず持ち、
`type` によって `content` / `thinking` / `tool_calls` / `model` / `error` / `media` が付く。
**会話IDはどのキーにも入っていない**（`id` を名前に含むキーが1つも無い）。ファイル名も全会話で同じ `transcript_full.jsonl` で、
会話IDが現れるのは `brain/<会話ID>/` のフォルダ名だけ。

## Decision

- **読むのは `transcript_full.jsonl` と `conversation_summaries.db` だけ。** 種別はファイル名だけで決める
  （`classifyFileName`）。省略版と `chunks/` の写しは自動的に外れる。
- **会話IDは「そのファイルから見て直近の、UUID の形をした祖先フォルダ名」**（`conversationIDOf`）。1つの規則で、
  生の置き場（`brain/<会話ID>/.system_generated/logs/…`）と同期先（`Kyou/Antigravity_<端末>_<日付>/<会話ID>/…`）の
  両方に効く。UUID の祖先が無いファイルは**読まずに設定画面の警告へ出す**。
- **同期も会話IDのフォルダを保つ。** `SyncDatas.ps1` / `syncDatas.sh` は `brain/*` を回して
  `Antigravity/<会話ID>/transcript_full.jsonl` へ置く。
- 同じ会話IDのファイルが複数見つかったら**一番大きいものだけ**を読む（ログは追記のみ。
  `gkill_plugin_vrchat` の「同じ起動のログが2か所」と同じ規則）。
- **Kyou の粒度はほかの AI チャット系4本と同じ** ―― `USER_INPUT` 1件で人間の Kyou、次の `USER_INPUT` までで応答の Kyou。
  応答の本文は空でない最後の `PLANNER_RESPONSE.content`（[ADR-0314](0314-codex-conversation-from-item-completed.md) の
  「最終回答の印が無いログでは最後の発言を本文にする」規則）。
- **Kyou ID は `UUIDv5(名前空間, "<会話ID>/<発言者>/<その単位の先頭レコードの step_index>")`。**
  レコードに id が無いので合成する。`step_index` は追記のみの通番なので会話が伸びても動かない。
- **ツールの実行結果の `content` は保持しない**（`GENERIC` / `VIEW_FILE` / `LIST_DIRECTORY` / `GREP_SEARCH` /
  `RUN_COMMAND` / `SEARCH_WEB` / `CODE_ACTION` / `GENERATE_IMAGE`）。`error` だけ1行の注記にする。
- **会話タイトルとプロジェクト名は `conversation_summaries.db` から読む任意の索引**扱い
  （Codex の `session_index.jsonl` と同じ）。読めない・載っていない会話は最初の人間の発言の冒頭で代用する。
  `kyou_cache` の別カラムで持ち `search_text` に焼かないので、あとからタイトルが付いても会話を読み直さない。
  この1本だけは差分（`(mtime, size)`）を見ずに毎回読む ―― SQLite はページ単位で書くので、
  タイトルが変わってもファイルのサイズは動かず、秒精度の mtime も同じ秒なら動かない。

## Rejected alternatives

- **`conversations/<会話ID>.db` の protobuf を解く（正本なので）** — スキーマが無い。`step_payload` は
  `step_type` が数値のままで、`TRAJECTORY_STEP_TYPE_*` のような enum 名の文字列はアプリのバンドルにも無い。
  形が変わったときに静かに読み違えるし、SQLite を同期で運ぶと会話を1件書き換えるたびにファイル全体が運ばれる。
  一方 `transcript_full.jsonl` は全会話に揃っていて、追記のみで、読める。
- **`conversation_summaries.db` の fts5 影テーブルの平文本文を使う** — 検索用のチャンクなので、
  発言の境界もツール呼び出しも thinking も無い。一覧の行に出す本文が作れない。
- **省略版 `transcript.jsonl` を読む** — `content` が切られている。`transcript_full.jsonl` と同じ行数あるので、
  両方読むと会話が二重になる。
- **同期で `<会話ID>.jsonl` へ改名して平置きする（ClaudeCode と見た目を揃える）** — 改名には
  `dvnf copy -f` が要り、`-f` は毎回上書きするので mtime が毎回動く。変わっていない会話まで
  毎回コピーされ、同期ハブへも毎回上がる。フォルダで分ければ `-f` が不要で差分コピーが効く。
- **パスから代用の会話IDを作る（パスのハッシュなど）** — 同期先のフォルダ名は端末名と日付を含むので、
  生の置き場と同期先で別の会話になり、同じ会話が二重に出る。フォルダを掘り直しただけでも ID が全部変わる。
- **`step_index` ではなく発言の連番で ID を作る（Codex と同じ材料）** — Codex のレコードには通番が無いので
  連番を振るしかないが、Antigravity は `step_index` を自分で持っている。途中に行が挿し込まれても
  後続の ID が動かないほうを採る。

## Consequences

- Antigravity が `brain/` の派生ログを書かなくなったら、このプラグインは0件になる（`conversations/*.db` は読まない）。
  設定画面の「対象ファイル数」で気づける。
- 会話IDのフォルダを外して平置きすると読まれない。設定画面に警告が出る。
- `provides` を書いていないので、タグも型別データも出さない（ほかの AI チャット系4本と同じ）。
  タグを付けると rykv の既定の絞り込み「タグ無し」から漏れて何も表示されなくなる。
- 同梱プラグインは8本から9本になり、AI チャット系（`chatview` を使う本数）は4本から5本になった。

## Evidence

実測（この PC の Antigravity、十数会話・数百ステップ）:

- `transcript_full.jsonl` は**全会話にあった**。`transcript.jsonl` も同数あり、行数は同じ。
- レコードに `id` を名前に含むキーは**1つも無かった**。`created_at` は全行 RFC 3339 の `…Z`。
- ツール結果の種別の件数は `PLANNER_RESPONSE.tool_calls` の件数と**ほぼ1対1**で続く（対角が支配的）。
  ツール結果の `content` はファイル閲覧で中央値 KB 級・最大で数十 KB。
- `PLANNER_RESPONSE` の多く（3分の2ほど）は `content` が空で、ツール呼び出しだけのステップだった。
  だから「空でない最後の `content` を本文にする」が要る。
- `model` を持つレコードは一部の会話に偏っていた。生ログからの独立な数え直しと、
  作ったキャッシュの「モデル名が入った応答の件数・会話数」が一致することを確認した。
- 単独モード（`--gkill-build-cache`）で1周させると、会話の数ぶんの transcript と索引1つを読み、
  人間の発言と応答がそれぞれ `USER_INPUT` の件数と応答のまとまりの件数に一致した。読めなかった行は0。

## Related tests

- `src/plugins/gkill_plugin_antigravity/scan_test.go`（`TestScanSourcesPicksOnlyTranscriptFull` /
  `TestConversationIDOfBothLayouts` / `TestScanSourcesReportsMissingConversationID` /
  `TestScanSourcesPicksLargestPerConversation`）
- `src/plugins/gkill_plugin_antigravity/loader_test.go`（`TestBuildMessagesSplitsPerUserInput` /
  `TestBuildMessagesKeepsRecordsBeforeFirstUserInput` / `TestBuildMessagesWithoutUserInput` /
  `TestBuildMessagesDropsToolResultContent` / `TestKyouIDIsStableAcrossAppend`）
- `src/plugins/gkill_plugin_antigravity/cache_test.go`（`TestSummariesChangeRefreshesTitleWithoutReingest` /
  `TestBuildIsIncrementalPerFile` / `TestConcurrentReadDuringBuild`）
