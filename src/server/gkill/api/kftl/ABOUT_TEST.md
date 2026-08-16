# kftl テスト仕様

## 概要

Go バックエンドの KFTL パーサパッケージのテスト（117テスト）。KFTL テキストの解析、ステートメント処理、リクエストマップ構築をカバーする。日本語プレフィックスと ASCII プレフィックスの両方をテストする。Mi時間フィールドのASCII `?` 対応、ASCII 内容パース（タグ `#` 除去・`,` 区切り・関連時刻/TimeIs時刻の `?` 除去）、Nlog タイトル/金額数不一致処理、Lantana 気分値範囲バリデーションのテストも含む。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル一覧

| ファイル | テスト内容 |
|---------|-----------|
| `kftl_factory_test.go` | KftlFactory のインスタンス生成とKFTLテキスト全体の解析・リクエスト生成 |
| `kftl_statement_test.go` | ステートメント単位の解析ロジック（日本語プレフィックス + ASCIIプレフィックス + ASCII内容パース + バリデーション、計80テスト） |
| `kftl_request_map_test.go` | リクエストマップの構築と各データ型へのマッピング |
| `kftl_mirekyou_test.go` | MiReKyou ブロック（`～～` / `~~`）の行の並び、タグの帰属、対象の解決、書き込み用rep未設定時のガード（計20テスト） |

## テスト内容

- **KftlFactory**: `GenerateAndExecuteRequests` による KFTL テキスト全体の処理フロー、コンテキスト管理、時刻計算（`nowFromCtx`）
- **Statement（日本語プレフィックス）**: 行ごとのステートメント解析、型プレフィックス判定（。！？、ーー等）、メタ情報抽出、全データ型のリクエストマップ生成
- **Statement（ASCIIプレフィックス）**: ASCII セーブ文字（!）、タグ（#）、区切り（,）、次秒区切り（,,）、テキストブロック（--）、関連時刻（?）、Mi（/mi）、Lantana（/mood）、Nlog（/expense）、KC（/num）、URLog（/url）、TimeIs Start（/start）、TimeIs End（/end）、TimeIs（/timeis）、TimeIs End If Exist（/end?）、TimeIs End By Tag（/endt）、TimeIs End By Tag If Exist（/endt?）、日本語+ASCII混在入力
- **Request Map**: 解析結果からの API リクエストマップ構築、各データ型（Kmemo, Mi, TimeIs 等）へのマッピング
- **Mi ASCII `?` 対応**: ASCII `?` による limitTime / estimateStartTime / estimateEndTime の設定（3テスト）
- **ASCII 内容パース**: タグ行の `#` プレフィックス除去と `、`/`,` 両区切り対応（交差ケース・回帰含む）、関連時刻行の `?` 除去と日時パース、`/endt` タグ名行の `,` 区切り、TimeIs 開始/終了時刻行の `？`/`?` 除去（7テスト）
- **Nlog 不一致警告**: タイトル数と金額数の不一致時にエラーなくmin(titles,amounts)件生成（1テスト）
- **Lantana 範囲チェック**: 気分値 0-10 の境界値テスト（0/5/10は正常、11/-1はエラー）（5テスト）
- **MiReKyou ブロック**: `～～` で開いて閉じるブロックの行の並び（タイトル行なし・途中で閉じる・空ブロック）、板名の前後どちらにも書けるタグ、波ダッシュ(U+301C)の正規化、対象 Kyou の id 解決、対象が無いときと書き込み用 rep が無いときのガード（20テスト）

## 実行方法

```bash
cd src/server && go test ./gkill/api/kftl/...
```

または:

```bash
npm run test_server
```
