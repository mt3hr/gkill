# plugins - 外部プラグイン

## 概要

gkill のプラグインシステム。外部データソース（AIチャット履歴など）を Kyou としてタイムラインに表示する。
プラグインは独立した Go バイナリとして実装し、gkill_server と **stdio 改行区切り JSON** で通信する。

## ディレクトリ構造

```
plugins/
├── examples/
│   └── gkill_example/          # サンプルプラグイン（固定Kyouを返す）
│       ├── main.go
│       ├── go.mod
│       ├── manifest.json
│       └── README.md
├── gkill_plugin_claudeai/       # Claude.ai チャット履歴プラグイン
│   ├── main.go
│   ├── loader.go
│   ├── source.go
│   ├── cache.go
│   ├── types.go
│   ├── html.go
│   ├── go.mod / go.sum
│   ├── manifest.json
│   └── README.md
├── gkill_plugin_chatgpt/        # ChatGPT チャット履歴プラグイン
│   ├── main.go
│   ├── loader.go
│   ├── source.go
│   ├── cache.go
│   ├── types.go
│   ├── html.go
│   ├── go.mod / go.sum
│   ├── manifest.json
│   └── README.md
├── gkill_plugin_fitbit/        # Fitbit / Google Health 日別集計プラグイン
│   ├── main.go
│   ├── config.go
│   ├── metrics.go
│   ├── loader.go
│   ├── timeparse.go
│   ├── builder.go
│   ├── cache.go
│   ├── query.go
│   ├── render.go
│   ├── html.go
│   ├── uuid.go
│   ├── types.go
│   ├── testdata/
│   ├── go.mod / go.sum
│   ├── manifest.json
│   └── README.md
├── gkill_plugin_google_locationhistory/  # Google ロケーション履歴プラグイン
│   ├── main.go
│   ├── formats.go
│   ├── parsers.go
│   ├── source.go
│   ├── cache.go
│   ├── html.go
│   ├── types.go
│   ├── testdata/
│   ├── go.mod / go.sum
│   ├── manifest.json
│   └── README.md
├── gkill_plugin_claudecode/     # Claude Code チャットログプラグイン
│   ├── main.go
│   ├── loader.go
│   ├── cache.go
│   ├── types.go
│   ├── render.go
│   ├── html.go
│   ├── loader_test.go
│   ├── testdata/
│   ├── go.mod / go.sum
│   ├── manifest.json
│   └── README.md
└── gkill_plugin_codex/          # Codex CLI セッションログプラグイン
    ├── main.go
    ├── types.go
    ├── reader.go               # 巨大行(実データ最大19.9MB)に耐えるレコードリーダ
    ├── scan.go
    ├── loader.go               # ファイル → 正規化した要素列
    ├── fold.go                 # 要素列 → Kyou、サブエージェントの畳み込み
    ├── cache.go
    ├── query.go                # 読み取り(ロックを取らない)
    ├── builder.go              # 常駐バックグラウンドビルダ
    ├── config.go
    ├── render.go
    ├── html.go
    ├── uuid.go
    ├── *_test.go
    ├── testdata/
    ├── go.mod / go.sum
    ├── manifest.json
    └── README.md
```

## 提供プラグイン

| プラグイン | data_type | 概要 |
|---|---|---|
| [`gkill_example`](examples/gkill_example/README.md) | `example_kyou` | サンプル実装。固定Kyouを2件返す |
| [`gkill_plugin_claudeai`](gkill_plugin_claudeai/README.md) | `claude_conversation` | Claude.ai のチャット履歴をタイムライン表示 |
| [`gkill_plugin_chatgpt`](gkill_plugin_chatgpt/README.md) | `chatgpt_conversation` | ChatGPT のチャット履歴をタイムライン表示 |
| [`gkill_plugin_claudecode`](gkill_plugin_claudecode/README.md) | `claude_code_turn` | Claude Code のチャットログを、自分の発言と一連の応答に分けてタイムライン表示 |
| [`gkill_plugin_codex`](gkill_plugin_codex/README.md) | `codex_turn` | Codex CLI のセッションログを、自分の発言と一連の応答に分けてタイムライン表示（サブエージェントは親に畳み込む） |
| [`gkill_plugin_fitbit`](gkill_plugin_fitbit/README.md) | `kc` | Google Takeout の Fitbit / Google Health を日別集計し、数値記録として返す（推移グラフで集計できる） |
| [`gkill_plugin_google_locationhistory`](gkill_plugin_google_locationhistory/README.md) | `google_location_visit` | Google Takeout のロケーション履歴を位置情報ログとして読み込む（記録は作らない） |

---

## プラグインの配置・セットアップ

### 配置先

```
$GKILL_HOME/plugins/{userID}/{プラグイン名}/
  manifest.json
  {実行ファイル}          # Linux/macOS: gkill_plugin_xxx
                          # Windows:      gkill_plugin_xxx.exe
  config.json             # データソース設定（初回起動時に自動生成される）
  conversations.json      # データファイル（プラグインフォルダに直接置く場合のみ）
```

### gkill_plugin_claudeai

1. Claude.ai の設定ページからデータをエクスポートし `conversations.json` を取得する
2. 配置先ディレクトリに `manifest.json`・実行ファイルを置き、gkill_server を再起動する
3. 同じディレクトリに `config.json` が自動生成されるので、`source_dirs` に
   `conversations.json` を置いたフォルダ（またはファイル）を書く

`source_dirs` を空のままにすると、従来どおりプラグインフォルダ自身を見る。

### gkill_plugin_chatgpt

1. ChatGPT の設定ページからデータをエクスポートし ZIP を解凍する
   - 新形式: `conversations-000.json`, `conversations-001.json`, ... （複数ファイル）
   - 旧形式: `conversations.json`（単一ファイル）
2. 配置先ディレクトリに `manifest.json`・実行ファイルを置き、gkill_server を再起動する
3. 自動生成された `config.json` の `source_dirs` に、解凍先フォルダを書く

こちらも `source_dirs` が空ならプラグインフォルダ自身を見る。

### gkill_plugin_claudecode

データファイルの配置は不要。読み込むフォルダは `config.json` で指定する。

1. 配置先ディレクトリに `manifest.json`・実行ファイルを置き、gkill_server を再起動する
2. 自動生成される `config.json` の `source_dirs` は既定で `~/.claude/projects`。
   他の場所を読ませたい場合は書き換える

### gkill_plugin_codex

データファイルの配置は不要。読み込むフォルダは `config.json` で指定する。

1. 配置先ディレクトリに `manifest.json`・実行ファイルを置き、gkill_server を再起動する
2. 自動生成される `config.json` の `source_dirs` は既定で `~/.codex/sessions` と
   `~/.codex/session_index.jsonl`。他の場所を読ませたい場合は書き換える
3. 取り込みはバックグラウンドで進むので、**置いた直後の1回目の検索は空が返る**。
   進捗は設定画面に出る（実測でフル構築4.5秒 / 245MB / 52ファイル）

`session_index.jsonl` はセッションuuid → スレッド名の対応表で、指定すると詳細画面に
スレッド名が出る。無くても取り込みはできる。

### gkill_plugin_fitbit / gkill_plugin_google_locationhistory

Google Takeout を読む2つ。**ZIP を解凍せず、そのままフォルダに置く。**

```
~/Kyou/GoogleTakeout_<端末>_<日付>/
  takeout-20260808T230152Z-1-001.zip
  takeout-20260808T230152Z-1-002.zip   ← 分割されていればそのまま並べる
```

1. Google Takeout から Fitbit / Google Health / タイムライン を書き出す
2. ダウンロードした ZIP を上のようなフォルダに置く（解凍しない）
3. 配置先ディレクトリに `manifest.json`・実行ファイルを置き、gkill_server を再起動する
4. 自動生成される `config.json` の `source_dirs` は既定で `~/Kyou/GoogleTakeout_*`

**展開済みのフォルダは読まない**（どの書き出しのものか判別できないため）。
走査は `plugin/sdk/source.go` の `sdk.OpenSources` で、2つのプラグインで共通。

新しい書き出しは**別のフォルダ**に置く。fitbit は日が重なったところで新しい書き出しだけを
採用するので、古い書き出しを消さなくても歩数などが二重にならない。
位置情報のほうは読み出し時に同じ点を1つに畳むので、そもそも重ならない
（むしろ Google は古いデータを間引くので、古い書き出しを残すと消えた期間が保たれる）。

---

## config.json（データソース設定）

`manifest.json` と同じフォルダに置く設定ファイル。プラグインの初回起動時に既定値で
**自動生成される**ので、生成されたファイルを手で編集して使う。既存ファイルは上書きされない。

```json
{
  "_comment": "書式の説明（読み飛ばされるので消してよい）",
  "_example_source_dirs": ["~/Kyou/ClaudeAIExport", "D:/Dropbox/claude_export/**/conversations*.json"],
  "source_dirs": []
}
```

| キー | 説明 |
|---|---|
| `source_dirs` | データソースの指定。配列で複数指定できる（1つなら文字列でも可） |
| `_` 始まりのキー | 書式の説明用。プラグインは読まないので消してよい |

`source_dirs` の書式:

- **フォルダ** を指定すると再帰的に走査し、そのプラグインが読むファイル名（`conversations.json`、
  `*.jsonl` など）を探す
- **ファイル** を直接指定すると、名前が規則に合わなくてもそのまま読む
- ワイルドカード `*` `**` `?` `[]` が使える（[go-zglob](https://github.com/mattn/go-zglob)）
- 先頭の `~` と環境変数（`$HOME` など）を展開する。ただし gkill を Windows サービスで動かして
  いる場合は**実行アカウントのホーム**になるため、絶対パスのほうが確実
- 空にすると各プラグインの既定（claudeai / chatgpt はプラグインフォルダ自身、
  claudecode は `~/.claude/projects`、codex は `~/.codex/sessions` と
  `~/.codex/session_index.jsonl`、fitbit / 位置情報は `~/Kyou/GoogleTakeout_*`）を使う
- 編集は**次の検索から反映される**（gkill_server の再起動は不要）
- **fitbit と位置情報だけは ZIP しか読まない。** フォルダを指定するとその下の `*.zip` を
  再帰的に探し、ZIP の中を走査する。ZIP を直接指定してもよい

---

## manifest.json

各プラグインのルートに置く定義ファイル。

```json
{
  "protocol_version": "1",
  "name": "gkill_plugin_xxx",
  "version": "1.0.0",
  "description": "プラグインの説明",
  "data_type": "xxx_data_type",
  "rep_name": "表示名",
  "executable": "gkill_plugin_xxx",
  "min_gkill_version": "1.1.7"
}
```

| フィールド | 説明 |
|---|---|
| `protocol_version` | プロトコルバージョン（現在は `"1"` 固定） |
| `name` | プラグイン識別子（ディレクトリ名・`executable` と合わせる） |
| `data_type` | このプラグインが返す Kyou の `data_type` 値 |
| `rep_name` | gkill UI 上のリポジトリ表示名 |
| `executable` | 実行ファイル名（拡張子なし。Windows では `.exe` を自動補完） |
| `min_gkill_version` | 動作に必要な最低 gkill バージョン |

**名前は `gkill_plugin_<名前>` にすること。** ディレクトリ名・`name`・`executable`・
配置先のフォルダ名がすべて同じ文字列である必要がある（配布スクリプトがこの1つの名前から
ソース・ビルド出力・配置先を組み立てるため）。

接頭辞まで規約なのは、Termux 側の配布スクリプト
（`termux-tasker/update_gkill_plugins.sh`）が更新前に

```bash
pkill -KILL -f gkill_plugin_
```

で起動中のプラグインを落としているから。この接頭辞を持たない名前だとプロセスが落ちず、
古いバイナリを掴んだまま生き残る。かつて `gkill_google_locationhistory_plugin` という
名前で作ってしまい、あとから改名した。

---

## プラグインプロトコル（stdio 改行区切り JSON）

gkill_server はプラグインをサブプロセスとして起動し、stdin/stdout で 1 行 = 1 JSON メッセージを送受信する。

### 起動引数

```
{executable} --gkill-plugin-dir {pluginDir} --gkill-user-id {userID} --gkill-protocol-version 1
```

### manifest.json の出力

提供プラグインは `manifest.json` を `//go:embed` でバイナリに埋め込んでおり、
`--gkill-print-manifest` で標準出力に吐いて終了する（stdio ループには入らない）。
配布スクリプトがバイナリだけを配り、配置先で `manifest.json` を作れるようにするため。
バイナリと manifest が必ず一致する。

```bash
./gkill_plugin_xxx --gkill-print-manifest > manifest.json
```

### config.json の出力

同様に `--gkill-print-config` で既定の `config.json` を標準出力に吐いて終了する。
通常はプラグインの起動時に自動生成されるので、配置スクリプトで先に用意したいとき以外は不要。

```bash
./gkill_plugin_xxx --gkill-print-config > config.json
```

### コマンド一覧

| コマンド | 方向 | 説明 |
|---|---|---|
| `ping` | gkill → plugin | 死活確認。`pong: true` を返す |
| `close` | gkill → plugin | プロセス終了 |
| `get_rep_name` | gkill → plugin | `rep_name` を返す |
| `find_kyous` | gkill → plugin | 検索条件付きで Kyou 一覧を返す |
| `get_kyou` | gkill → plugin | ID 指定で 1 件返す |
| `get_content_html` | gkill → plugin | Kyou 詳細表示用 HTML を返す |
| `get_config_html` | gkill → plugin | 設定フォーム HTML を返す |
| `post_config` | gkill → plugin | フォームデータを受けて設定を保存する |

### リクエスト形式

```json
{
  "id": "req-001",
  "command": "find_kyous",
  "query": {
    "words": ["Claude"],
    "not_words": [],
    "words_and": false,
    "tags": [],
    "not_tags": [],
    "tags_and": false,
    "calendar_start_date": "2026-01-01T00:00:00Z",
    "calendar_end_date": "2026-06-01T00:00:00Z",
    "is_deleted": false,
    "only_latest_data": false,
    "limit": 100
  }
}
```

### レスポンス形式

```json
{
  "id": "req-001",
  "kyous": [ ... ],
  "errors": []
}
```

---

## プラグイン SDK（`gkill/plugin/sdk`）

プラグイン作者向けに Go SDK を提供。`sdk.Run()` を呼ぶだけでプロトコルのループを自動処理する。

```go
package main

import (
    "context"
    sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

func main() {
    sdk.Run(sdk.Handler{
        RepName: "MyPlugin",

        FindKyous: func(ctx context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
            // 外部データを取得して Kyou スライスを返す
            return []sdk.Kyou{ ... }, nil
        },

        GetContentHTML: func(ctx context.Context, kyouID string, cfg sdk.Config) (string, error) {
            // Kyou 詳細の HTML 文字列を返す
            return "<html>...</html>", nil
        },

        GetConfigHTML: func(ctx context.Context, cfg sdk.Config) (string, error) {
            // 設定フォームの HTML を返す（不要なら省略可）
            return "<html>...</html>", nil
        },
    })
}
```

### `sdk.Query` フィールド

| フィールド | 型 | 説明 |
|---|---|---|
| `Words` | `[]string` | 含むワード |
| `NotWords` | `[]string` | 除外ワード |
| `WordsAnd` | `bool` | `true` = AND 検索 |
| `Tags` | `[]string` | 含むタグ |
| `NotTags` | `[]string` | 除外タグ |
| `TagsAnd` | `bool` | `true` = AND 検索 |
| `CalendarStartDate` | `*time.Time` | 期間開始（nil = 無制限） |
| `CalendarEndDate` | `*time.Time` | 期間終了（nil = 無制限） |
| `IsDeleted` | `bool` | 削除済みのみ取得 |
| `OnlyLatestData` | `bool` | 最新データのみ |
| `Limit` | `int` | 最大件数（0 = 無制限） |

### `sdk.Kyou` フィールド

| フィールド | 型 | 説明 |
|---|---|---|
| `ID` | `string` | 一意な識別子 |
| `RepName` | `string` | リポジトリ名（manifest の `rep_name` と合わせる） |
| `DataType` | `string` | manifest の `data_type` と合わせる |
| `RelatedTime` | `time.Time` | タイムライン上の時刻 |
| `CreateTime` | `time.Time` | 作成時刻 |
| `UpdateTime` | `time.Time` | 更新時刻 |
| `CreateApp` | `string` | 作成アプリ名 |
| `CreateUser` | `string` | 作成ユーザ名 |
| `UpdateApp` | `string` | 更新アプリ名 |
| `UpdateUser` | `string` | 更新ユーザ名 |
| `Tags` | `[]string` | タグ一覧（省略可） |
| `Texts` | `[]string` | テキスト注釈一覧（省略可） |
| `ImageSource` | `string` | サムネイル画像 URL（省略可） |

### `sdk.Config`

`map[string]any` のエイリアス。プラグインディレクトリの `config.json` に永続化される。

```go
// 設定値取得（デフォルト値付き）
msg := cfg.Get("message", "Hello!")
```

`Handler.DefaultConfig` を設定しておくと、`config.json` が無いときに
その内容で自動生成される（既存ファイルは上書きしない）。nil なら何も生成しない。

```go
sdk.Run(sdk.Handler{
    DefaultConfig: sdk.Config{"source_dirs": []string{}},
    // ...
})
```

---

## ビルド

各プラグインは独立した Go モジュールを持つ。

```bash
# gkill_example
cd src/plugins/examples/gkill_example
go build -o gkill_example .

# gkill_plugin_claudeai
cd src/plugins/gkill_plugin_claudeai
go build -o gkill_plugin_claudeai .

# gkill_plugin_chatgpt
cd src/plugins/gkill_plugin_chatgpt
go build -o gkill_plugin_chatgpt .

# gkill_plugin_claudecode
cd src/plugins/gkill_plugin_claudecode
go build -o gkill_plugin_claudecode .

# gkill_plugin_codex
cd src/plugins/gkill_plugin_codex
go build -o gkill_plugin_codex .

# gkill_plugin_fitbit
cd src/plugins/gkill_plugin_fitbit
go build -o gkill_plugin_fitbit .

# gkill_plugin_google_locationhistory
cd src/plugins/gkill_plugin_google_locationhistory
go build -o gkill_plugin_google_locationhistory .
```

---

## 関連ドキュメント

- [server/gkill/plugin/sdk/](../server/gkill/plugin/sdk/) — プラグイン SDK ソース
- [server/gkill/api/gkill_plugin/](../server/gkill/api/gkill_plugin/) — プラグインプロトコル型定義
- [server/gkill/dao/reps/plugin_repository_impl.go](../server/gkill/dao/reps/plugin_repository_impl.go) — プラグインサブプロセス管理
