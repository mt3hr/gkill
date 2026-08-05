# api テスト仕様

## 概要

`GkillAPI` シングルトンクラスの全メソッドをテストする。全11データ型に対する CRUD 操作、設定管理、共有機能、アップロード、トランザクション、通知、エラーハンドリング、セッション管理、エンドポイントアドレス検証をカバーしている。

## テストフレームワーク

Vitest

## テストファイル

| ファイル | 内容 |
|---------|------|
| `src/client/__tests__/unit/api/gkill-api.test.ts` | GkillAPI の全メソッドテスト、Go とのエンドポイント整合 |
| `src/client/__tests__/unit/api/find-kyou-query.test.ts` | `FindKyouQuery`（rykv / mi の検索条件） |
| `src/client/__tests__/unit/api/hydrate.test.ts` | `hydrate()` / `hydrate_all()` — 生JSONからクラスインスタンスへの詰め替え（`gkill-api.ts` / `datas/kyou.ts` のファイル全体 eslint-disable を解消したヘルパー） |

## テスト内容

- **データ型別 CRUD**: Kmemo, Mi, TimeIs, URLog, Nlog, Lantana, KC, Tag, Text, Notification, ReKyou の追加・更新・削除・取得
- **ZIPブラウズ**: `browse_zip_contents()` メソッドのテスト（BrowseZipContentsRequest/Response の送受信）
- **設定操作**: アプリケーション設定、サーバ設定の読み書き
- **共有機能**: Kyou の共有設定 CRUD
- **アップロード**: ファイルアップロード処理
- **トランザクション**: 複数操作のトランザクション処理
- **通知**: プッシュ通知ターゲットの管理
- **エラーハンドリング**: API エラーレスポンスの処理
- **セッション管理**: ログイン・ログアウト・セッション検証

### Go とのエンドポイント整合

エンドポイントのURLは TypeScript 側（`gkill-api.ts`）と Go 側
（`gkill_server_api_address.go`）で独立に書かれている。片方だけリネームしても
ビルドも type-check も通り、実行して404になって初めて気付く。

`endpoint address parity with Go` が Go のソースからアドレス一覧を読み取り、
クライアント側の `*_address` プロパティと両方向で突き合わせる。
以下は許容リストに入れてあり、いずれも理由をコード内のコメントに書いている。

- **クライアントにしか無い5件** (`/api/update_tag_struct` など) — `gkill-api.ts`
  の中でしか名前が出てこない未使用の定義。呼び出しコードは無いが、
  呼ぶと404になる。定義を消すのはクライアント本体の変更なので既知として通している
- **サーバにしか無い5件** — MCPサーバ / Wear OS から使うもの
  (`/api/submit_kftl_text`, `/api/get_kyous_mcp`, `/api/get_idf_file_path`)、
  保守用 (`/api/update_cache`)、Service Worker 配信 (`/serviceWorker.js`)

> 以前は「`login_address` は `/api/login` である」といった個別テストが3本あったが、
> これは定数を言い換えているだけでズレを検出できないため置き換えた。

### `FindKyouQuery`（検索条件）

rykv / mi の検索条件そのもの。ここが壊れると検索条件が黙って落ち、
画面上は正常に見えたまま結果だけが変わるため気付きにくい。

- **`parse_words_and_not_words`** — `-` 前置の除外語、半角・全角スペースの混在分割、
  単独の `-`、語中のハイフン、`timeis_keywords` 側も同じルールで分解されること
- **`apply_rep_summary_to_detaul` / `rep_to_struct`** — dvnf形式 `type_device_time`
  の3分割、非dvnf名のフォールバック（`device: 'なし'`）、
  type と device の両方がチェックされている Rep だけが選ばれること、
  `ignore_check_rep_rykv` の除外
- **`apply_hide_tags`** — `is_force_hide` を木構造から再帰収集し、呼び直しで累積しないこと
- **`generate_default_query_for_rykv` / `_for_mi`** — mi 側は Rep を絞らず
  カレンダー条件も付けない、といった差分
- **`clone` / `parse_find_kyou_query` のフィールド網羅** — 全57フィールドに
  非既定値を入れた個体を往復させ、落ちたフィールドが無いことを確認する。
  両者のコピー対象が一致することも見る（片方だけに足すと
  「保存はできるのに復元できない」非対称なバグになる）

## テストヘルパー

- `src/client/__tests__/helpers/mock-api.ts` — API モックユーティリティ
- `src/client/__tests__/helpers/factory.ts` — テストデータファクトリ（`makeKmemo`, `makeMi`, `makeApplicationConfig` 等）
- `src/client/__tests__/helpers/clone-parity.ts` — `clone()` のフィールド網羅チェック

## 実行方法

```bash
npm run test_client_unit
```
