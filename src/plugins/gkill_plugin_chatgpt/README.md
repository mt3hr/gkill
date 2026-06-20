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
└── conversations.json      # ChatGPT からエクスポートしたデータ
```

## プラグイン情報

| 項目 | 値 |
|---|---|
| `rep_name` | `ChatGPT` |
| `data_type` | `chatgpt_conversation` |
| プロトコルバージョン | `1` |
| 最小 gkill バージョン | `1.1.3` |

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | エントリポイント、SDK ハンドラ登録 |
| `loader.go` | `conversations.json` の読み込み・パース |
| `cache.go` | メモリキャッシュ（再起動まで保持） |
| `html.go` | 会話詳細の HTML 生成 |
| `types.go` | ChatGPT エクスポート形式の型定義 |
| `manifest.json` | プラグインメタ情報 |
| `go.mod` | 独立 Go モジュール |

## 関連資料

- プラグイン SDK: [`src/server/gkill/plugin/README.md`](../../server/gkill/plugin/README.md)
- プラグインシステム全体: [`src/plugins/README.md`](../README.md)
