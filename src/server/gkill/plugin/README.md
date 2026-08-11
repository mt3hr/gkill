# gkill/plugin

gkill プラグインシステムのサーバー側実装。プラグインプロセスのライフサイクル管理と通信制御を担う。

## ディレクトリ構造

```
plugin/
└── sdk/                # プラグイン作者向け Go SDK（8ファイル。うちテスト3）
    ├── types.go        # 公開型定義（Query, Kyou, Config）
    ├── handler.go      # Handler struct（プラグイン作者が実装するインターフェース）
    ├── sdk.go          # Run() — メインループ（stdin/stdout 改行区切りJSONループ）
    ├── config.go       # LoadConfig / SaveConfig / EnsureConfig（config.json 読み書き）
    ├── sdk_test.go     # Run() ループのテスト（TestRunLoop_* 14本）
    └── config_test.go  # EnsureConfig のテスト（4本）
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

        GetGPSLogs: func(ctx context.Context, q sdk.GPSLogQuery, cfg sdk.Config) (sdk.GPSLogPage, error) {
            // 期間内の GPS ログを返す（manifest.json の provides に "gpslog" を書いたときのみ必須）
            // q.Limit を必ず尊重すること。無視するとgkill側の受信バッファ(32MB)を超える
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
| `get_gps_logs` | gkill → plugin | 期間内の GPS ログ取得（`provides` に `gpslog` があるプラグインのみ。ページングあり） |

起動引数:
- `--gkill-plugin-dir <path>` — プラグイン専用ディレクトリ（config.json を保存する場所）
- `--gkill-user-id <id>` — リクエスト元ユーザー ID
- `--gkill-protocol-version <version>` — プロトコルバージョン（現在は `"1"`）

## 型別データ・付随データ

`manifest.json` の `provides` に種別を書くと、`sdk.Kyou` の `Typed` / `Tags` / `Texts` /
`Notifications` が gkill 本体の型別リポジトリに載る。

```go
sdk.Kyou{
    ID: "...", DataType: "kc", RelatedTime: t, UpdateTime: t,
    Tags:  []string{"fitbit"},
    Typed: &sdk.TypedData{KC: &sdk.KC{Title: "歩数", NumValue: "12345"}},
}
```

`Typed` に非nilにしてよいのは高々1つ。型別データは ID も時刻も持たず、親の Kyou からコピーされる。
`provides` を書かないプラグインでは何も登録されない（従来どおりの動作）。

詳細は [`documents/reverse/plugin-system.md`](../../../../documents/reverse/plugin-system.md) の14章。

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

- プラグイン実装例: [`src/plugins/`](../../../plugins/README.md)
- プラグインシステム設計: [`documents/reverse/plugin-system.md`](../../../../documents/reverse/plugin-system.md)
