# kftl - KFTL パーサ（クライアント側）

## 概要

KFTL（gkill 独自テキストフォーマット）のクライアント側パーサ・エディタロジック。
フロントエンドの `/kftl` ページ（KFTL エディタ）で使用される。
サーバ側の対応実装: `src/server/gkill/api/kftl/`

## ディレクトリ構造

```
kftl/
├── (ルートファイル 9個)          # コア型定義
├── kftl_kmemo/                 # Kmemo 行（2ファイル）
├── kftl_kc/                    # KC 行（4ファイル）
├── kftl_lantana/               # Lantana 行（3ファイル）
├── kftl_mi/                    # Mi 行（7ファイル）
├── kftl_nlog/                  # Nlog 行（5ファイル）
├── kftl_urlog/                 # URLog 行（4ファイル）
├── kftl_timeis/                # TimeIs 行（5ファイル）
│   ├── kftl_timeis_start/      # TimeIs 開始（3ファイル）
│   └── kftl_timeis_end/        # TimeIs 終了（4ファイル）
│       ├── kftl_timeis_end_exist/       # 存在時終了（2ファイル）
│       ├── kftl_timeis_end_tag/         # タグ指定終了（2ファイル）
│       └── kftl_timeis_end_tag_exist/   # タグ指定・存在時終了（2ファイル）
├── kftl_tag/                   # Tag 行（1ファイル）
├── kftl_text/                  # Text 行（3ファイル）
├── kftl_split/                 # Split 行（2ファイル）
├── kftl_related_time/          # RelatedTime 行（1ファイル）
├── kftl_none/                  # None 行（1ファイル）
└── kftl_prototype/             # プロトタイプ（1ファイル）
```

## ルートファイル（9ファイル）

| ファイル | サーバ側対応 | 役割 |
|---------|-------------|------|
| `kftl-statement.ts` | `kftl_statement.go` | KFTL テキスト全体のパースエントリポイント |
| `kftl-statement-line.ts` | `kftl_statement_line.go` | 各行のインタフェース定義 |
| `kftl-statement-line-context.ts` | `kftl_statement_line_context.go` | 行パース時のコンテキスト |
| `kftl-statement-line-constructor-factory.ts` | `kftl_factory.go` | 行コンストラクタファクトリ |
| `kftl-request.ts` | `kftl_request.go` | リクエストインタフェース |
| `kftl-request-base.ts` | — | リクエスト基底クラス |
| `kftl-request-map.ts` | `kftl_request_map.go` | リクエスト ID マップ |
| `line-label-data.ts` | — | 行ラベルデータ（UI 表示用） |
| `text-area-info.ts` | — | テキストエリア情報（エディタ UI 用） |

## データ型別サブディレクトリ

### `kftl_kmemo/`（2ファイル）— テキストメモ

| ファイル | 役割 |
|---------|------|
| `kftl-kmemo-statement-line.ts` | Kmemo 行の解釈 |
| `kftl-kmemo-request.ts` | Kmemo リクエスト生成 |

### `kftl_kc/`（4ファイル）— 数値記録

| ファイル | 役割 |
|---------|------|
| `kftl-start-kc-statement-line.ts` | KC 開始行（`ーか` プレフィックス） |
| `kftl-kc-title-statement-line.ts` | KC タイトル行 |
| `kftl-kc-num-value-statement-line.ts` | KC 数値行 |
| `kftl-kc-request.ts` | KC リクエスト生成 |

### `kftl_lantana/`（3ファイル）— 気分値

| ファイル | 役割 |
|---------|------|
| `kftl-start-lantana-statement-line.ts` | Lantana 開始行（`ーら` プレフィックス） |
| `kftl-lantana-mood-statement-line.ts` | Lantana 気分値行 |
| `kftl-lantana-request.ts` | Lantana リクエスト生成 |

### `kftl_mi/`（7ファイル）— タスク

| ファイル | 役割 |
|---------|------|
| `kftl-start-mi-statement-line.ts` | Mi 開始行（`ーみ` プレフィックス） |
| `kftl-mi-title-statement-line.ts` | Mi タイトル行 |
| `kftl-mi-board-name-statement-line.ts` | Mi ボード名行 |
| `kftl-mi-limit-time-statement-line.ts` | Mi 期限行 |
| `kftl-mi-estimate-start-time-statement-line.ts` | Mi 見積開始時刻行 |
| `kftl-mi-estimate-end-time-statement-line.ts` | Mi 見積終了時刻行 |
| `kftl-mi-request.ts` | Mi リクエスト生成 |

### `kftl_nlog/`（5ファイル）— 支出記録

| ファイル | 役割 |
|---------|------|
| `kftl-start-nlog-statement-line.ts` | Nlog 開始行（`ーん` プレフィックス） |
| `kftl-nlog-title-statement-line.ts` | Nlog タイトル行 |
| `kftl-nlog-shop-name-statement-line.ts` | Nlog 店名行 |
| `kftl-nlog-amount-statement-line.ts` | Nlog 金額行 |
| `kftl-nlog-request.ts` | Nlog リクエスト生成 |

### `kftl_urlog/`（4ファイル）— ブックマーク

| ファイル | 役割 |
|---------|------|
| `kftl-start-ur-log-statement-line.ts` | URLog 開始行（`ーう` プレフィックス） |
| `kftlur-log-title-statement-line.ts` | URLog タイトル行 |
| `kftlur-log-url-statement-line.ts` | URLog URL 行 |
| `kftlur-log-request.ts` | URLog リクエスト生成 |

### `kftl_timeis/`（5ファイル + サブディレクトリ）— タイムスタンプ

TimeIs は最も複雑な KFTL 型で、開始/終了の複数パターンを持つ。

**ルート:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-statement-line.ts` | TimeIs 開始行（`ーち` プレフィックス） |
| `kftl-time-is-title-statement-line.ts` | TimeIs タイトル行 |
| `kftl-time-is-start-time-statement-line.ts` | TimeIs 開始時刻行 |
| `kftl-time-is-end-time-statement-line.ts` | TimeIs 終了時刻行 |
| `kftl-time-is-request.ts` | TimeIs リクエスト生成 |

**`kftl_timeis_start/`（3ファイル）— TimeIs 開始専用:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-start-statement-line.ts` | TimeIs 開始専用行（`ーた` プレフィックス） |
| `kftl-time-is-start-title-statement-line.ts` | タイトル行 |
| `kftl-time-is-start-request.ts` | リクエスト生成 |

**`kftl_timeis_end/`（4ファイル）— TimeIs 終了:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-end-statement-line.ts` | TimeIs 終了行（`ーえ` プレフィックス） |
| `kftl-time-is-end-title-statement-line.ts` | タイトル行 |
| `kftl-time-is-end-by-tag-request.ts` | タグ指定終了リクエスト |
| `kftl-time-is-end-by-title-request.ts` | タイトル指定終了リクエスト |

**`kftl_timeis_end/kftl_timeis_end_exist/`（2ファイル）— 存在時のみ終了:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-end-if-exist-statement-line.ts` | 存在時終了行（`ーいえ` プレフィックス） |
| `kftl-time-is-end-if-exist-title-statement-line.ts` | タイトル行 |

**`kftl_timeis_end/kftl_timeis_end_tag/`（2ファイル）— タグ指定終了:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-end-by-tag-statement-line.ts` | タグ指定終了行（`ーたえ` プレフィックス） |
| `kftl-time-is-end-by-tag-tag-name-statement-line.ts` | タグ名行 |

**`kftl_timeis_end/kftl_timeis_end_tag_exist/`（2ファイル）— タグ指定・存在時終了:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-end-by-tag-if-exist-statement-line.ts` | タグ指定・存在時終了行（`ーいたえ` プレフィックス） |
| `kftl-time-is-end-by-tag-if-exist-tag-name-statement-line.ts` | タグ名行 |

### メタ行サブディレクトリ

**`kftl_tag/`（1ファイル）:**
- `kftl-tag-statement-line.ts` — タグ行（`。` プレフィックス）

**`kftl_text/`（3ファイル）:**
- `kftl-start-text-statement-line.ts` — テキスト開始行（`ーー` プレフィックス）
- `kftl-text-statement-line.ts` — テキスト本文行
- `kftl-end-text-statement-line.ts` — テキスト終了行

**`kftl_split/`（2ファイル）:**
- `kftl-split-statement-line.ts` — 区切り行（`、` プレフィックス）
- `kftl-split-and-next-second-statement-line.ts` — 次秒区切り行（`、、` プレフィックス）

**`kftl_related_time/`（1ファイル）:**
- `kftl-related-time-statement-line.ts` — 関連時刻行（`？` プレフィックス）

**`kftl_none/`（1ファイル）:**
- `kftl-none-statement-line.ts` — 認識不能行（スキップ）

**`kftl_prototype/`（1ファイル）:**
- `kftl-prototype-request.ts` — プロトタイプリクエスト

## 開発ガイドライン

### サーバ側との対応関係

- クライアント側はエディタ UI とプレビュー表示が主な役割
- 実際のデータ保存はサーバ側 `api/kftl/` パッケージが担当
- 新しいデータ型を追加する場合は両方に実装が必要

### 命名規則

- ファイル名: kebab-case（例: `kftl-kmemo-statement-line.ts`）
- ディレクトリ名: snake_case + `kftl_` プレフィックス（例: `kftl_kmemo/`）
- クラス名: PascalCase + `KFTL` プレフィックス（例: `KFTLKmemoStatementLine`）
