# kftl テスト仕様

## 概要

KFTL (gkill 独自テキストフォーマット) の TypeScript 側**行分類器**（行ラベル用）と、
メモ帳の送信経路（サーバの `/api/parse_kftl_text` → `/api/submit_kftl_text` → 引き直し）をテストする。
「どの行が正しいか」「何を書くか」の判定はサーバの Go 実装だけが持つ（ADR-0507）ので、
入力の検証・リクエストの中身・展開の正しさは `src/server/gkill/api/kftl/`（[ABOUT_TEST](../../../server/gkill/api/kftl/ABOUT_TEST.md)）と
`gkill_server_api/handle_parse_kftl_text_test.go` / `handle_submit_kftl_text_test.go` が固定する。

## テストフレームワーク

Vitest

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/kftl/kftl-statement.test.ts` | 行の分類（ブロックの行の並び・タグ行やテキストブロックを挟んだときの位置）と行ラベルの先読み（50行上限、受け皿の「**********」） |
| `src/client/__tests__/unit/kftl/kftl-type-detection.test.ts` | ステートメント型の判定（日本語プレフィックス + ASCIIプレフィックス、否定ケース含む） |
| `src/client/__tests__/unit/kftl/kftl-individual-types.test.ts` | ステートメント型ごとの補足テスト（Split と SplitAndNextSecond の排他、Kmemo の catch-all、プレフィックスの一意性、startsWith 型と exact 型の差） |
| `src/client/__tests__/unit/kftl/kftl-date-time.test.ts` | KFTL の日時文字列のパース（欠けた年月日の補完。ラベルの「読めない」判定に使う） |
| `src/client/__tests__/unit/kftl/kftl-schedule-field-time.test.ts` | Mi / MiReKyou の予定日時欄のパース。行頭の `？`/`?` は例外にする（ラベルを「不正な期限」にするため） |
| `src/client/__tests__/unit/kftl/kftl-repeat.test.ts` | 繰り返し「？？」の4行の読み方（条件の語彙・回数/終了日・既存時・起点）。候補日時の計算は Go 側 `kftl_repeat_test.go` |
| `src/client/__tests__/unit/kftl/kftl-submit-emits.test.ts` | 送信の結果を一覧へ伝える経路（解析→送信の順、`registered_kyou` / `updated_kyou`、失敗時は何も上げない、サーバの `invalid_lines` でピンクにして送信しない、`tags` / `mi_board_names` からの確認、打鍵後のデバウンス、タブ、複数ウィンドウ） |

## テスト内容

- **Statement Parsing**: 行単位のステートメント分類（プレフィックス、次の行の決め方）
- **Type Detection**: `kmemo:`, `mi:`, `timeis:` 等のステートメント型判定
- **MiReKyou ブロック**: `～～` で開いて閉じるブロックの行の並び（タイトル行なし・途中で閉じる・空ブロック）、板名の前後どちらにも書けるタグ、波ダッシュ(U+301C)の正規化
- **Individual Types**: ステートメント型は全53種（基底 `KFTLStatementLine` を除く）。このファイルはその全数を個別に回すものではなく、型判定で取り違えが起きやすい箇所（Split / SplitAndNextSecond、Kmemo の catch-all、exact-match プレフィックスの重複、startsWith と exact の違い）を補足的に検証する
- **送信経路**: API はサーバの応答を差し替える小さな偽サーバ（「、」で区切った件数ぶんの id、「。」で始まる行をタグとして返す）。判定の中身はサーバの責務なので、ここではビューの振る舞い（順序・emit・タブ・排他）だけを見る

## 実行方法

```bash
npm run test_client_unit
```
