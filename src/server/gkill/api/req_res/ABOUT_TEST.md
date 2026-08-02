# req_res テスト仕様

## 概要

API リクエスト / レスポンス構造体の **ワイヤ契約** テスト。
これらの構造体は TypeScript クライアント（`src/client/classes/api/`）と
MCP サーバ（`src/mcp/`）が直接依存している JSON の形そのものなので、
「JSONのフィールド名」と「omitempty の効き方」を固定する。

## テストフレームワーク

Go `testing` パッケージ

## テストファイル

| ファイル | テスト内容 |
|---------|-----------|
| `req_res_test.go` | JSONフィールド名の契約、omitempty、`ShouldIncludeTimeIs` の三値解釈 |

## テスト内容

- **JSONフィールド名** (`TestRequestResponse_JSONFieldNames` / `TestMCPPayloadDTO_JSONFieldNames`)
  クライアントとMCPが参照するフィールド名をテーブルで固定する。
  フィールドを増やす分には落ちないが、既存フィールドのリネーム・削除で落ちる。
  MiReKyou のペイロードキーが `mirekyou`（`mi_re_kyou` ではない）といった
  間違えやすい箇所もここに含めている
- **omitempty** (`TestMCPPayloadDTO_OmitsEmptyOptionalFields`)
  MCPのレスポンスは1リクエストで数百件返るため、空フィールドが載ると
  `max_size_mb` の上限にすぐ達してしまう
- **ネストしたペイロード** (`TestKyouMCPDTO_CarriesPluginPayload`)
  `KyouMCPDTO.Payload` は `any` 型。MCPクライアントは `payload.kind` を見て
  分岐するので、具体的なペイロードがそのままネストして出ることを確認する
- **`ShouldIncludeTimeIs`** — `is_include_timeis` が未指定 / true / false の三値。
  `*bool` にしているのは「未指定なら true」を表すためで、
  省略時にTimeIsが落ちるとMCPクライアント側の記録内容が黙って減る

### 純粋なJSON往復テストを置いていない理由

以前このファイルには「構造体 → JSON → 構造体 で値が保持される」ことだけを見る
往復テストが19本あった。これらの構造体はカスタム Marshaler を持たない素の
struct なので、往復が成り立つことは `encoding/json` の保証であり、
**json タグ名を書き換えても往復は常に成功する**（＝実際に壊れるケースを検出できない）。

そのため、往復ではなく「出力されたJSONのキー名」を見るテストへ置き換えている。

## 実行方法

```bash
cd src/server && go test ./gkill/api/req_res/...
```

または:

```bash
npm run test_server
```
