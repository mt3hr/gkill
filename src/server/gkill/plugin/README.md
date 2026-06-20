# gkill/plugin

gkill プラグインシステムのサーバー側実装。プラグインプロセスのライフサイクル管理と通信制御を担う。

## ディレクトリ構造

```
plugin/
└── sdk/           # プラグイン作者向け Go SDK（4ファイル）
    ├── types.go   # 公開型定義（Query, Kyou, Config）
    ├── handler.go # Handler struct（プラグイン作者が実装するインターフェース）
    ├── sdk.go     # Run() — メインループ（stdin/stdout 改行区切りJSONループ）
    └── config.go  # LoadConfig / SaveConfig（config.json 読み書き）
```

## プラグイン SDK の使い方

プラグイン作者は `sdk.Run(sdk.Handler{...})` を呼び出すだけでよい。

```go
import sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"

func main() {
    sdk.Run(sdk.Handler{
        RepName: "MyPlugin",  // manifest.json の rep_name と一致させること

        FindKyous: func(ctx context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
            // 外部データソースから Kyou を取得して返す（必須）
        },

        GetContentHTML: func(ctx context.Context, kyouID string, cfg sdk.Config) (string, error) {
            // Kyou 詳細ビューの HTML を返す（省略可、デフォルト実装あり）
        },

        GetConfigHTML: func(ctx context.Context, cfg sdk.Config) (string, error) {
            // プラグイン設定フォームの HTML を返す（省略可）
        },

        PostConfig: func(ctx context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
            // フォームデータを受けて設定を更新する（省略可、デフォルトで JSON 保存）
        },
    })
}
```

## 通信プロトコル

gkill サーバーとプラグインプロセスは **stdin/stdout 改行区切り JSON** で通信する。

| コマンド | 方向 | 説明 |
|---|---|---|
| `ping` | gkill → plugin | 死活確認 |
| `close` | gkill → plugin | プロセス終了 |
| `get_rep_name` | gkill → plugin | Rep 表示名取得 |
| `find_kyous` | gkill → plugin | Kyou 検索（クエリ付き） |
| `get_kyou` | gkill → plugin | 特定 Kyou 取得（ID 指定） |
| `get_content_html` | gkill → plugin | 詳細 HTML 取得（kyou_id 指定） |
| `get_config_html` | gkill → plugin | 設定フォーム HTML 取得 |
| `post_config` | gkill → plugin | 設定フォームデータ保存 |

起動引数:
- `--gkill-plugin-dir <path>` — プラグイン専用ディレクトリ（config.json を保存する場所）
- `--gkill-user-id <id>` — リクエスト元ユーザー ID
- `--gkill-protocol-version <version>` — プロトコルバージョン（現在は `"1"`）

## プラグインの配置場所

```
$GKILL_HOME/
└── plugins/
    └── {userID}/
        └── {plugin_name}/
            ├── manifest.json     # プラグインメタ情報
            ├── {executable}      # ビルド済みバイナリ（OS/アーキテクチャ別）
            └── config.json       # 設定ファイル（自動生成）
```

## 関連資料

- プラグイン実装例: [`src/plugins/`](../../../../plugins/README.md)
- プラグインシステム設計: [`documents/reverse/plugin-system.md`](../../../../../documents/reverse/plugin-system.md)
