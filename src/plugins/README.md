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
│       └── manifest.json
├── gkill_plugin_claudeai/       # Claude.ai チャット履歴プラグイン
│   ├── main.go
│   ├── loader.go
│   ├── cache.go
│   ├── types.go
│   ├── html.go
│   ├── go.mod
│   └── manifest.json
└── gkill_plugin_chatgpt/        # ChatGPT チャット履歴プラグイン
    ├── main.go
    ├── loader.go
    ├── cache.go
    ├── types.go
    ├── html.go
    ├── go.mod
    └── manifest.json
```

## 提供プラグイン

| プラグイン | data_type | 概要 |
|---|---|---|
| [`gkill_example`](examples/gkill_example/README.md) | `example_kyou` | サンプル実装。固定Kyouを2件返す |
| [`gkill_plugin_claudeai`](gkill_plugin_claudeai/README.md) | `claude_conversation` | Claude.ai のチャット履歴をタイムライン表示 |
| [`gkill_plugin_chatgpt`](gkill_plugin_chatgpt/README.md) | `chatgpt_conversation` | ChatGPT のチャット履歴をタイムライン表示 |

---

## プラグインの配置・セットアップ

### 配置先

```
$GKILL_HOME/plugins/{userID}/{プラグイン名}/
  manifest.json
  {実行ファイル}          # Linux/macOS: gkill_plugin_xxx
                          # Windows:      gkill_plugin_xxx.exe
  conversations.json      # データファイル（必要なプラグインのみ）
```

### gkill_plugin_claudeai

1. Claude.ai の設定ページからデータをエクスポートし `conversations.json` を取得する
2. 配置先ディレクトリに `manifest.json`・実行ファイル・`conversations.json` を置く
3. gkill_server を再起動する（または gkill 設定画面でプラグインをリロードする）

### gkill_plugin_chatgpt

1. ChatGPT の設定ページからデータをエクスポートし ZIP を解凍する
2. 配置先ディレクトリに `manifest.json`・実行ファイルと、次のどちらかを置く:
   - 新形式: `conversations-000.json`, `conversations-001.json`, ... （複数ファイル）
   - 旧形式: `conversations.json`（単一ファイル）
3. gkill_server を再起動する

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
  "min_gkill_version": "1.1.3"
}
```

| フィールド | 説明 |
|---|---|
| `protocol_version` | プロトコルバージョン（現在は `"1"` 固定） |
| `name` | プラグイン識別子（ディレクトリ名と合わせる） |
| `data_type` | このプラグインが返す Kyou の `data_type` 値 |
| `rep_name` | gkill UI 上のリポジトリ表示名 |
| `executable` | 実行ファイル名（拡張子なし。Windows では `.exe` を自動補完） |
| `min_gkill_version` | 動作に必要な最低 gkill バージョン |

---

## プラグインプロトコル（stdio 改行区切り JSON）

gkill_server はプラグインをサブプロセスとして起動し、stdin/stdout で 1 行 = 1 JSON メッセージを送受信する。

### 起動引数

```
{executable} --gkill-plugin-dir {pluginDir} --gkill-user-id {userID} --gkill-protocol-version 1
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
```

---

## 関連ドキュメント

- [server/gkill/plugin/sdk/](../server/gkill/plugin/sdk/) — プラグイン SDK ソース
- [server/gkill/api/gkill_plugin/](../server/gkill/api/gkill_plugin/) — プラグインプロトコル型定義
- [server/gkill/dao/reps/plugin_repository_impl.go](../server/gkill/dao/reps/plugin_repository_impl.go) — プラグインサブプロセス管理
