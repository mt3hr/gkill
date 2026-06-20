# gkill_example

gkill プラグイン SDK のサンプル実装。固定の Kyou を2件返し、ツイート風カードで詳細 HTML を表示する。新しいプラグインを作成する際のテンプレートとして使用する。

## 機能

- 固定の Kyou を2件返す（日付フィルタに対応）
- 設定で表示メッセージを変更可能（`message` キー）
- 詳細ビューにカード形式の HTML を表示
- プラグイン設定フォームを提供（メッセージ編集）

## ビルド

```bash
cd src/plugins/examples/gkill_example
go build -o gkill_example .
# Windows の場合
go build -o gkill_example.exe .
```

## 配置

```
$GKILL_HOME/plugins/{userID}/gkill_example/
├── manifest.json      # このディレクトリの manifest.json をコピー
└── gkill_example      # ビルドしたバイナリを配置（.exe は自動補完）
```

## 設定項目

| キー | 説明 | デフォルト |
|---|---|---|
| `message` | タイムラインに表示するメッセージ | `"Hello from gkill_example plugin!"` |

設定は gkill の設定画面（ApplicationConfig → プラグイン）から変更できる。

## ファイル構成

| ファイル | 内容 |
|---|---|
| `main.go` | SDK を使ったサンプル実装 |
| `manifest.json` | プラグインメタ情報（名前・バージョン・data_type 等） |
| `go.mod` | 独立 Go モジュール |

## 関連資料

- プラグイン SDK: [`src/server/gkill/plugin/README.md`](../../../server/gkill/plugin/README.md)
- プラグインシステム全体: [`src/plugins/README.md`](../../README.md)
