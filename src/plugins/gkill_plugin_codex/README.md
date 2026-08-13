# gkill_plugin_codex

OpenAI Codex CLI のセッションログ（ロールアウトJSONL）を gkill のタイムラインに表示するプラグイン。

**人間の発言1つ**と、**それに続く一連の応答**を、それぞれ1つの Kyou として扱う。
ツールの実行結果は保持しない（実データではログのバイトの94.7%がこれで、日記としては要らない）。
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
`gkill_plugin_` の接頭辞も必須（Termux 側の配布スクリプトが `pkill -KILL -f gkill_plugin_` で
更新前にプロセスを落としているため、接頭辞が無いと古いバイナリを掴んだまま生き残る）。

`manifest.json` と既定の `config.json` はバイナリ自身から出せる。

```bash
./gkill_plugin_codex --gkill-print-manifest > manifest.json
./gkill_plugin_codex --gkill-print-config   > config.json
```

> **PowerShell 5.1 では `>` を使わないこと。** UTF-16LE で書かれ、さらにプラグインの UTF-8 出力を
> CP932 として解釈するので二重に壊れる。壊れた `manifest.json` は
> `plugin_manager.go` が**無言で読み飛ばす**ので、プラグインが消えたようにしか見えない。
> `scripts/UpdateGkillPlugins.ps1` はこれを避けるため `StandardOutputEncoding` と
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

| ログのレコード | Kyou での扱い |
|---|---|
| `event_msg/user_message` | 人間の発言 = 1 Kyou。IDE の前置きは剥がして折りたたみへ |
| `event_msg/agent_message` | 応答の本文 |
| `event_msg/agent_reasoning` | 思考（折りたたみ） |
| `response_item/function_call` | ツール呼び出し（引数を200文字に要約） |
| `response_item/custom_tool_call` | 同上。`apply_patch` は対象ファイル一覧に潰す |
| `response_item/web_search_call` / `tool_search_call` | ツール呼び出し（検索語） |
| `event_msg/patch_apply_end` | 変更したファイル一覧（パス・種別・増減行数） |
| `event_msg/item_completed`（`Plan`） | 計画 |
| `event_msg/task_complete` | 所要時間のチップ |
| `event_msg/turn_aborted` / `context_compacted` | 短い通知 |
| `event_msg/sub_agent_activity` | サブエージェントの親子リンク |
| `session_meta` / `turn_context` | プロジェクト名・ブランチ・モデル名などのメタ情報 |

**保持しないもの**: ツールの実行結果（`*_output` / `mcp_tool_call_end`）、
暗号化された思考（`response_item/reasoning`）、トークン数、
`response_item/message`（`event_msg` と二重になるうえ、環境プリアンブルの注入が混ざる）、
`compacted`（圧縮前の履歴を抱えていて、拾うと会話が二重になる）。

### 会話は `event_msg` レーンだけを使う

Codex のログは同じ発言が2つのレーンに出る。実データ全17バージョン（0.104〜0.147）で確かめた結果、

- `event_msg/user_message` が**実際に人間が打った入力**。全バージョンに存在する
- `response_item/message` role=user には `<recommended_plugins>` などの注入が混ざるので使えない

ツール呼び出しだけは `response_item` にしか無いので、そちらから拾う。

### IDE の前置き

VSCode 拡張から使うと、発言の先頭に開いているタブの一覧が付く（実データでは178件中108件）。

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

## 動作の要点

### スレッドIDはファイル名の uuid

`session_meta.session_id` は使えない。実データ52ファイルのうち**23ファイルに存在せず**、
存在しても**サブエージェントでは親のIDが入っている**（52ファイルに対し `session_id` は38種しかない）。
これをキーにすると親子の Kyou ID が衝突する。

`session_meta.id` はファイル名の uuid と**52/52で一致**するので、そちらを正とする。

### `session_meta` は identity と environment で扱いを分ける

1ファイルに1〜13回出る（resume のたびに書かれる）。
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

実データの最大は**1行 19,912,604 バイト**（ツールの出力）で、1MBを超える行が77行ある。
`bufio.Scanner` はもちろん、`ReadString('\n')` でも捨てるだけの19.9MBを文字列に起こしてしまう。

`reader.go` は行の先頭512バイトだけを覗いて種別を決め、要らない行は改行まで**溜めずに読み捨てる**。
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
- 検索用テキストは1件512KBで打ち切る（実データには畳み込みで5.4MBに達する Kyou が1件ある）

実測（52ファイル / 245MB / 2026-08）: **フル構築 4.5秒**、差分は1秒未満、
`cache.db` は約19MB、Kyou 301件。

キャッシュは `$GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db` に置かれ、
`gkill_server clear_cache plugin <all|user_id...>` で消せる。
**パーサを直したときは消すこと** ―― 差分更新は変わっていないファイルを二度と読まないので、
直した解析が過去に遡らない。

## プラグイン情報

| 項目 | 値 |
|---|---|
| `name` / `executable` | `gkill_plugin_codex` |
| `rep_name` | `Codex` |
| `data_type` | `codex_turn` |
| `protocol_version` | `1` |
| `min_gkill_version` | `1.1.7` |
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
| `render.go` | 詳細ビューのHTML |
| `html.go` | 設定画面のHTML |
| `cache_path.go` | キャッシュDBの置き場所の解決 |
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
