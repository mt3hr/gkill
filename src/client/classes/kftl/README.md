# kftl - KFTL 行分類器（クライアント側・行ラベル専用）

## 概要

KFTL（gkill 独自テキストフォーマット）の本文を行に分類して、エディタの**行ラベル**
（「メモ」「タスク」「板名」… の表示と、まだ書いていない行の先読み）を作る。
フロントエンドの `/kftl` ページ・各画面のメモ帳ダイアログ・打刻メモ帳で使用される。

**ここは表示のためだけの分類器で、「どの行が正しいか」「何を書くか」は持たない。**
判定と書き込みはサーバの Go 実装（`src/server/gkill/api/kftl/`）だけが持ち、Web は
`/api/parse_kftl_text`（解析のみ）の応答で「おかしな行」をピンクに塗り、未知タグ・未知板名の
確認に使い、`/api/submit_kftl_text` で書く（[ADR-0507](../../../../documents/adr/0507-kftl-single-implementation-on-server.md)）。
2026-09-15 までは TS 側にも解釈と `add_*` API への fan-out があり、Go だけに入った修正が
Web に届かないまま残る事故（`/mood` 単独で気分0が書かれる等）を繰り返していた。

日本語プレフィックス（`。` `ーー` `ーみ` 等、i18n キー経由）に加えて、非日本語ロケール向けの
**ASCII プレフィックス**（`#` `--` `~~` `/mi` `/mood` `/expense` `/num` `/url` `/start` `/end` `/timeis`
`/end?` `/endt` `/endt?` `,` `,,` `?` `??` `!`）も受け付ける。ASCII 定数と判定・除去ヘルパーは
`kftl-prefixes.ts` に集約されており、Go 側 `kftl_factory.go` の `splitter*Ascii` 定数と対応する
（対応表はサーバ側 [README](../../../server/gkill/api/kftl/README.md) を参照）。
行の種別（接頭辞の規則と次の行の決め方）はサーバと同じでなければラベルが嘘になるので、
接頭辞を足す・変えるときは Go 側 `kftl_factory.go` と同じコミットで直す。

## ディレクトリ構造

```
kftl/
├── (ルートファイル 9個)          # 分類器の骨格・接頭辞・日時パース・ラベル型
├── kftl_kmemo/                 # Kmemo 行（1ファイル）
├── kftl_kc/                    # KC 行（3ファイル）
├── kftl_lantana/               # Lantana 行（2ファイル）
├── kftl_mi/                    # Mi 行（7ファイル）
├── kftl_mirekyou/              # MiReKyou 行（8ファイル）
├── kftl_nlog/                  # Nlog 行（6ファイル）
├── kftl_urlog/                 # URLog 行（3ファイル）
├── kftl_timeis/                # TimeIs 行（4ファイル）
│   ├── kftl_timeis_start/      # TimeIs 開始（2ファイル）
│   └── kftl_timeis_end/        # TimeIs 終了（2ファイル）
│       ├── kftl_timeis_end_exist/       # 存在時終了（2ファイル）
│       ├── kftl_timeis_end_tag/         # タグ指定終了（2ファイル）
│       └── kftl_timeis_end_tag_exist/   # タグ指定・存在時終了（2ファイル）
├── kftl_tag/                   # Tag 行（1ファイル）
├── kftl_text/                  # Text 行（3ファイル）
├── kftl_split/                 # Split 行（2ファイル）
├── kftl_related_time/          # RelatedTime 行（1ファイル）
├── kftl_repeat/                # 繰り返し「？？」（5ファイル）
└── kftl_none/                  # None 行（1ファイル）
```

## ルートファイル（9ファイル）

| ファイル | サーバ側対応 | 役割 |
|---------|-------------|------|
| `kftl-statement.ts` | `kftl_statement.go`（`generateKFTLLines`） | 本文を行に分類し、行ラベル（先読み50行を含む）を作るエントリポイント |
| `kftl-statement-line.ts` | `kftl_statement_line.go` | 各行の基底クラス（ラベル名・textarea での折り返し行数） |
| `kftl-statement-line-context.ts` | `kftl_statement_line_context.go` | 行の分類時のコンテキスト（次の行のコンストラクタ・target_id の引き回し） |
| `kftl-statement-line-constructor-factory.ts` | `kftl_factory.go` | 行コンストラクタファクトリ |
| `kftl-prefixes.ts` | `kftl_factory.go`（`splitter*Ascii`） | ASCII プレフィックスの定数と判定・除去ヘルパー。保存マーカー `！`/`!` の判定もここ |
| `line-label-data.ts` | — | 行ラベルデータ（UI 表示用） |
| `text-area-info.ts` | — | テキストエリア情報（エディタ UI 用） |
| `kftl-date-time.ts` | `kftl_related_time_statement_line.go`（`dateFormats`） | KFTL の日時文字列のパース（関連時刻・期限のラベルの「読めない」判定に使う） |
| `kftl-schedule-field-time.ts` | `kftl_related_time_statement_line.go`（`parseScheduleFieldTime`） | Mi / MiReKyou の予定日時欄。行頭の `？`/`?` は例外にする（ラベルを「不正な期限」にするため） |

## データ型別サブディレクトリ

各ディレクトリは「開始行 → 項目行 → 次の行」の連鎖（コンストラクタが次の行のコンストラクタを決める）だけを持つ。

### `kftl_kmemo/`（1ファイル）— テキストメモ

| ファイル | 役割 |
|---------|------|
| `kftl-kmemo-statement-line.ts` | Kmemo 行（どの接頭辞にも当たらない行の受け皿） |

### `kftl_kc/`（3ファイル）— 数値記録

| ファイル | 役割 |
|---------|------|
| `kftl-start-kc-statement-line.ts` | KC 開始行（`ーか` プレフィックス） |
| `kftl-kc-title-statement-line.ts` | KC タイトル行 |
| `kftl-kc-num-value-statement-line.ts` | KC 数値行 |

### `kftl_lantana/`（2ファイル）— 気分値

| ファイル | 役割 |
|---------|------|
| `kftl-start-lantana-statement-line.ts` | Lantana 開始行（`ーら` プレフィックス） |
| `kftl-lantana-mood-statement-line.ts` | Lantana 気分値行 |

### `kftl_mi/`（7ファイル）— タスク

| ファイル | 役割 |
|---------|------|
| `kftl-start-mi-statement-line.ts` | Mi 開始行（`ーみ` プレフィックス） |
| `kftl-mi-block.ts` | ブロックの中の次の行の先読み（タグ行・テキストブロックを挟んでも項目の位置を消費しない） |
| `kftl-mi-title-statement-line.ts` | Mi タイトル行 |
| `kftl-mi-board-name-statement-line.ts` | Mi ボード名行 |
| `kftl-mi-limit-time-statement-line.ts` | Mi 期限行 |
| `kftl-mi-estimate-start-time-statement-line.ts` | Mi 見積開始時刻行 |
| `kftl-mi-estimate-end-time-statement-line.ts` | Mi 見積終了時刻行 |

### `kftl_mirekyou/`（8ファイル）— 既存の記録をタスク化

`～～`（ASCII は `~~`）で開いて同じ記号で閉じるブロック。同じレコードで書いた Kyou を
タスク化するので、対象の id はバケツリレーされてきたものを使う。
Mi と違ってタイトル行を持たず、ブロックの中に書いたタグは MiReKyou 自身に付く（付け先の解決はサーバ）。

| ファイル | 役割 |
|---------|------|
| `kftl-start-mi-re-kyou-statement-line.ts` | MiReKyou 開始行（`～～` プレフィックス） |
| `kftl-mi-re-kyou-board-name-statement-line.ts` | MiReKyou ボード名行 |
| `kftl-mi-re-kyou-estimate-start-time-statement-line.ts` | MiReKyou 見積開始時刻行 |
| `kftl-mi-re-kyou-estimate-end-time-statement-line.ts` | MiReKyou 見積終了時刻行 |
| `kftl-mi-re-kyou-limit-time-statement-line.ts` | MiReKyou 期限行 |
| `kftl-mi-re-kyou-tag-statement-line.ts` | ブロック内のタグ行（前置・後置とも）。次の行の先読みもここ |
| `kftl-mi-re-kyou-none-statement-line.ts` | 項目行を書き終えたあとの受け皿。行ラベルは「**********」で、閉じる行は引き続き `～～` |
| `kftl-end-mi-re-kyou-statement-line.ts` | MiReKyou 終了行（`～～` プレフィックス） |

### `kftl_nlog/`（6ファイル）— 支出記録

`ーん`（ASCII は `/expense`）で開き、店名のあとに（品名, 金額）のペアを繰り返すブロック。
金額の行のあとに書いたタグ行・テキストブロックは「直前の支払い」に付き、ブロック全体で
共有するのは店名と関連時刻（`？`）だけ（付け先の解決はサーバ）。

| ファイル | 役割 |
|---------|------|
| `kftl-start-nlog-statement-line.ts` | Nlog 開始行（`ーん` プレフィックス） |
| `kftl-nlog-block.ts` | ブロックの中に留まる次行の先読み |
| `kftl-nlog-title-statement-line.ts` | Nlog タイトル行。ここで支払いごとの target_id を採番する（ラベルの縞の単位） |
| `kftl-nlog-shop-name-statement-line.ts` | Nlog 店名行 |
| `kftl-nlog-amount-statement-line.ts` | Nlog 金額行 |
| `kftl-nlog-related-time-statement-line.ts` | ブロック内の関連時刻行 |

### `kftl_urlog/`（3ファイル）— ブックマーク

| ファイル | 役割 |
|---------|------|
| `kftl-start-ur-log-statement-line.ts` | URLog 開始行（`ーう` プレフィックス） |
| `kftlur-log-title-statement-line.ts` | URLog タイトル行 |
| `kftlur-log-url-statement-line.ts` | URLog URL 行 |

### `kftl_timeis/`（4ファイル + サブディレクトリ）— タイムスタンプ

TimeIs は最も複雑な KFTL 型で、開始/終了の複数パターンを持つ。

**ルート:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-statement-line.ts` | TimeIs 開始行（`ーち` プレフィックス） |
| `kftl-time-is-title-statement-line.ts` | TimeIs タイトル行 |
| `kftl-time-is-start-time-statement-line.ts` | TimeIs 開始時刻行 |
| `kftl-time-is-end-time-statement-line.ts` | TimeIs 終了時刻行 |

**`kftl_timeis_start/`（2ファイル）— TimeIs 開始専用:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-start-statement-line.ts` | TimeIs 開始専用行（`ーた` プレフィックス） |
| `kftl-time-is-start-title-statement-line.ts` | タイトル行 |

**`kftl_timeis_end/`（2ファイル）— TimeIs 終了:**

| ファイル | 役割 |
|---------|------|
| `kftl-start-time-is-end-statement-line.ts` | TimeIs 終了行（`ーえ` プレフィックス） |
| `kftl-time-is-end-title-statement-line.ts` | タイトル行 |

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

> タグ行とテキスト終了行は、次の行を kmemo か none にして**そこでブロックを打ち切る**。
> ブロックの中で使うときは `KFTLBlockReentryProvider`（`kftl-statement-line.ts`）を渡すと、
> 次の行もブロックの中として解釈される。渡さなければ（既定の `null`）従来どおり。
> テキストは出口を決めるのが終了行なので、開始行から終了行まで持ち回る必要がある。

**`kftl_split/`（2ファイル）:**
- `kftl-split-statement-line.ts` — 区切り行（`、` プレフィックス）
- `kftl-split-and-next-second-statement-line.ts` — 次秒区切り行（`、、` プレフィックス）

**`kftl_related_time/`（1ファイル）:**
- `kftl-related-time-statement-line.ts` — 関連時刻行（`？` プレフィックス）

### `kftl_repeat/`（5ファイル）— 繰り返し「？？」

`？？` の単独行で開いて同じ記号で閉じる4行ブロック。**直前に書いた記録を日付を変えて何度も作る。**
候補日時の計算・展開・既存判定・上限はサーバだけが持つ（設計と却下案は
[ADR-0506](../../../../documents/adr/0506-kftl-repeat-block-expands-into-records.md)）。
ここにあるのは4行のラベル（「毎週金曜」「3回」「不正な条件」）を出すための読み方だけ。

| ファイル | サーバ側対応 | 役割 |
|---------|-------------|------|
| `kftl-repeat-spec.ts` | `kftl_repeat.go`（parse 系のみ） | 条件の語彙・回数/終了日・既存時・起点のパース（ラベル用） |
| `kftl-repeat-block.ts` | `kftl_repeat_lines.go`（`generateRepeatBlockNextConstructor`） | ブロックの中の次の行を決める先読みと行の位置の定数 |
| `kftl-start-repeat-statement-line.ts` | 同（`kftlStartRepeatStatementLine`） | 開始行 |
| `kftl-repeat-field-statement-line.ts` | 同（`kftlRepeatFieldStatementLine`） | 位置で意味が決まる項目行 |
| `kftl-end-repeat-statement-line.ts` | 同（`kftlEndRepeatStatementLine`） | 終了行 |

**`kftl_none/`（1ファイル）:**
- `kftl-none-statement-line.ts` — 認識不能行（スキップ）

## 開発ガイドライン

### サーバ側との対応関係

- クライアント側は**行ラベルの表示だけ**。判定（おかしな行）・確認に使うタグと板名・書き込みはサーバ側 `api/kftl/` パッケージが担当
- 新しいデータ型・接頭辞を追加する場合は、Go 側（解釈と書き込み）と TS 側（行の分類とラベル）の両方に実装が必要。**検証ルールを TS に足さないこと**（ピンクは `/api/parse_kftl_text` の応答で塗る）
- 送信の流れは `src/client/classes/use-kftl-view.ts` の `do_submit`（解析 → 確認 → `submit_kftl_text` → `get_kyou` で引き直して `registered_kyou`）

### 命名規則

- ファイル名: kebab-case（例: `kftl-kmemo-statement-line.ts`）
- ディレクトリ名: snake_case + `kftl_` プレフィックス（例: `kftl_kmemo/`）
- クラス名: PascalCase + `KFTL` プレフィックス（例: `KFTLKmemoStatementLine`）
