# gkill プラグインシステム仕様

## 1. 概要

プラグインシステムは、外部バイナリをサブプロセスとして起動し、stdio 改行区切り JSON で通信する拡張機構です。ChatGPT の会話履歴や Claude.ai の会話履歴など、gkill 本体が対応していない外部データを Kyou として統合表示できます。

### 主な特徴

| 特徴 | 内容 |
|---|---|
| 実行モデル | 外部バイナリをサブプロセスとして起動し stdin/stdout で通信 |
| 通信プロトコル | 改行区切り JSON（newline-delimited JSON） |
| データ粒度 | プラグインが決める。ChatGPT / Claude.ai は 1 メッセージ = 1 Kyou、Claude Code は 1 ターン（自分の発言と、それに対する一連の応答）= 1 Kyou（`data_type` は `claude_code_turn`） |
| コンテンツ表示 | `GetContentHTML` が返す HTML を iframe (srcdoc) で描画 |
| ダークテーマ | postMessage 経由で親ページからテーマを通知、CSS変数で切り替え |

---

## 2. プラグインディレクトリ構成

```
$GKILL_HOME/plugins/{userID}/{pluginName}/
  manifest.json        — プラグインメタデータ
  config.json          — プラグイン設定（SDK が起動時に自動生成）
  {executable}         — プラグインバイナリ（Linux/macOS）
  {executable}.exe     — プラグインバイナリ（Windows）
```

例:
```
$GKILL_HOME/plugins/admin/gkill_plugin_claudeai/
  manifest.json
  config.json
  gkill_plugin_claudeai        （Linux）
  gkill_plugin_claudeai.exe    （Windows）
```

プラグインが作るキャッシュは、gkill の他の派生キャッシュと同じくキャッシュディレクトリ配下に置く
（「6. SQLite3 キャッシュ」参照）。

---

## 3. manifest.json

プラグインディレクトリ直下に配置するメタデータファイル。

実例（`src/plugins/gkill_plugin_claudeai/manifest.json`）:

```json
{
  "protocol_version": "1",
  "name": "gkill_plugin_claudeai",
  "version": "1.0.0",
  "description": "Claude.ai のチャット履歴をgkillタイムラインに表示する。conversations.json をプラグインフォルダに置いて使用。",
  "data_type": "claude_conversation",
  "rep_name": "Claude.ai",
  "executable": "gkill_plugin_claudeai",
  "min_gkill_version": "1.1.3"
}
```

定義は `src/server/gkill/api/gkill_plugin/plugin_manifest.go` の `PluginManifest` 構造体（8フィールド）。

| フィールド | 説明 |
|---|---|
| `protocol_version` | プロトコルバージョン。現在は `"1"`（`--gkill-protocol-version` で渡される） |
| `name` | プラグインの識別名。**ディレクトリ名と一致させること**（`rep_name` とは別物） |
| `version` | プラグインのバージョン（例: `"1.0.0"`） |
| `description` | プラグインの説明文 |
| `data_type` | このプラグインが生成する Kyou の `data_type` 値。既存の `data_type`（kmemo, kc 等）と衝突しない一意な名前にする |
| `rep_name` | タイムライン上でのリポジトリ表示名。`GetRepName()` が返す値 |
| `executable` | 実行ファイル名（拡張子なし、OS に応じて `.exe` 等を自動付与） |
| `min_gkill_version` | このプラグインが動作する最低 gkill バージョン |

### manifest.json / config.json の自動生成

同梱プラグイン3本（chatgpt / claudeai / claudecode）は manifest.json を `//go:embed` でバイナリに埋め込んでおり、
次のフラグで標準出力に書き出せる。サンプルの `gkill_example` は埋め込みもフラグも持たない。

| フラグ | 出力 |
|---|---|
| `--gkill-print-manifest` | 埋め込み済みの `manifest.json` |
| `--gkill-print-config` | 既定の `config.json` |

また初回起動時に `sdk.EnsureConfig`（`plugin/sdk/config.go:37-54`）が
`Handler.DefaultConfig` の内容で manifest.json の隣に `config.json` を作成する。
**既存ファイルは決して上書きしない**（既に存在する場合・既定値が nil の場合・プラグインディレクトリが空文字の場合は何もしない）。

---

## 4. プラグインプロセスプロトコル

### 起動

`pluginRepositoryImpl.ensureStarted()` がプラグインバイナリをサブプロセスとして起動する。

```go
cmd := exec.CommandContext(context.Background(),
    execPath,
    "--gkill-plugin-dir", p.pluginDir,
    "--gkill-user-id", p.userID,
    "--gkill-protocol-version", p.manifest.ProtocolVersion,
)
```

`context.Background()` を使用するため、HTTP リクエストがキャンセルされてもプロセスは終了しない。
個々の呼び出しの打ち切りについても同様で、詳細は下記「タイムアウトと打ち切り」を参照。

### 通信フォーマット

- **リクエスト**: 親プロセスが `stdin` に JSON を 1 行（`\n` 終端）書き込む
- **レスポンス**: プラグインプロセスが `stdout` に JSON を 1 行（`\n` 終端）書き込む

```
親 → stdin:  {"id":"uuid","command":"find_kyous","query":{...}}\n
親 ← stdout: {"id":"uuid","kyous":[...]}\n
```

`bufio.Scanner` のバッファサイズは**親子で異なる**。

| 側 | 上限 | 実装 |
|---|---|---|
| 親（gkill 本体） | **32MB** | `dao/reps/plugin_repository_impl.go:95` |
| 子（プラグイン SDK） | **1MB** | `plugin/sdk/sdk.go:83-84` |

親側が 32MB なのは大きな HTML レスポンスで `bufio.Scanner: token too long` を防ぐため。
リクエスト JSON は 1MB を超えないため SDK 側は既定のままになっている。

### タイムアウトと打ち切り

打ち切りの契機は2つあり、**プロセスに手を出すかどうかが異なる**。

| 契機 | 意味 | 挙動 |
|---|---|---|
| 呼び出し元のキャンセル | HTTPクライアントの切断・サーバ終了。誰も結果を必要としていない | 待つのをやめて即 return。**プロセスには触らない** |
| 順番待ちの打ち切り | 実行スロットが空かない（既定10秒 `maxPluginQueueWait`）。混んでいるだけ | `ErrPluginBusy` を返す。**プロセスには触らない** |
| gkill自身のデッドライン | 既定30秒（`IsAlive` は5秒）。応答が返らない＝詰まっている | プロセスを**回収**（`Process.Kill()` して `started=false`） |

順序が要点で、**まず順番待ちをして、スロットを取ってから期限を張る**。
期限をスロット取得より前に張ると、行列に並んでいるだけで期限を食い潰し、
正常に応答しているプラグインを期限切れとして殺してしまう
（2026-08-06以前はこうなっていて、一覧の行数ぶんの本文取得が同時に来ると
プロセスの回収と再起動が延々と繰り返され、待ち行列がまったく消化されなくなった）。
呼び出し元が Deadline を持つ場合は、その絶対時刻ではなく**残り時間の長さ**を
実行予算として引き継ぐ。こうすると `IsAlive` の「5秒だけ待つ」という意図を保ったまま、
行列で予算を食い潰すことがなくなる。

`callCommand` は呼び出し元の `context` からキャンセルを切り離し（`context.WithoutCancel`）、
実行予算ぶんの `timeoutCtx` をスロット取得後に作る。Deadline が無い場合は既定30秒。
判定はタイミングではなくエラーの種類で行う（`context.DeadlineExceeded` なら回収、
`context.Canceled` なら放置）。呼び出し元が Deadline 付きだと両者がほぼ同時に Done になり、
`select` がどちらを選ぶか決まらないため。

この切り分けが無いと、フロントが `AbortController` で前のリクエストを打ち切るたび
（ダッシュボードの再取得など）にユーザーのプラグインプロセスが落ちる。

打ち切った呼び出しの応答は遅れて届くので、`sendRequest` はレスポンスIDを
`req.ID` と突き合わせ、一致しないものを読み捨てる。ID が空のものは SDK の
パースエラー応答なので自分宛てとして扱う。

### コマンド一覧

| コマンド（`PluginRequest.Command`） | 説明 |
|---|---|
| `find_kyous` | 検索クエリに合致する Kyou 一覧を返す |
| `get_kyou` | 指定 ID の Kyou 1 件を返す |
| `get_rep_name` | gkill 上のリポジトリ表示名を返す（`sdk/sdk.go:106-108`） |
| `get_content_html` | 指定 ID の Kyou のコンテンツ HTML を返す |
| `get_config_html` | プラグイン設定画面の HTML を返す |
| `post_config` | 設定フォームの送信データを受け取る |
| `ping` | 疎通確認 |
| `close` | プロセス終了要求 |

### 型定義

| パス | 内容 |
|---|---|
| `src/server/gkill/api/gkill_plugin/` | `PluginRequest`, `PluginResponse`, `PluginKyou`, `PluginQuery`, `PluginManifest` |
| `src/server/gkill/plugin/sdk/` | プラグイン側 SDK（`sdk.Run()`, `sdk.Handler`, `sdk.Kyou`, `sdk.Query`） |

### 設定ファイル（config.json）

プラグインの設定は `manifest.json` と同じフォルダ（`--gkill-plugin-dir` で渡される
`$GKILL_HOME/plugins/{userID}/{プラグイン名}/`）の `config.json` に置く。

`sdk.Run()` は起動時に `sdk.EnsureConfig()` を呼び、`Handler.DefaultConfig` が設定されていて
`config.json` が無ければ既定値で生成する。**既存ファイルは決して上書きしない**。
`DefaultConfig` が nil（`gkill_example` など）なら何も生成しない。

**設定はプラグイン Kyou のコンテキストメニュー「プラグイン設定」から編集できる。**
`plugin-html-view.vue` が `plugin-config-dialog.vue` を持ち、メニューから `rep_name` を受けて開く。
ダイアログは `get_plugin_config_html` で設定 HTML を取得して iframe に表示する。

iframe は `sandbox="allow-scripts allow-forms"`（`allow-same-origin` なし）なので、
**iframe 自身は gkill の API を叩けない**。保存は親（ダイアログ）が postMessage で肩代わりする。

| 向き | メッセージ |
|---|---|
| iframe → 親 | `{ gkill_plugin_config: { <キー>: <値>, ... } }` |
| 親 → iframe | `{ gkill_plugin_config_result: { ok: boolean, error?: string } }` |
| 親 → iframe | `{ gkill_theme: "dark" | "light" }`（テーマ通知） |

親は受け取ったフォームを `post_plugin_config` に送り、成功したら設定 HTML を取り直して
読み込み件数などの表示を更新する。生成された `config.json` を手で編集する経路も従来どおり使える。

`gkill-api.ts` の `get_plugin_list()` は今も MCP 専用で、フロントエンドに呼び出し元が無い。

`/api/get_plugin_list` の唯一の実利用者は MCP サーバの `gkill_get_plugin_list` ツール
（`src/mcp/lib/plugin-tools.mjs`）。`/api/get_plugin_content_html` は画面（`plugin-html-view.vue`）と
MCP の本文インライン埋め込み（`inlinePluginContents`）の両方が使う。

---

## 5. 並行制御

### 問題

複数の HTTP リクエストが同時に `stdin`/`stdout` を操作すると、送受信データが混在し「unexpected end of JSON input」などのエラーが発生する。

### 解決策

`pluginRepositoryImpl` が `sync.Mutex` を保持し、`callCommand()` 全体をロックする。

```go
type pluginRepositoryImpl struct {
    // 容量1のチャネル。ミューテックスと違い「待つのをやめられる」
    callSlot  chan struct{}
    userID    string
    pluginDir string
    manifest  gkill_plugin.PluginManifest
    proc      *pluginProcess
}

func (p *pluginRepositoryImpl) callCommand(ctx context.Context, req gkill_plugin.PluginRequest) (*gkill_plugin.PluginResponse, error) {
    // 1. 行列に並ぶ（待ちきれなければ ErrPluginBusy。プロセスには手を出さない）
    release, err := p.acquireCallSlot(ctx, queueWait)
    defer release()
    // 2. スロットを取ってから期限を張る
    // 3. ensureStarted → sendRequest（クラッシュ時のみ1回リトライ）
}
```

`pluginProcess` 構造体（プロセスハンドル側）には mutex を置かない。

stdout の読み取りは**プロセスごとに常駐する1本の goroutine**（リーダー）が担当し、
`bufio.Scanner` に触れるのはリーダーだけにしている（`bufio.Scanner` は並行安全ではない）。
リクエスト側はバッファ1の `respCh` 経由で応答を受け取る。誰も読んでいなくても
リーダーが残らないよう、送信は `retired` チャネルとの `select` で行う。

### 設計判断まとめ

| 設計判断 | 内容 | 理由 |
|---|---|---|
| Mutex の位置 | `pluginRepositoryImpl` struct | プロセス再起動後も同じ mutex を使い続けられる |
| プロセス起動 | `context.Background()` を使用 | HTTP リクエストキャンセルでプロセスが終了するのを防ぐ |
| 呼び出し元のキャンセル | 待つのをやめるだけで、プロセスには触らない | フロントは全リクエストに `AbortController` を張っている。切断でプロセスを落とすと、画面を操作しただけでプラグインが死ぬ |
| 呼び出しタイムアウト | 期限が無ければ既定 30 秒を注入し、超過時は `Process.Kill()` | 応答しないプラグインが gkill 全体を止めるのを防ぐ。回収しないと以降の呼び出しも詰まったままになる |
| レスポンスの突き合わせ | `resp.ID` と `req.ID` を照合し、不一致なら読み捨てる | 打ち切った呼び出しの応答が遅れて届いても、別の記録の中身を返さないようにする |
| stdout の読み取り | プロセスごとに常駐リーダー1本 | `bufio.Scanner` を複数 goroutine で共有しない。打ち切りのたびに goroutine が増えない |
| クラッシュ復旧 | 失敗時に回収 → `ensureStarted()` → 再送信を1回リトライ。ただし**打ち切り時はリトライしない** | プロセスが予期せず終了した場合の自動復旧。打ち切りの再試行で待ち時間が倍増するのを避ける |
| Scanner バッファ | 親 32MB / SDK 1MB | 大きなHTMLレスポンスで `bufio.Scanner: token too long` を防ぐ（親側のみ拡張） |

実装: `src/server/gkill/dao/reps/plugin_repository_impl.go`

---

## 6. SQLite3 キャッシュ（ChatGPT / Claude.ai / Claude Code プラグイン）

ChatGPT・Claude.ai・Claude Code プラグインはデータファイルをパースしてメッセージを SQLite3 にキャッシュする。

### キャッシュ位置

```
$GKILL_HOME/caches/plugin_cache/{userID}/{pluginName}/cache.db
```

thumb_cache / video_cache / git_commit_log_cache と同じく `gkill_options.CacheDir` 配下に置く。
gkill 本体は起動時に環境変数 `GKILL_HOME` を設定し（`common.InitGkillOptions`）、プラグインの
サブプロセスはその環境を引き継ぐため、プラグインは環境変数からキャッシュルートを解決できる。
環境変数が無い場合は `--gkill-plugin-dir`（`$GKILL_HOME/plugins/{userID}/{pluginName}`）から遡って
推定し、それも解けなければプラグインフォルダ直下にフォールバックする
（実装: `src/plugins/*/cache_path.go` の `cacheDBPath`）。

キャッシュは `clear_cache plugin <all|user_id...>` で削除できる（`clear_cache all` にも含まれる）。

### テーブル構成

**スキーマは ChatGPT / Claude.ai と Claude Code で別物**なので分けて示す。

#### ChatGPT / Claude.ai（`gkill_plugin_{chatgpt,claudeai}/cache.go`）

**cache_meta（キャッシュ管理テーブル）**

| カラム | 型 | 説明 |
|---|---|---|
| `key` | TEXT PRIMARY KEY | メタ情報のキー（ソース署名の保存に使う） |
| `value` | TEXT NOT NULL | 値 |

**conv_cache（会話テーブル）**

| カラム | 型 | 説明 |
|---|---|---|
| `conv_id` | TEXT PRIMARY KEY | 会話 ID |
| `title` | TEXT NOT NULL | 会話タイトル |
| `create_time_unix` | INTEGER NOT NULL | 作成時刻（Unix秒） |

**msg_cache（メッセージテーブル）**

| カラム | 型 | 説明 | ChatGPT | Claude.ai |
|---|---|---|---|---|
| `msg_id` | TEXT PRIMARY KEY | メッセージ ID | ○ | ○ |
| `conv_id` | TEXT NOT NULL | 所属会話 ID | ○ | ○ |
| `sender` | TEXT NOT NULL | 送信者（`user`/`assistant` or `human`/`assistant`） | ○ | ○ |
| `text` | TEXT NOT NULL | メッセージテキスト | ○ | ○ |
| `related_time_unix` | INTEGER NOT NULL | 関連時刻（Unix秒） | ○ | ○ |
| `create_time_unix` | INTEGER NOT NULL | 作成時刻（Unix秒） | ○ | ○ |
| `update_time_unix` | INTEGER | 更新時刻（Unix秒） | — | ○ |

インデックス: `idx_msg_conv(conv_id)`, `idx_msg_time(related_time_unix)`

#### Claude Code（`gkill_plugin_claudecode/cache.go`）

`cache_meta` に加えて以下の2テーブルを持つ。`schema_version` によるマイグレーション付き。

**file_cache（取り込み済みファイル）**

| カラム | 型 | 説明 |
|---|---|---|
| `path` | TEXT PRIMARY KEY | ソースファイルパス |
| `mtime_unix` | INTEGER NOT NULL | 最終更新時刻（Unix秒） |
| `size` | INTEGER NOT NULL | ファイルサイズ |
| `kind` | TEXT NOT NULL | ファイル種別 |
| `session_id` | TEXT NOT NULL | 所属セッション ID |

**message_cache（メッセージ）**

| カラム | 型 | 説明 |
|---|---|---|
| `message_id` | TEXT PRIMARY KEY | メッセージ ID |
| `role` | TEXT NOT NULL | `human` / `assistant` 等 |
| `source_path` | TEXT NOT NULL | 取り込み元ファイル |
| `session_id` | TEXT NOT NULL | セッション ID |
| `session_title` | TEXT NOT NULL | セッションタイトル |
| `project` | TEXT NOT NULL | プロジェクト名 |
| `branch` | TEXT NOT NULL | git ブランチ名 |
| `message_text` | TEXT NOT NULL | 表示用テキスト |
| `search_text` | TEXT NOT NULL | 検索用テキスト |
| `body_json` | TEXT NOT NULL | 元データの JSON |
| `related_time_unix` | INTEGER NOT NULL | 関連時刻（Unix秒） |
| `update_time_unix` | INTEGER NOT NULL | 更新時刻（Unix秒） |

インデックス: `idx_msg_time(related_time_unix)`, `idx_msg_session(session_id)`, `idx_msg_src(source_path)`

### キャッシュ無効化

プラグインによって2方式ある。

- **ChatGPT / Claude.ai（全体再構築）** — `cache_meta` に保存したソースの**署名**を現在のソース状態と
  突き合わせ（`needsRebuild`）、変化していれば `rebuild()` が**キャッシュ全体を作り直す**。
- **Claude Code（セッション単位の差分更新）** — `refresh`（`cache.go:209`）が `file_cache` の
  `MtimeUnix` / `Size` を突き合わせて（`:236-243`）変化のあったファイルだけを `dirtySessions` として拾い、
  そのセッションだけ作り直す。変化が無ければ早期リターンする（`:255-257`）。

### 主要メソッド

| プラグイン | メソッド | 説明 |
|---|---|---|
| ChatGPT / Claude.ai | `GetMessages(pluginDir, src)` | 全メッセージ一覧取得（`FindKyous` で使用） |
| ChatGPT / Claude.ai | `GetMsgByID(pluginDir, src, msgID)` | `msg_cache LEFT JOIN conv_cache WHERE msg_id = ?` で1件取得 |
| ChatGPT / Claude.ai | `GetConvForMsg(...)` | メッセージが属する会話を取得 |
| Claude Code | `GetMessages(pluginDir, src)` / `GetMessage(pluginDir, src, messageID)` / `GetStats(...)` | Claude Code は `GetMsgByID` ではなく `GetMessage` |

いずれも第2引数に展開済みソース（`expandedSource`）を取る。

### source_dirs 設定

3プラグインとも `config.json` の `source_dirs` キーで取り込み元フォルダを指定する
（Claude Code の既定は `["~/.claude/projects"]`）。

- `*` `**` `?` `[]` のグロブ、`~`、環境変数を展開する
- **検索のたびに読み直される**（起動時に一度読むだけではないので、gkill の再起動なしに反映される）
- 既定の `config.json` には `_comment` と `_example_source_dirs` の説明キーも書き出される

---

## 7. GetContentHTML — Kyou 1 件分の HTML 生成

1 Kyou = 1 メッセージの粒度で、クリックした件のみを HTML として返す。

### 処理フロー

ChatGPT / Claude.ai:

```
GetContentHTML(kyouID)
  → globalCache.GetMsgByID(pluginDir, src, kyouID)  // 1件のみ取得
  → renderSingleMsgHTML(convTitle, msg)             // 単一メッセージHTML生成
```

Claude Code:

```
GetContentHTML(kyouID)
  → globalCache.GetMessage(pluginDir, src, messageID)
  → renderMessageHTML(t)        // 見つかった場合
  → renderNotFoundHTML()        // 見つからない場合
```

### HTML 構造

- 会話タイトル（`convTitle`）をヘッダに表示
- 送信者ラベル:
  - ChatGPT: `user` → 「あなた」 / その他 → 「ChatGPT」
  - Claude.ai: `human` → 「あなた」 / その他 → 「Claude」
- CSS カスタムプロパティでテーマを切り替え

### CSS 変数（Light / Dark）

| 変数 | ライト値 | ダーク値 | 用途 |
|---|---|---|---|
| `--bg` | `#ffffff` | `#212121` | 背景色 |
| `--text` | `#333333` | `#e0e0e0` | テキスト色 |
| `--msg-user-bg` | `#dbeafe` | `#1a3557` | ユーザメッセージ背景（**ChatGPT**。Claude.ai は `--msg-human-bg`） |
| `--msg-assistant-bg` | `#f3f4f6` | `#2d2d2d` | アシスタントメッセージ背景 |
| `--scrollbar-thumb` | `#2672ed` | `#2672ed` | スクロールバー（gkill primary色） |
| `--scrollbar-track` | — | — | スクロールバーの溝 |
| `--sender-color` | — | — | 送信者ラベルの文字色 |
| `--ts-color` | — | — | タイムスタンプの文字色 |
| `--title-color` | — | — | 会話タイトルの文字色 |

> ユーザ側メッセージの背景変数名はプラグインで異なる（ChatGPT は `--msg-user-bg`、
> Claude.ai は `--msg-human-bg`）。送信者ラベルの語彙（`user` / `human`）に合わせてある。

---

## 8. ダークテーマ通知（postMessage）

iframe は `sandbox="allow-scripts allow-forms"` で動作するため、`allow-same-origin` がなくセッション Cookie にアクセスできない。テーマ情報は postMessage で通知する。

### 通知フロー

```
親（plugin-html-view.vue）→ iframe:
  { gkill_theme: 'dark' | 'light' }

タイミング:
  1. iframe の onload イベント発火時
  2. application_config.use_dark_theme の watch 変更時
```

### iframe 側の処理

```javascript
window.addEventListener('message', function(e) {
    if (e.data && e.data.gkill_theme) {
        document.documentElement.setAttribute('data-theme', e.data.gkill_theme);
        setTimeout(notifySize, 10);  // レイアウト安定後にサイズ再計測
    }
});
```

`[data-theme="dark"]` セレクタで CSS 変数を上書きし、Light / Dark を切り替える。

---

## 9. コンテンツベース高さ調整（postMessage）

iframe はコンテンツの高さを自分では制御できないため、postMessage で親に高さを伝える。

### iframe 側（送信）

```javascript
function notifySize() {
    window.parent.postMessage({
        gkill_iframe_size: {
            width: document.documentElement.scrollWidth,
            height: document.documentElement.scrollHeight
        }
    }, '*');
}
// トリガー: DOMContentLoaded / ResizeObserver / テーマ変更後
```

### 親側（受信）

```typescript
function on_window_message(e: MessageEvent): void {
    // 自分の iframe 以外からのメッセージは無視
    if (!iframe_ref.value || e.source !== iframe_ref.value.contentWindow) return
    if (e.data && e.data.gkill_iframe_size) {
        const h = e.data.gkill_iframe_size.height
        if (typeof h === 'number' && h > 0) iframe_content_height.value = h
    }
}
```

`iframe_content_height` が 0 の間は 80px をフォールバック高さとして使用する。`scrolling="no"` で iframe 自身のスクロールバーを非表示にし、スクロールは親コンポーネントに委譲する。

実装: `src/client/pages/views/plugin-html-view.vue`

---

## 10. PluginKyou コンテキストメニュー

GitCommitLogContextMenu と同じ項目に加えて、プラグイン固有の「プラグイン設定」を持つ。

| メニュー項目 | 表示条件 |
|---|---|
| タグ履歴からタグを追加 | 常時 |
| タグを追加 | 常時 |
| テキストを追加 | 常時 |
| リポスト | 常時 |
| タスク化 | 常時 |
| 通知を追加 | 常時 |
| 内容コピー | 常時 |
| IDをコピー | 常時 |
| プラグイン設定 | 常時（`plugin-config-dialog.vue` を開く） |
| フォルダを開く | `session_is_local` 時のみ |
| ファイルを開く | `session_is_local` 時のみ |

### 実装ファイル

| ファイル | 役割 |
|---|---|
| `src/client/pages/views/plugin-html-view.vue` | メインViewコンポーネント（iframeでHTML表示） |
| `src/client/pages/views/plugin-html-view-props.ts` | `PluginHtmlViewProps`（`KyouViewPropsBase` を拡張） |
| `src/client/classes/use-plugin-html-view.ts` | Viewレベル Composable（コンテキストメニュー表示・crudRelayHandlers） |
| `src/client/pages/views/plugin-html-context-menu.vue` | コンテキストメニューコンポーネント |
| `src/client/pages/views/plugin-html-context-menu-props.ts` | `PluginHtmlContextMenuProps`（`= KyouViewPropsBase`） |
| `src/client/classes/use-plugin-html-context-menu.ts` | コンテキストメニュー Composable |

---

## 11. PWA キャッシュ

### キャッシュ設定

| 項目 | 内容 |
|---|---|
| エンドポイント | `POST /api/get_plugin_content_html` |
| キャッシュストア | `gkill-post-kyou-cache`（他 Kyou データと共通） |
| キャッシュキー | `/cache/api/plugin_content_html/{kyou_id}` |
| 識別子フィールド | `body.kyou_id`（他エンドポイントの `body.id` / `body.target_id` とは命名が異なる） |

### キャッシュ無効化

`delete_gkill_kyou_cache(id)` の `data_types` 配列に `'plugin_content_html'` が含まれており、タグ追加・削除などの操作時に自動でキャッシュがクリアされる。

実装:
- `src/client/serviceWorker.ts` — キャッシュ保存ロジック
- `src/client/classes/delete-gkill-cache.ts` — キャッシュ削除ロジック

---

## 12. 既存プラグイン一覧

| プラグイン名 | Rep名 | DataType | ソースディレクトリ |
|---|---|---|---|
| gkill_plugin_chatgpt | ChatGPT | `chatgpt_conversation` | `src/plugins/gkill_plugin_chatgpt/` |
| gkill_plugin_claudeai | Claude.ai | `claude_conversation` | `src/plugins/gkill_plugin_claudeai/` |
| gkill_plugin_claudecode | ClaudeCode | `claude_code_turn` | `src/plugins/gkill_plugin_claudecode/` |
| gkill_example | （サンプル） | `example_kyou` | `src/plugins/examples/gkill_example/` |

`gkill_example` は固定の Kyou を返すだけのサンプル実装で、`DefaultConfig` を持たない
（＝ `config.json` を生成しない）。上3つは `source_dirs` 設定と SQLite3 キャッシュを持つ。

### ビルド手順

```bash
cd src/plugins/gkill_plugin_chatgpt
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_chatgpt.exe .

cd src/plugins/gkill_plugin_claudeai
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_claudeai.exe .

cd src/plugins/gkill_plugin_claudecode
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_claudecode.exe .
```

デプロイ先: `$GKILL_HOME/plugins/{userID}/{pluginName}/`

---

## 13. MCP からのプラグイン内容取得

AIクライアント（MCP）からもプラグインの記録を読める。プラグインKyouの本文はgkill本体に保存されていない（`convertPluginKyouToKyou` は Texts / Tags / ImageSource を落とし、メタデータだけをKyouにする）ので、AIに本文を届けるには画面と同じく `GetContentHTML` を経由するしかない。

### 提供ツール（read / write / readwrite の3サーバ共通）

| ツール名 | gkill API | 説明 |
|---|---|---|
| `gkill_get_plugin_list` | `/api/get_plugin_list` | プラグイン一覧（name / version / description / data_type / rep_name / is_alive） |

読み取り専用。設定書き換え（`/api/post_plugin_config`）はMCPに公開していない。

本文取得は独立したツールではなく、`gkill_get_kyous` の引数 `include_plugin_content` で行う。かつては1件ずつ取る `gkill_get_plugin_content` ツールがあったが、N件読むのにツール呼び出しがN+1回必要で、そのたびにLLMのターンを消費していたため廃止した。

### 取得導線

```
gkill_get_plugin_list        … どのプラグインが入っているか（data_type / rep_name）を知る
  ↓
gkill_get_kyous              … include_plugin_content:true を付けて検索する。
   (include_plugin_content)     payload.kind = "plugin" のKyouに本文が入って返る
                               （data_type / rep_name / kyou_id / plugin_name / description
                                 + content_status / content_text）
```

インライン化は MCP 層（`src/mcp/lib/plugin-tools.mjs` の `inlinePluginContents`）が担当する。`/api/get_kyous_mcp` のレスポンスから `kind:"plugin"` のペイロードを集め、`rep_name` ごとにグループ化して `/api/get_plugin_content_html` を叩き、HTML→テキスト変換した結果をペイロードに書き戻す。gkill側には一括取得エンドポイントもプラグインプロトコルの一括コマンドも無く、追加していない。

### 並列度と安全弁

**同一プラグインへ並列に投げてはいけない。** プラグインの stdio は1本しかなく呼び出しは直列化されるので、同時発行しても速くならず順番待ちが伸びるだけで、待ちきれなかったぶんは `ErrPluginBusy` になる。したがってインライン取得は **rep内は必ず直列・rep間は並列（既定4）** にしてある。（2026-08-06以前は期限を排他ロックの**前**に張っていたため、ロック待ちで期限を食い潰して `Process.Kill()` が走り、MCPの読み取りが原因でユーザーのプラグインが落ちていた。）

実行中のリクエストを abort しても、いまはプロセスには影響しない（呼び出し元のキャンセルではプロセスを回収しない）。それでも MCP 側は abort せず「新しいリクエストを始めない」だけでデッドラインを実現している。MCPサーバは古い gkill にも接続しうるためである。

そのほかの上限: 1回あたり20件・1件4000文字（`plugin_content_max_text_length` で最大200000）・合計200000文字。あるrepで1件失敗したらそのrepの残りは投げず `content_skipped_reason: "rep_error"` にする（タイムアウトでプロセスが死んでいる可能性が高く、投げ続けてもコールドスタートで待たされるだけのため）。個別の失敗は `content_status` に落ち、`gkill_get_kyous` 全体は失敗しない。

`handle_get_kyous_mcp.go` のペイロード構築は既存 data_type の switch で分岐しており、そのどれにも当たらないKyouは `repositories.PluginReps` から `rep_name` で manifest を引き当てて `PluginPayloadMCPDTO`（`kind: "plugin"`）にする。`kyou_id` をペイロードに載せているのは、`include_id` 指定なしでもコンテンツ取得ができるようにするため（idfペイロードが `rep_name` / `file_name` を常に載せているのと同じ考え方）。

### HTML → テキスト変換

`plugin_content_format` は既定 `text`。プラグインのコンテンツHTMLは `<style>` と `<script>` を含む完結したHTML文書で、バイト数の大半が表示用のボイラープレートになるため、そのまま返すとAIのトークンを浪費するだけになる。MCPサーバ側（`src/mcp/lib/html-text.mjs`）で正規表現ベースの軽量変換をかける:

- `<script>` / `<style>` / コメントは中身ごと破棄
- `<br>` とブロック要素の境界を改行に変換（タグ隣接だけで空行ができないよう、内部マーカー経由で連続をまとめる）
- `<details>` / `<summary>`（ツール実行・thinkingの折りたたみ）の中身は残す
- HTMLエンティティをデコード（`&amp;` は最後に処理し、エスケープ済みマークアップが復活しないようにする）
- `plugin_content_max_text_length`（既定4000文字）を超えたら切り詰め、`content_status: "truncated"` にする

`plugin_content_format: "html"` で生HTMLを `content_html` に、`"both"` で両方返す。長い記録1件の全文が欲しいときは `query.use_ids` + `query.ids` でその1件に絞り、`plugin_content_max_text_length` を上げる。

---

## 関連資料

- [program-spec.md](program-spec.md) — `pluginRepositoryImpl` の実装詳細
- [api-endpoints.md](api-endpoints.md) — `get_plugin_content_html` エンドポイント
- [frontend-architecture.md](frontend-architecture.md) — `plugin-html-view.vue` コンポーネント・PWAキャッシュ
- [sequence-diagrams.md](sequence-diagrams.md) — プラグインコンテンツHTML取得シーケンス
- [`src/mcp/README.md`](../../src/mcp/README.md) — MCPのプラグインツール（`gkill_get_plugin_list`）と本文のインライン埋め込み
- [glossary.md](glossary.md) — PluginKyou, PluginRepository 用語定義
