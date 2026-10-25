# gkill_plugin_codex

OpenAI Codex CLI のセッションログ（ロールアウトJSONL）を gkill のタイムラインに表示するプラグイン。

**人間の発言1つ**と、**それに続く一連の応答**を、それぞれ1つの Kyou として扱う（Claude.ai プラグインと同じ1メッセージ1件）。
応答は Codex の最終回答を本文にし、途中報告・思考・ツール呼び出し・変更ファイルは「作業の経過」の折りたたみに入れる。
ツールの実行結果は保持しない（実データではログのバイトの9割超がこれで、日記としては要らない）。
サブエージェントの会話は親の応答に畳み込む。

gkill 本体には一切手を入れない。独立バイナリと `manifest.json` を所定の場所へ置くだけで動く。

## セットアップ

### 1. ビルド

```bash
cd src/plugins/gkill_plugin_codex
go build .

# Windows 向けにクロスコンパイルする場合
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_codex.exe .
# Android(Termux) 向け。純Go なので NDK も WSL も要らない
CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -o gkill_plugin_codex .
```

### 2. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_codex/
    manifest.json          # 必須。バイナリに埋め込んであるので --gkill-print-manifest で出せる
    gkill_plugin_codex     # Linux / macOS / Android
    gkill_plugin_codex.exe # Windows
    config.json            # 初回起動時に自動生成される。既存のものは上書きされない
```

ディレクトリ名・`manifest.json` の `name`・`executable` は**すべて `gkill_plugin_codex`** で一致させること。
`gkill_plugin_` の接頭辞も必須（運用でプロセス名からプラグインをまとめて見分けて更新前に止めるため。
接頭辞が無いと古いバイナリを掴んだまま生き残る）。

`manifest.json` と既定の `config.json` はバイナリ自身から出せる。

```bash
./gkill_plugin_codex --gkill-print-manifest > manifest.json
./gkill_plugin_codex --gkill-print-config   > config.json
```

> **PowerShell 5.1 では `>` を使わないこと。** UTF-16LE で書かれ、さらにプラグインの UTF-8 出力を
> CP932 として解釈するので二重に壊れる。壊れた `manifest.json` は
> `plugin_manager.go` が**無言で読み飛ばす**ので、プラグインが消えたようにしか見えない。
> 配布スクリプトはこれを避けるため `StandardOutputEncoding` と
> `File.WriteAllText(..., UTF8Encoding(false))` を使っている。

置いたら gkill を再起動する（リポジトリの構築時にプラグインを探すため）。

### 3. データソースのフォルダを指定する

`config.json` の `source_dirs` に、ロールアウトJSONL が入ったフォルダかファイルを書く。
設定画面（Kyou の右クリック →「プラグイン設定」）からも編集できる。

```json
{
  "source_dirs": [
    "~/.codex/sessions",
    "~/.codex/session_index.jsonl"
  ],
  "subagent_mode": "fold",
  "scan_workers": 0
}
```

| キー | 意味 |
|---|---|
| `source_dirs` | 走査対象。配列でも改行区切りの文字列でもよい。`*` `**` `?` `[]` のワイルドカード、先頭の `~`、環境変数（`$HOME` など）が使える。空にすると `~/.codex/sessions` を見る |
| `subagent_mode` | `fold`（既定・サブエージェントの会話を親の応答に畳み込む）か `own_kyou`（サブエージェントも独立した Kyou にする） |
| `scan_workers` | 同時に読むファイル数。0 なら自動（CPU数の半分、最大4） |

- フォルダを指定すると再帰的に走査して `rollout-*.jsonl` を探す
- `session_index.jsonl` も指定しておくと、詳細画面にスレッド名が出る
- 変更は**次の検索から反映される**（gkill の再起動は不要）
- `$HOME` は**リテラルで書いてよい**。プラグイン側が `os.ExpandEnv` する。
  ただし gkill を Windows サービスで動かしている場合は実行アカウントのホームになるので、
  確実に指定したいなら絶対パスにする

複数の端末で Codex を使っているなら、各端末の `~/.codex/sessions` を dvnf で
1か所へ集めたフォルダをパターンで指す（例: `~/PC/Codex_*`）。
こうすると別の端末で書いたログも読めて、gkill のバックアップ体系にも乗る。
同じセッションが複数の指定に含まれても、Kyou の ID が UUID なので重複は自然に統合される。

## 取り込む内容

会話は Codex の版で書かれる場所が違う。0.150 までは「旧レーン」、0.151 以降は「新レーン」（`event_msg/item_completed` の `item.type`）にだけ出る。

| ログのレコード（旧レーン / 新レーン） | Kyou での扱い |
|---|---|
| `event_msg/user_message` / `item_completed` の `UserMessage` | 人間の発言 = 1 Kyou。IDE・アプリ内ブラウザの前置きは剥がして折りたたみへ。画像は「(画像)」の印だけ |
| `event_msg/agent_message` / `AgentMessage` | 応答の発言。`phase` が `final_answer` なら本文、`commentary`（途中報告）なら作業の経過 |
| `event_msg/agent_reasoning` / `Reasoning` | 思考（作業の経過） |
| `response_item/function_call`（両レーン共通） | ツール呼び出し（引数を200文字に要約） |
| `response_item/function_call` の `request_user_input*`（両レーン共通） | Codex から人間への質問。設問と選択肢を本文に出す |
| `response_item/custom_tool_call`（両レーン共通） | ツール呼び出し。`apply_patch` は対象ファイル一覧に潰す |
| `response_item/web_search_call` / `item_completed` の `Extension`（kind `web.search`） | ツール呼び出し（検索語） |
| `response_item/tool_search_call` | ツール呼び出し（検索語） |
| `event_msg/patch_apply_end` / `FileChange` | 変更したファイル一覧（パス・種別・増減行数） |
| `item_completed` の `Plan`（旧版のころから新レーンだけ） | 計画 |
| `event_msg/task_complete` | 所要時間のチップ |
| `event_msg/turn_aborted` | 短い通知 |
| `event_msg/context_compacted` / `ContextCompaction` | 短い通知 |
| `event_msg/sub_agent_activity` | サブエージェントの親子リンク |
| `event_msg/task_started` / `turn_context` | ターンの区切り（両レーンの重複判定に使う）。`turn_context` はモデル名も |
| `session_meta` / `turn_context` | プロジェクト名・ブランチ・モデル名などのメタ情報 |

**保持しないもの**: ツールの実行結果（`*_output` / `mcp_tool_call_end`、新レーンの `CommandExecution` / `McpToolCall` / `ImageView`）、
Web 検索以外の `Extension`（画像生成は1行に画像そのものを抱えて 2MiB を超える）、
暗号化された思考（`response_item/reasoning`）、トークン数（`token_count` / `token_usage_record`）、
`response_item/message`（`event_msg` と二重になるうえ、環境プリアンブルの注入が混ざる）、
`compacted`（圧縮前の履歴を抱えていて、拾うと会話が二重になる）。

### 会話は `event_msg` の2つのレーンから読む

Codex のログは同じ発言が複数の場所に出る。実データで確かめた結果、

- 0.150 までは `event_msg/user_message` が**実際に人間が打った入力**
- 0.151 以降は `event_msg/user_message` などが**1行も出ず**、`event_msg/item_completed` の `UserMessage` だけになる。
  旧レーンしか読まないと人間の発言が見つからず、セッション全体がツール呼び出しだけの応答1件に溜まる（例外もエラーも出ない）
- `response_item/message` role=user には `<recommended_plugins>` や `<environment_context>` などの注入が混ざるので、どの版でも使えない

両方のレーンが同じターンに並ぶと会話が二重になるので、**旧レーンの会話が出たターンでは新レーンの会話を捨てる**。
ファイル単位で選ばないのは、旧版で始めたセッションを新版で resume すると1ファイルに両方の形が並ぶため。
ターンは `task_started` / `turn_context` の `turn_id` で追い、新レーンの項目は自分の `turn_id` を持つ。

ツール呼び出しは `response_item` にしか無いので、そちらから拾う。経緯と却下案は
[ADR-0314](../../../documents/adr/0314-codex-conversation-from-item-completed.md)。

### 応答は最終回答を本文にし、経過を畳む

実データでは最終回答1つにつき途中報告がおよそ6つ付く。rykv は一覧の行に詳細HTMLをそのまま描くので、
全部を本文に並べると会話が途中報告に埋もれる。

- 本文: `phase=final_answer` の発言と、Codex からの質問（時系列）
- 作業の経過（折りたたみ1つ）: 途中報告・思考・ツール呼び出し・変更ファイル・計画・通知・サブエージェント（時系列）
- 最終回答が無い区切り（作業中に人間が割り込んだ、`phase` を持たない古い版）は、最後の発言を本文にする
- Codex は質問ツールを呼ぶとき、たいてい同じ文面を最終回答にも書く。最終回答が設問の書き出し（30文字）を含むなら、
  質問は作業の経過の側へ回して本文に二度出さない
- 途中報告も検索テキストには入れるので、ワード検索では引ける

### 発言の前置きと質問への回答

VSCode 拡張から使うと、発言の先頭に開いているタブの一覧が付く（実データでは6割ほど）。

```
# Context from my IDE setup:
## Active file: main.go
## Open tabs:
- main.go: src/main.go
## My request for Codex:
バグを直してください。
```

`## My request for Codex:` より後ろだけを本文にし、前置きは詳細画面の折りたたみへ入れる。
rykv は一覧の行に詳細HTMLをそのまま描くので、前置きを残すとどの行もタブ一覧で埋まってしまう。

アプリ内ブラウザから送ると、先頭に `<in-app-browser-context>…</in-app-browser-context>`（表示中のページの情報）が付く。
これも剥がして折りたたみへ入れる（自動で付く定型文なので検索テキストには入れない）。

Codex の質問（`request_user_input*`）への回答は、人間の発言として
`<send_user_message_question_reply>[{"question": …, "answer": …}]</send_user_message_question_reply>` の形で届く。
包みと JSON を外して回答の文だけを本文にし、「あなた（質問への回答）」として描く。設問が複数なら「設問 → 回答」を並べる。
閉じタグが無いなど知らない形は、何も剥がさずそのまま本文にする。

## 動作の要点

### スレッドIDはファイル名の uuid

`session_meta.session_id` は使えない。実データでは**半数近くのファイルに存在せず**、
存在しても**サブエージェントでは親のIDが入っている**（ファイル数より少ない種類しかない）。
これをキーにすると親子の Kyou ID が衝突する。

`session_meta.id` はファイル名の uuid と**全ファイルで一致**するので、そちらを正とする。

### `session_meta` は identity と environment で扱いを分ける

1ファイルに1回〜十数回出る（resume のたびに書かれる）。
**サブエージェントのファイルには2つ目として「親の」`session_meta` が入っている**ので、

- identity（`id` / `thread_source` / `parent_thread_id` / `agent_path` / `agent_nickname`）は**1つ目だけ**を使う
- environment（`cwd` / `git.*` / `originator` / `cli_version`）は全部をマージして最初の非空を採る
  （1つ目の `git` が null で2つ目に入っている実例がある）

### Kyou の ID は合成する

`event_msg` のレコードには id が無いので、`UUIDv5(名前空間, "gkill_plugin_codex|スレッドID|ロール|連番")`
で作る。ロールアウトは追記のみなので連番は安定していて、同じセッションを2箇所
（実ログと集約コピー）から拾っても Kyou は重複しない。

**名前空間 (`uuid.go` の `codexNamespace`) は永久に変えないこと。**
変えると全 Kyou の ID が作り直され、gkill 側で付けたタグやテキストの紐付けが全部切れる。

「追記のみ」の前提が破れた（過去の行が消された）疑いは、
ファイルサイズの減少・発言数の減少・最初の発言の時刻の変化で検出し、設定画面に警告を出す。
自動修復はしない。

### 巨大な1行

実データの最大は**1行 数十MB**（ツールの出力）で、1MBを超える行が数十行ある。
`bufio.Scanner` はもちろん、`ReadString('\n')` でも捨てるだけの数十MBを文字列に起こしてしまう。

`reader.go` は行の先頭512バイトだけを覗いて種別を決め、要らない行は改行まで**溜めずに読み捨てる**。
`event_msg/item_completed` は会話もツールの実行結果も同じ種別に載るので、`item.type` と `item.kind` まで先頭から読む
（実データでは必ず先頭512バイト以内にある）。
保持する行だけを2MiBまで積む。判定できなかった行は「捨てる」ではなく「拾う」――
キー順や種別名が変わった日に会話が静かに消えるのを防ぐため。

### 取り込みはバックグラウンド

gkill のハンドラに許されているのは実質**数十ミリ秒**（`IsAlive` の期限は5秒、
一覧は行数ぶんの `get_content_html` が1本のスロットに並ぶ）。
そこで常駐 goroutine が走査と取り込みを回し、ハンドラは「今キャッシュにあるぶん」を即座に返す。

**プロセス起動後の1回目の検索は空が返る。これは仕様。** 進捗は設定画面に出る。

ロックは3つに分けてある。`mu` は DB の遅延初期化だけ、`buildMu` は構築どうしだけ、
**読み取りはどちらも取らない**。兼用すると初回構築のあいだ検索が全部詰まり、
gkill のデッドラインでプロセスが殺され続ける。

### キャッシュは2段構成

差分の単位は「パースはファイル、畳み直しはスレッド木」。

```
file_cache   : 走査結果。差分判定は (mtime, size)
thread_item  : 1ファイルを正規化した結果。ファイル単位で差し替える
dirty_thread : 畳み直しが要るスレッド
thread_title : session_index.jsonl 由来のスレッド名
kyou_cache   : Kyou 1件。body_json に詳細ビュー用の本体が入る
```

- スレッド名は `search_text` に焼き込まない。`session_index.jsonl` は名前が付くたび書き換わるので、
  焼き込むと毎回すべてのスレッドを畳み直すことになる
- 子が新しく現れたときも、親が消えたときも、**両方のルートを畳み直し対象にする**
- 検索用テキストは1件512KBで打ち切る（実データには畳み込みで数MBに達する Kyou がある）

実測（数十ファイル / 数百MB）: **フル構築 数秒**、差分は1秒未満、
`cache.db` は数十MB、Kyou 数百件。

キャッシュは `$GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db` に置かれ、
`gkill_server clear_cache plugin <all|user_id...>` で消せる。
**パーサを直したときは `cache.go` の `cacheSchemaVersion` を上げること** ―― 差分更新は変わっていないファイルを
二度と読まないので、そのままでは直した解析が過去に遡らない。世代が変わると次の起動で全テーブルを作り直す
（そのあいだ検索は空になる）。

## プラグイン情報

| 項目 | 値 |
|---|---|
| `name` / `executable` | `gkill_plugin_codex` |
| `rep_name` | `Codex` |
| `data_type` | `codex_turn` |
| `protocol_version` | `1` |
| `min_gkill_version` | `1.1.11` |
| `provides` | 宣言しない（タグ・型別データは出さない） |

タグは付けない。gkill 1.1.7 以降は `manifest.json` に `"provides": ["tag"]` を書けば
プラグインのタグもタグ一覧（`get_all_tag_names`）に載るが、同梱の `gkill_plugin_claudecode` と
揃えて宣言していない。プロジェクト名・ブランチ・モデル名は検索テキストに入れてあるので
ワード検索で引ける。

## ファイル構成

| ファイル | 役割 |
|---|---|
| `main.go` | print系フラグの先行処理、`sdk.Run`、5つのハンドラ |
| `types.go` | ワイヤ構造体とドメイン型、定数 |
| `reader.go` | 巨大行に耐えるレコードリーダ |
| `scan.go` | ソース展開・ファイル分類・`session_index.jsonl` 読み |
| `loader.go` | ファイル → 正規化した要素列（第1段） |
| `fold.go` | 要素列 → Kyou、サブエージェントの畳み込み、Kyou ID |
| `cache.go` | スキーマ、取り込み、畳み直し |
| `query.go` | 読み取り（ロックを取らない） |
| `builder.go` | 常駐バックグラウンドビルダ |
| `config.go` | 設定の解釈と既定値 |
| `render.go` | 詳細ビューのHTML（最終回答を本文にし、経過を畳む） |
| `html.go` | 設定画面のHTML |
| `uuid.go` | UUIDv5 |

## テスト

```bash
cd src/plugins/gkill_plugin_codex
go test ./...
```

`npm run test_plugins`（`npm test` に含まれる）は `src/plugins/` 配下の `go.mod` を
自動で見つけるので、登録作業は要らない。

フィクスチャ（`testdata/`）は合成データ。実ログは絶対パス・リポジトリURL・コミットハッシュ・
実際のプロンプトを含むので入れない。巨大行のフィクスチャはテスト内で一時ディレクトリに生成する。

とくに落としてはいけないテスト:

- `TestParseRolloutTakesIdentityFromFirstMetaOnly` — サブエージェントの2つ目の `session_meta` で
  自分が親にすり替わらないこと
- `TestParseRolloutReadsItemCompletedLane` — 0.151 以降の形（会話が `item_completed` にだけ載る）を読めること
- `TestParseRolloutDropsItemLaneInLegacyTurns` — 両方のレーンが同じターンに並んでも会話が二重にならないこと
- `TestKyouIDIsStableAcrossAppend` — 追記しても既存の Kyou ID が動かないこと
- `TestKyouIDIsDeterministic` — 名前空間を固定する
- `TestClassifyHeadUnknownPayloadIsKept` — 判定できない行を捨てないこと
- `TestLineReaderDrainsGiantSkippedLine` — 3MB の行を溜めずに捨て、次の行が読めること
- `TestConcurrentReadDuringBuild` — 構築中に検索が詰まらないこと
- `TestSessionIndexChangeDoesNotReparseRollouts` — スレッド名の更新でフル再構築しないこと

## 関連資料

- [プラグイン一覧](../README.md)
- [プラグイン機構の設計](../../../documents/reverse/plugin-system.md)
- [SDK](../../server/gkill/plugin/README.md)
