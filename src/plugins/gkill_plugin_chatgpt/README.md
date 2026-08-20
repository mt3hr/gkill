# gkill_plugin_chatgpt

ChatGPT のチャット履歴を gkill タイムラインに表示するプラグイン。ChatGPT からエクスポートした `conversations.json` を読み込み、会話を Kyou として gkill に統合する。

## セットアップ

### 1. データファイルの準備

ChatGPT の設定ページ（Settings → Data controls → Export data）からチャット履歴をエクスポートし、ZIPを解凍して `conversations.json` を取得する。

### 2. ビルド

```bash
cd src/plugins/gkill_plugin_chatgpt
go build -o gkill_plugin_chatgpt .
# Windows の場合
go build -o gkill_plugin_chatgpt.exe .
```

### 3. 配置

```
$GKILL_HOME/plugins/{userID}/gkill_plugin_chatgpt/
├── manifest.json           # このディレクトリの manifest.json をコピー
├── gkill_plugin_chatgpt    # ビルドしたバイナリ（.exe は自動補完）
├── config.json             # データソース設定（初回起動時に自動生成される）
└── conversations.json      # ChatGPT からエクスポートしたデータ
                            # （config.json で別フォルダを指定するなら不要）
```

キャッシュは gkill のキャッシュディレクトリに作られる。
`gkill_server clear_cache plugin <all|user_id...>` で削除でき、次回起動時に作り直される。

```
$GKILL_HOME/caches/plugin_cache/{userID}/gkill_plugin_chatgpt/cache.db
```

以前のバージョンはプラグインフォルダ直下に `cache.db` を作っていた。移行はしないので、
残っている古い `cache.db` `cache.db-wal` `cache.db-shm` は手で消してよい。

### 4. データソースのフォルダを指定する

プラグインの初回起動時に、`manifest.json` と同じフォルダに `config.json` が
**自動生成される**。これを編集してデータソースを指定する。既存ファイルは上書きされない。

```json
{
  "_comment": "書式の説明（読み飛ばされるので消してよい）",
  "_example_source_dirs": ["~/Kyou/ChatGPTExport", "D:/Dropbox/chatgpt_export/**/conversations*.json"],
  "source_dirs": [
    "~/Kyou/ChatGPTExport",
    "D:/Dropbox/export"
  ]
}
```

- 配列でも文字列（改行区切り）でも書ける
- ワイルドカード `*` `**` `?` `[]` が使える。マッチしたフォルダは再帰的に走査して
  `conversations-000.json` などの分割ファイル（無ければ `conversations.json`） を探し、マッチしたファイルはそのまま読む
- 先頭の `~` と環境変数（`$HOME` など）を展開する
  （Windows サービスとして動かす場合は実行アカウントのホームになるため、絶対パスが確実）
- `source_dirs` を空にすると、プラグインフォルダ直下を見る（従来どおりの配置）
- `_` で始まるキーは書式の説明用。プラグインは読まないので消してよい
- 編集は次の検索から反映される（gkill の再起動は不要）

現在の指定と見つかったファイルは、gkill のプラグイン設定画面で確認できる。

### manifest.json / config.json の出力

`manifest.json` はバイナリに埋め込まれている。配置先で吐かせることもできる。
既定の `config.json` も同様に吐ける（通常は起動時に自動生成されるので不要）。

```bash
./gkill_plugin_chatgpt --gkill-print-manifest > manifest.json
./gkill_plugin_chatgpt --gkill-print-config   > config.json
```

## プラグイン情報

| 項目 | 値 |
|---|---|
| `rep_name` | `ChatGPT` |
| `data_type` | `chatgpt_conversation` |
| プロトコルバージョン | `1` |
| 最小 gkill バージョン | `1.1.7` |

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | エントリポイント、SDK ハンドラ登録 |
| `loader.go` | `conversations.json` の読み込み・パース |
| `cache.go` | SQLite3 キャッシュ（テーブル定義・差分判定） |
| `html.go` | 会話詳細の HTML 生成 |
| `types.go` | ChatGPT エクスポート形式の型定義 |
| `manifest.json` | プラグインメタ情報 |
| `go.mod` | 独立 Go モジュール |

## 関連資料

- プラグイン SDK: [`src/server/gkill/plugin/README.md`](../../server/gkill/plugin/README.md)
- プラグインシステム全体: [`src/plugins/README.md`](../README.md)
