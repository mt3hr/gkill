# gkill プラグインシステム仕様

## 1. 概要

プラグインシステムは、外部バイナリをサブプロセスとして起動し、stdio 改行区切り JSON で通信する拡張機構です。ChatGPT の会話履歴や Claude.ai の会話履歴など、gkill 本体が対応していない外部データを Kyou として統合表示できます。

### 主な特徴

| 特徴 | 内容 |
|---|---|
| 実行モデル | 外部バイナリをサブプロセスとして起動し stdin/stdout で通信 |
| 通信プロトコル | 改行区切り JSON（newline-delimited JSON） |
| データ粒度 | 1 メッセージ = 1 Kyou（例: ChatGPT の 1 発言が 1 件） |
| コンテンツ表示 | `GetContentHTML` が返す HTML を iframe (srcdoc) で描画 |
| ダークテーマ | postMessage 経由で親ページからテーマを通知、CSS変数で切り替え |

---

## 2. プラグインディレクトリ構成

```
$GKILL_HOME/plugins/{userID}/{pluginName}/
  manifest.json        — プラグインメタデータ
  {executable}         — プラグインバイナリ（Linux/macOS）
  {executable}.exe     — プラグインバイナリ（Windows）
  cache.db             — プラグイン独自 SQLite キャッシュ（任意）
```

例:
```
$GKILL_HOME/plugins/admin/gkill_plugin_claudeai/
  manifest.json
  gkill_plugin_claudeai        （Linux）
  gkill_plugin_claudeai.exe    （Windows）
  cache.db
```

---

## 3. manifest.json

プラグインディレクトリ直下に配置するメタデータファイル。

```json
{
  "name": "Claude.ai",
  "rep_name": "Claude.ai",
  "executable": "gkill_plugin_claudeai",
  "protocol_version": "1"
}
```

| フィールド | 説明 |
|---|---|
| `name` | プラグインの識別名 |
| `rep_name` | gkill 上のリポジトリ名。`GetRepName()` が返す値 |
| `executable` | 実行ファイル名（`.exe` を除いたベース名。OS に応じて `.exe` を自動付与） |
| `protocol_version` | プロトコルバージョン（`--gkill-protocol-version` で渡される） |

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

### 通信フォーマット

- **リクエスト**: 親プロセスが `stdin` に JSON を 1 行（`\n` 終端）書き込む
- **レスポンス**: プラグインプロセスが `stdout` に JSON を 1 行（`\n` 終端）書き込む

```
親 → stdin:  {"id":"uuid","command":"find_kyous","query":{...}}\n
親 ← stdout: {"id":"uuid","kyous":[...]}\n
```

`bufio.Scanner` のバッファは **32MB** に設定してあり、大きな HTML レスポンスに対応する。

### コマンド一覧

| コマンド（`PluginRequest.Command`） | 説明 |
|---|---|
| `find_kyous` | 検索クエリに合致する Kyou 一覧を返す |
| `get_kyou` | 指定 ID の Kyou 1 件を返す |
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

---

## 5. 並行制御

### 問題

複数の HTTP リクエストが同時に `stdin`/`stdout` を操作すると、送受信データが混在し「unexpected end of JSON input」などのエラーが発生する。

### 解決策

`pluginRepositoryImpl` が `sync.Mutex` を保持し、`callCommand()` 全体をロックする。

```go
type pluginRepositoryImpl struct {
    mu        sync.Mutex  // すべての操作を直列化するミューテックス
    userID    string
    pluginDir string
    manifest  gkill_plugin.PluginManifest
    proc      *pluginProcess
}

func (p *pluginRepositoryImpl) callCommand(_ context.Context, req gkill_plugin.PluginRequest) (*gkill_plugin.PluginResponse, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    // ensureStarted → sendRequest（エラー時は1回リトライ）
}
```

`pluginProcess` 構造体（プロセスハンドル側）には mutex を置かない。

### 設計判断まとめ

| 設計判断 | 内容 | 理由 |
|---|---|---|
| Mutex の位置 | `pluginRepositoryImpl` struct | プロセス再起動後も同じ mutex を使い続けられる |
| プロセス起動 | `context.Background()` を使用 | HTTP リクエストキャンセルでプロセスが終了するのを防ぐ |
| クラッシュ復旧 | 失敗時に `started=false` → `ensureStarted()` → 再送信を1回リトライ | プロセスが予期せず終了した場合の自動復旧 |
| Scanner バッファ | 32MB | 大きなHTMLレスポンスで `bufio.Scanner: token too long` を防ぐ |

実装: `src/server/gkill/dao/reps/plugin_repository_impl.go`

---

## 6. SQLite3 キャッシュ（ChatGPT / Claude.ai プラグイン）

ChatGPT・Claude.ai プラグインは JSON データファイルをパースしてメッセージを SQLite3 にキャッシュする。

### キャッシュ位置

```
{pluginDir}/cache.db
```

### テーブル構成

**conv_cache（会話テーブル）**

| カラム | 型 | 説明 |
|---|---|---|
| `conv_id` | TEXT PRIMARY KEY | 会話 ID |
| `title` | TEXT | 会話タイトル |
| `file_path` | TEXT | ソースファイルパス |
| `mtime` | INTEGER | ソースファイルの最終更新時刻（Unix秒） |

**msg_cache（メッセージテーブル）**

| カラム | 型 | 説明 | ChatGPT | Claude.ai |
|---|---|---|---|---|
| `msg_id` | TEXT PRIMARY KEY | メッセージ ID | ○ | ○ |
| `conv_id` | TEXT | 所属会話 ID | ○ | ○ |
| `sender` | TEXT | 送信者（`user`/`assistant` or `human`/`assistant`） | ○ | ○ |
| `text` | TEXT | メッセージテキスト | ○ | ○ |
| `related_time_unix` | INTEGER | 関連時刻（Unix秒） | ○ | ○ |
| `create_time_unix` | INTEGER | 作成時刻（Unix秒） | ○ | ○ |
| `update_time_unix` | INTEGER | 更新時刻（Unix秒） | — | ○ |

### キャッシュ無効化

ソースファイルの `mtime` が変化したとき、その会話に属するレコードを再構築する。

### 主要メソッド

| メソッド | 説明 |
|---|---|
| `GetMessages(pluginDir)` | 全メッセージ一覧取得（`FindKyous` で使用） |
| `GetMsgByID(pluginDir, msgID)` | `msg_cache LEFT JOIN conv_cache WHERE msg_id = ?` で1件取得 |

---

## 7. GetContentHTML — 単一メッセージ HTML 生成

1 Kyou = 1 メッセージの粒度で、クリックした件のみを HTML として返す。

### 処理フロー

```
GetContentHTML(kyouID)
  → globalCache.GetMsgByID(pluginDir, kyouID)  // 1件のみ取得
  → renderSingleMsgHTML(convTitle, msg)          // 単一メッセージHTML生成
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
| `--msg-user-bg` | `#dbeafe` | `#1a3557` | ユーザメッセージ背景 |
| `--msg-assistant-bg` | `#f3f4f6` | `#2d2d2d` | アシスタントメッセージ背景 |
| `--scrollbar-thumb` | `#2672ed` | `#2672ed` | スクロールバー（gkill primary色） |

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

GitCommitLogContextMenu と同一の項目を提供する。

| メニュー項目 | 表示条件 |
|---|---|
| タグ履歴からタグを追加 | 常時 |
| タグを追加 | 常時 |
| テキストを追加 | 常時 |
| リポスト | 常時 |
| 通知を追加 | 常時 |
| IDをコピー | 常時 |
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

### ビルド手順

```bash
cd src/plugins/gkill_plugin_chatgpt
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_chatgpt.exe .

cd src/plugins/gkill_plugin_claudeai
GOOS=windows GOARCH=amd64 go build -o gkill_plugin_claudeai.exe .
```

デプロイ先: `$GKILL_HOME/plugins/{userID}/{pluginName}/`

---

## 関連資料

- [program-spec.md](program-spec.md) — `pluginRepositoryImpl` の実装詳細
- [api-endpoints.md](api-endpoints.md) — `get_plugin_content_html` エンドポイント
- [frontend-architecture.md](frontend-architecture.md) — `plugin-html-view.vue` コンポーネント・PWAキャッシュ
- [sequence-diagrams.md](sequence-diagrams.md) — プラグインコンテンツHTML取得シーケンス
- [glossary.md](glossary.md) — PluginKyou, PluginRepository 用語定義
