# gkill_plugin_claudecode

Claude Code のチャットログ（セッション JSONL）を gkill タイムラインに表示するプラグイン。
**1メッセージ = 1 Kyou** として扱う。人間の発言と Claude のテキスト応答がそれぞれ別の Kyou になり、
ChatGPT / Claude.ai プラグインと粒度が揃う。

ツール実行と thinking は独立した Kyou にはせず、**直近の Claude の発言に折りたたんで**付ける
（1セッションでツール実行が数百回に及ぶため）。

## セットアップ

### 1. ビルド

```bash
cd src/plugins/gkill_plugin_claudecode
go build -o gkill_plugin_claudecode .
# Windows の場合
go build -o gkill_plugin_claudecode.exe .
```

### 2. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_claudecode/
├── manifest.json             # このディレクトリの manifest.json をコピー
├── gkill_plugin_claudecode   # ビルドしたバイナリ（.exe は自動補完）
├── config.json               # データソースのフォルダ指定（任意。次項参照）
└── cache.db                  # 自動生成されるキャッシュ
```

`manifest.json` はバイナリに埋め込まれているので、配置先で吐かせることもできる。
配布スクリプト（`scripts/UpdateGkillPlugins.ps1` / `termux-tasker/update_gkill_plugins.sh`）は
バイナリだけを配り、`manifest.json` が無ければこれで生成する。バイナリと必ず一致する。

```bash
./gkill_plugin_claudecode --gkill-print-manifest > manifest.json
```

### 3. データソースのフォルダを指定する

プラグインフォルダに `config.json` を置く。未設定なら `~/.claude/projects` を使うので、
既定の場所で足りるなら置かなくてよい。

```json
{
  "source_dirs": [
    "C:\\Users\\user\\.claude\\projects",
    "~/DevPC/ClaudeCode_*"
  ]
}
```

**複数指定**は配列で書く（1つなら文字列でも可。文字列を改行 `\n` で区切ってもよい）。

**ワイルドカード**が使える。マッチしたフォルダは再帰的に走査し、マッチしたファイルはそのまま
対象にする。

| 書き方 | 意味 |
|---|---|
| `C:\Users\user\.claude\projects` | そのフォルダを再帰的に走査 |
| `~/.claude/projects/C--Users-user-Git-*` | 特定のプロジェクトだけを選ぶ |
| `~/DevPC/ClaudeCode_*` | dvnf の日付付きフォルダをパターンで拾う |
| `C:\logs\**\*.jsonl` | `**` で階層をまたいで jsonl だけ拾う |

先頭の `~` と環境変数（`$HOME` など）も展開される。同じセッションが複数の指定に含まれても、
Kyou の ID が UUID なので重複は自然に統合される。

> **Windows サービスで動かしている場合の注意**
> `GkillServer` は LocalSystem で動くことがある。Go の `os.UserHomeDir()` は Windows では
> `%USERPROFILE%` を見るため、`~` や既定値は実行アカウントのホーム
> （LocalSystem なら `C:\Windows\system32\config\systemprofile`）を指してしまう。
> サービス配下では**絶対パスで書くのが確実**。

**編集は次の検索から反映される**（gkill の再起動は不要）。SDK は `config.json` をプロセス起動時に
一度しか読まないため、このプラグインは毎回 `sdk.LoadConfig` で読み直している。

現在の設定・パターンの展開結果・読み込み状況（対象ファイル数・発言数・最終スキャン時刻）は、
gkill のプラグイン設定画面（`get_config_html`）で確認できる。何にもマッチしなかった指定もそこに出る。
**設定画面は表示のみで保存はできない** — gkill 側に保存を呼ぶ導線が無いため。`config.json` を直接編集する。

ファイル種別はパスではなく中身で判定する。そのため次のどちらの配置でも読める。

- Claude Code が書く入れ子の配置: `~/.claude/projects/<プロジェクト>/<セッションID>.jsonl`
- 平置きにコピーした配置: 同じフォルダに `<セッションID>.jsonl` と `agent-<ID>.jsonl` が並ぶ形

## 取り込む内容

| ファイル | 用途 |
|---|---|
| `<セッションID>.jsonl` | メインのトランスクリプト。ここから発言を切り出す |
| `agent-<ID>.jsonl` | サブエージェントの会話。Agent ツールを実行した発言に折りたたんで表示 |
| `agent-<ID>.meta.json` | サブエージェントの種別・説明・`toolUseId`。親との紐付けに使う |
| `history.jsonl` | 対象外（プロンプト履歴のみで会話の文脈がない） |

Kyou になるのは次の2種類。

| 発言 | 内容 |
|---|---|
| 人間の発言 | プロンプト本文。青背景で「あなた」と表示 |
| Claude の発言 | テキスト応答。グレー背景で「Claude」と表示 |

Claude の発言には次が折りたたみで付く。**その発言の前後に現れたもの**をまとめる
（実データでは `thinking → text → tool_use` の順が多いため、テキストの前に来た thinking も拾う）。

- ツール実行（ツール名と**入力の1行要約**のみ。実行結果は保持しない）
- thinking
- サブエージェントの会話
- `task-notification` などシステム発の入力（1行の注記）

## 動作の要点

- **Kyou の ID** は各レコードの `uuid`。同じセッションが複数フォルダにあっても重複しない
- **タグは付けない**。gkill のタグ一覧（`get_all_tag_names`）にはプラグインが返したタグが載らず、
  タグを付けると rykv の既定の絞り込み「no tags」から漏れて何も表示されなくなるため。
  プロジェクト名とブランチ名は詳細HTMLのチップで表示し、ワード検索の対象にも含めている
- **キャッシュ** は `cache.db`（SQLite3）。ファイル単位で mtime とサイズを見て、
  変化のあったセッションだけ作り直す。146MB 規模のソース（3389 発言）で初回12秒、以降は1秒未満。
  スキーマを変えたときは `cache_meta` の `schema_version` を見て自動で作り直す
- **長い行**への対応: トランスクリプトには100万文字を超える行が実在するため、
  `bufio.Scanner` ではなく `bufio.Reader.ReadString` で読む

## プラグイン情報

| 項目 | 値 |
|---|---|
| `rep_name` | `ClaudeCode` |
| `data_type` | `claude_code_turn` |
| プロトコルバージョン | `1` |
| 最小 gkill バージョン | `1.1.3` |

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | エントリポイント、SDK ハンドラ登録、検索フィルタ |
| `loader.go` | 指定のパターン展開・フォルダ走査・JSONL パース・発言の切り出し・サブエージェント紐付け |
| `cache.go` | SQLite3 キャッシュ（ファイル単位の差分更新） |
| `render.go` | 発言の詳細 HTML 生成 |
| `html.go` | 設定画面の HTML 生成 |
| `types.go` | JSONL レコードと発言の型定義 |
| `loader_test.go` | 発言の切り出し・種別判定・要約のユニットテスト |
| `testdata/` | テスト用の合成トランスクリプト |
| `manifest.json` | プラグインメタ情報 |
| `go.mod` | 独立 Go モジュール |

## テスト

```bash
cd src/plugins/gkill_plugin_claudecode
go test ./...
```

`npm test` は `src/server` のみを対象にするため、このテストは手動で実行する。

## 関連資料

- プラグイン SDK: [`src/server/gkill/plugin/README.md`](../../server/gkill/plugin/README.md)
- プラグインシステム全体: [`src/plugins/README.md`](../README.md)
