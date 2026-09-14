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
| `～～` | MiReKyou | 既存の記録をタスク化（開始・終了とも同じ記号） |
| `？` | RelatedTime | 関連時刻 |
| `？？` | Repeat | 繰り返し（開始・終了とも同じ記号。直前の記録を日付を変えて何度も作る） |
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

日本語プレフィックスと並んで、非日本語ロケール向けの **ASCII プレフィックス**も受け付ける
（`kftl_factory.go` の `splitter*Ascii` 定数。TypeScript 側は `classes/kftl/kftl-prefixes.ts` の定数と対応）:

| ASCII | 対応する日本語 | データ型 |
|---|---|---|
| `#` | `。` | Tag |
| `--` | `ーー` | Text |
| `~~` | `～～` | MiReKyou |
| `?` | `？` | RelatedTime |
| `??` | `？？` | Repeat |
| `,` | `、` | Split |
| `,,` | `、、` | SplitNextSecond |
| `/num` | `ーか` | KC |
| `/mi` | `ーみ` | Mi |
| `/mood` | `ーら` | Lantana |
| `/expense` | `ーん` | Nlog |
| `/start` | `ーた` | TimeIs Start |
| `/end` | `ーえ` | TimeIs End |
| `/timeis` | `ーち` | TimeIs |
| `/end?` | `ーいえ` | TimeIs End If Exist |
| `/endt` | `ーたえ` | TimeIs End By Tag |
| `/endt?` | `ーいたえ` | TimeIs End By Tag If Exist |
| `/url` | `ーう` | URLog |
| `!` | `！` | Save |

## ファイル一覧（32ファイル）

### コア構造

| ファイル | 役割 |
|---------|------|
| `kftl_factory.go` | `kftlFactory` — 行コンストラクタファクトリ。プレフィックス定数定義。各データ型の `generateXxxConstructor()` メソッドを提供 |
| `kftl_statement.go` | `KFTLStatement` — KFTL テキスト全体のパースエントリポイント。`prepareRequests()`（行の解釈→全行の適用→繰り返しの展開。書かない）を、`GenerateAndExecuteRequests()`（temp rep へ実行→`CommitTx`）と `Analyze()`（`/api/parse_kftl_text`。行別エラー・タグ・板名・件数だけ返す）が共有する（ADR-0507）。各リクエストは一時リポジトリに積み、末尾で `CommitTx`（1つの SQLite トランザクション）で確定する。失敗したら `DiscardTx` して何も残さない |
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
| `kftl_mirekyou.go` | MiReKyou | 既存の記録をタスク化する行の解釈・リクエスト生成（`～～` で開いて閉じるブロック。タイトル行は無く、ブロック内のタグは MiReKyou 自身に付く） |
| `kftl_nlog.go` | Nlog | 支出記録行の解釈・リクエスト生成（`ーん` で開き、店名のあとに（品名, 金額）のペアを繰り返すブロック。**支払い1組ごとに1つのリクエスト**になり RequestID をそのまま Nlog の ID にするので、金額の行のあとに書いたタグ・テキストはその支払いに付く。ブロック全体で共有するのは店名と関連時刻だけ。`ーん` より前のタグ・テキストはエラー） |
| `kftl_timeis.go` | TimeIs | タイムスタンプ行の解釈・リクエスト生成（Start/End/EndIfExist/EndByTag/EndByTagIfExist） |
| `kftl_urlog.go` | URLog | ブックマーク行の解釈・リクエスト生成 |

### ステートメント行（メタ情報系）

| ファイル | 役割 |
|---------|------|
| `kftl_tag_statement_line.go` | Tag 行（`。` プレフィックス）。`resume` を渡すとブロックの中に留まる |
| `kftl_text_statement_lines.go` | Text 行（`ーー` プレフィックス）。出口を決めるのは終了行なので `resume` を開始行から終了行まで持ち回る |
| `kftl_related_time_statement_line.go` | RelatedTime 行（`？` プレフィックス） |
| `kftl_split_statement_lines.go` | Split / SplitNextSecond 行（`、` / `、、` プレフィックス） |
| `kftl_none_statement_line.go` | None 行 — どのプレフィックスにも一致しない行。空行なら無視、**非空なら入力エラー**。次の行の決め方は `generateDefaultConstructor` へ委譲するので、既知のプレフィックスの行はここへ来ない |

### 繰り返し（`？？`）

`？？` の単独行で開いて同じ記号で閉じる4行ブロック。**直前に書いた記録を日付を変えて何度も作る。**
展開（複製の生成）は行の解釈ではなく **`GenerateAndExecuteRequests` の実行ループ直前**でやる
（クライアント側は本文が変わるたびに全行を解釈し直すので、そこで複製すると打鍵1回あたり最大1000件になる）。
設計と却下案は [ADR-0506](../../../../../documents/adr/0506-kftl-repeat-block-expands-into-records.md)。

| ファイル | 役割 |
|---------|------|
| `kftl_repeat.go` | 条件の語彙（曜日・毎日・N週おき・毎月N日・第N曜日・最終曜日）、回数/終了日、既存時、起点のパースと候補日時の計算 |
| `kftl_repeat_lines.go` | 行クラス6種、ブロックの先読み、`expandRepeats`（同じ spec を共有するリクエストを1グループとして複製） |
| `kftl_repeat_duplicate.go` | 型別の「同じ記録があるか」の判定。**型をまたいだ抜けが問題になるのでここへ集約**（1つ実装し忘れるとその型だけ既定の冪等性が黙って消える） |

### テスト（9ファイル）

| ファイル | 説明 |
|---------|------|
| `kftl_factory_test.go` | ファクトリのプレフィックス判定テスト |
| `kftl_request_map_test.go` | リクエストマップの集約テスト |
| `kftl_statement_test.go` | KFTL テキスト全体のパース・実行テスト |
| `kftl_analyze_test.go` | `Analyze`（書かない入口）が `GenerateAndExecuteRequests` と同じ行エラー集合を返すこと、タグ・板名・件数の列挙、空白だけの値の行の拒否、`/end` 系の対象検索に設定の playing 条件（語・タグ・非表示タグ）を写すこと |
| `kftl_mirekyou_test.go` | MiReKyou ブロックの行の並び・タグの帰属・対象の解決テスト |
| `kftl_nlog_test.go` | 支出ブロックの支払いごとのタグ・テキストの帰属、ブロック全体に効く関連時刻、ブロック前のメタ情報行の拒否 |
| `kftl_date_time_test.go` | 関連時刻・打刻時刻の書式と、欠けた年月日の補完 |
| `kftl_schedule_field_time_test.go` | Mi / MiReKyou の予定日時欄。関連時刻の接頭辞「？」を入力エラーにする（[ADR-0505](../../../../../documents/adr/0505-schedule-time-field-rejects-related-time-prefix.md)） |
| `kftl_repeat_test.go` | 繰り返しブロックの語彙・候補日時（クライアント側と対の表）・行の並び・展開・繰り返せない型・既存スキップ。全8型の複製で日時欄が年ごと正しくずれること（打刻・気分値・数値・ブックマーク・支出のタグ用関連時刻を含む） |

## 開発ガイドライン

### 新しいデータ型を追加する場合

1. `kftl_factory.go` に新しいプレフィックス定数を追加
2. 新しい `kftl_xxx.go` ファイルを作成し、`KFTLStatementLine` インタフェースを実装
3. `kftl_factory.go` の `generateDefaultConstructor` に新しいプレフィックスの分岐を追加
4. TypeScript 側 `src/client/classes/kftl/` にも行の分類（行ラベル用）を追加。検証・書き込みは足さない（判定はここだけ。ADR-0507）

### 命名規則

- ファイル名: `kftl_` プレフィックス + snake_case
- 構造体名: `KFTL` プレフィックス + PascalCase（例: `KFTLKmemoStatementLine`）
- コンストラクタ: `newKFTLXxxStatementLine(lineText, ctx)` パターン

### TimeIs の更新方式

TimeIs の「終了」は既存レコードの更新ではなく、**同一 ID で新しいレコードを追加**（append-only）する方式。
`AddTimeIsInfo` に同一 ID + 新しい end_time を渡すことで「更新」を表現する。
