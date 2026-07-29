# gkill_plugin_claudecode

Claude Code のチャットログ（セッション JSONL）を gkill タイムラインに表示するプラグイン。
**1ターン = 1 Kyou** として扱う。ターンは人間のプロンプトを起点とし、次の人間の入力が来るまでの
Claude の応答・ツール実行をひとまとめにする。

Claude.ai プラグインが「1メッセージ = 1 Kyou」なのに対し、Claude Code は1セッションのツール実行が
数百回に及ぶため、会話の区切り（ターン）を単位にしている。

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
ターンIDが UUID なので重複は自然に統合される。

> **Windows サービスで動かしている場合の注意**
> `GkillServer` は LocalSystem で動くことがある。Go の `os.UserHomeDir()` は Windows では
> `%USERPROFILE%` を見るため、`~` や既定値は実行アカウントのホーム
> （LocalSystem なら `C:\Windows\system32\config\systemprofile`）を指してしまう。
> サービス配下では**絶対パスで書くのが確実**。

**編集は次の検索から反映される**（gkill の再起動は不要）。SDK は `config.json` をプロセス起動時に
一度しか読まないため、このプラグインは毎回 `sdk.LoadConfig` で読み直している。

現在の設定・パターンの展開結果・読み込み状況（対象ファイル数・ターン数・最終スキャン時刻）は、
gkill のプラグイン設定画面（`get_config_html`）で確認できる。何にもマッチしなかった指定もそこに出る。
**設定画面は表示のみで保存はできない** — gkill 側に保存を呼ぶ導線が無いため。`config.json` を直接編集する。

ファイル種別はパスではなく中身で判定する。そのため次のどちらの配置でも読める。

- Claude Code が書く入れ子の配置: `~/.claude/projects/<プロジェクト>/<セッションID>.jsonl`
- 平置きにコピーした配置: 同じフォルダに `<セッションID>.jsonl` と `agent-<ID>.jsonl` が並ぶ形

## 取り込む内容

| ファイル | 用途 |
|---|---|
| `<セッションID>.jsonl` | メインのトランスクリプト。ここからターンを切り出す |
| `agent-<ID>.jsonl` | サブエージェントの会話。親ターンの Agent ツール実行に折りたたんで表示 |
| `agent-<ID>.meta.json` | サブエージェントの種別・説明・`toolUseId`。親との紐付けに使う |
| `history.jsonl` | 対象外（プロンプト履歴のみで会話の文脈がない） |

ターン本文には次を入れる。

- 人間の発言と Claude のテキスト応答（展開表示）
- ツール実行（折りたたみ。ツール名と**入力の1行要約**のみ。実行結果は保持しない）
- thinking（折りたたみ）
- サブエージェントの会話（折りたたみ）
- `task-notification` などシステム発の入力（1行の注記として、直前のターンに吸収）

## 動作の要点

- **ターンID** はプロンプトレコードの `uuid`。同じセッションが複数フォルダにあっても重複しない
- **タグは付けない**。gkill のタグ一覧（`get_all_tag_names`）にはプラグインが返したタグが載らず、
  タグを付けると rykv の既定の絞り込み「no tags」から漏れて何も表示されなくなるため。
  プロジェクト名とブランチ名は詳細HTMLのチップで表示し、ワード検索の対象にも含めている
- **キャッシュ** は `cache.db`（SQLite3）。ファイル単位で mtime とサイズを見て、
  変化のあったセッションだけ作り直す。146MB 規模のソースでも初回のみ数秒、以降は 1 秒未満
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
| `loader.go` | 指定のパターン展開・フォルダ走査・JSONL パース・ターン分割・サブエージェント紐付け |
| `cache.go` | SQLite3 キャッシュ（ファイル単位の差分更新） |
| `render.go` | ターン詳細の HTML 生成 |
| `html.go` | 設定画面の HTML 生成 |
| `types.go` | JSONL レコードとターンの型定義 |
| `loader_test.go` | ターン分割・種別判定・要約のユニットテスト |
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
