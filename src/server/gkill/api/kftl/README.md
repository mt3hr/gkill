# kftl - KFTL パーサ（サーバ側）

## 概要

KFTL（gkill 独自のテキストフォーマット）をパースし、データ追加リクエストを生成・実行するパッケージ。
ユーザが KFTL エディタ（フロントエンドの `/kftl` ページや Wear OS アプリ）で入力したテキストを受け取り、
Kmemo / KC / Lantana / Mi / Nlog / TimeIs / URLog などのデータとしてリポジトリに保存する。

クライアント側の対応実装: `src/client/classes/kftl/`

## 設計思想

### 変換パイプライン

```
KFTL テキスト入力
    ↓
KFTLStatement（テキスト全体を保持）
    ↓  行分割・プレフィックス判定
KFTLStatementLine（各行の解釈）
    ↓  行グループ化・リクエスト生成
KFTLRequest（データ追加リクエスト）
    ↓  実行
Repository への保存
```

### kftlFactory パターン

- `kftlFactory` は行ごとの状態（前の行がメタ情報かどうか）を管理する
- TypeScript 側の `KFTLStatementLineConstructorFactory` に対応
- Go 側では **グローバル状態を避けるため、Statement ごとにインスタンスを生成** する設計

### 行プレフィックスによる型判定

KFTL テキストの各行は、先頭の文字列（プレフィックス）でデータ型が決定される:

| プレフィックス | データ型 | 説明 |
|---|---|---|
| `。` | Tag | タグ付け |
| `ーー` | Text | テキスト開始 |
| `？` | RelatedTime | 関連時刻 |
| `、` | Split | 区切り（次ステートメントへ） |
| `、、` | SplitNextSecond | 区切り（次秒へ） |
| `ーか` | KC | 数値記録 |
| `ーみ` | Mi | タスク |
| `ーら` | Lantana | 気分値（0-10） |
| `ーん` | Nlog | 支出記録 |
| `ーた` | TimeIs Start | タイムスタンプ開始 |
| `ーえ` | TimeIs End | タイムスタンプ終了 |
| `ーち` | TimeIs | タイムスタンプ（開始+終了） |
| `ーいえ` | TimeIs End If Exist | 存在時のみ終了 |
| `ーたえ` | TimeIs End By Tag | タグ指定終了 |
| `ーいたえ` | TimeIs End By Tag If Exist | タグ指定・存在時のみ終了 |
| `ーう` | URLog | ブックマーク |
| `！` | Save | 保存実行 |
| (上記以外) | Kmemo | テキストメモ（デフォルト） |

## ファイル一覧（22ファイル）

### コア構造

| ファイル | 役割 |
|---------|------|
| `kftl_factory.go` | `kftlFactory` — 行コンストラクタファクトリ。プレフィックス定数定義。各データ型の `generateXxxConstructor()` メソッドを提供 |
| `kftl_statement.go` | `KFTLStatement` — KFTL テキスト全体のパースエントリポイント。`GenerateAndExecuteRequests()` でパース→リクエスト生成→実行を一括処理 |
| `kftl_statement_line.go` | `KFTLStatementLine` インタフェース — 各行が実装すべきメソッド定義。`StatementLineConstructorFunc` 型定義 |
| `kftl_statement_line_context.go` | `KFTLStatementLineContext` — 行パース時のコンテキスト（BaseTime, AddSecond, UserID, Device 等） |

### リクエスト生成

| ファイル | 役割 |
|---------|------|
| `kftl_request.go` | `KFTLRequest` インタフェース — リクエストの共通メソッド定義（`Execute()`, `GetID()` 等） |
| `kftl_prototype_request.go` | `KFTLPrototypeRequest` — リクエストのプロトタイプ実装。新規 Kyou 追加時の共通ロジック |
| `kftl_request_map.go` | `KFTLRequestMap` — リクエストの ID マップ管理。同一 ID のリクエストを集約 |

### データ型別パーサ

| ファイル | 対応データ型 | 説明 |
|---------|-------------|------|
| `kftl_kmemo.go` | Kmemo | テキストメモ行の解釈・リクエスト生成 |
| `kftl_kc.go` | KC | 数値記録行の解釈・リクエスト生成 |
| `kftl_lantana.go` | Lantana | 気分値行の解釈・リクエスト生成 |
| `kftl_mi.go` | Mi | タスク行の解釈・リクエスト生成 |
| `kftl_nlog.go` | Nlog | 支出記録行の解釈・リクエスト生成 |
| `kftl_timeis.go` | TimeIs | タイムスタンプ行の解釈・リクエスト生成（Start/End/EndIfExist/EndByTag/EndByTagIfExist） |
| `kftl_urlog.go` | URLog | ブックマーク行の解釈・リクエスト生成 |

### ステートメント行（メタ情報系）

| ファイル | 役割 |
|---------|------|
| `kftl_tag_statement_line.go` | Tag 行（`。` プレフィックス） |
| `kftl_text_statement_lines.go` | Text 行（`ーー` プレフィックス） |
| `kftl_related_time_statement_line.go` | RelatedTime 行（`？` プレフィックス） |
| `kftl_split_statement_lines.go` | Split / SplitNextSecond 行（`、` / `、、` プレフィックス） |
| `kftl_none_statement_line.go` | None 行 — 認識できないプレフィックスの行（スキップ） |

### テスト（3ファイル）

| ファイル | 説明 |
|---------|------|
| `kftl_factory_test.go` | ファクトリのプレフィックス判定テスト |
| `kftl_request_map_test.go` | リクエストマップの集約テスト |
| `kftl_statement_test.go` | KFTL テキスト全体のパース・実行テスト |

## 開発ガイドライン

### 新しいデータ型を追加する場合

1. `kftl_factory.go` に新しいプレフィックス定数を追加
2. 新しい `kftl_xxx.go` ファイルを作成し、`KFTLStatementLine` インタフェースを実装
3. `kftl_factory.go` の `generateDefaultConstructor` に新しいプレフィックスの分岐を追加
4. TypeScript 側 `src/client/classes/kftl/` にも対応する実装を追加

### 命名規則

- ファイル名: `kftl_` プレフィックス + snake_case
- 構造体名: `KFTL` プレフィックス + PascalCase（例: `KFTLKmemoStatementLine`）
- コンストラクタ: `newKFTLXxxStatementLine(lineText, ctx)` パターン

### TimeIs の更新方式

TimeIs の「終了」は既存レコードの更新ではなく、**同一 ID で新しいレコードを追加**（append-only）する方式。
`AddTimeIsInfo` に同一 ID + 新しい end_time を渡すことで「更新」を表現する。
