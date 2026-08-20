# datas テスト仕様

## 概要

フロントエンドで使用する TypeScript データモデルクラスをテストする（35ファイル）。各モデルのデフォルトコンストラクション、フィールド代入、シリアライゼーションを検証している。

## テストフレームワーク

Vitest

## テストファイル一覧

| ファイル | テスト対象モデル |
|---------|----------------|
| `src/client/__tests__/unit/datas/kmemo.test.ts` | Kmemo（テキストメモ） |
| `src/client/__tests__/unit/datas/mi.test.ts` | Mi（タスク） |
| `src/client/__tests__/unit/datas/kyou.test.ts` | Kyou（基底レコード） |
| `src/client/__tests__/unit/datas/tag.test.ts` | Tag（タグ） |
| `src/client/__tests__/unit/datas/time-is.test.ts` | TimeIs（タイムスタンプ） |
| `src/client/__tests__/unit/datas/ur-log.test.ts` | URLog（ブックマーク） |
| `src/client/__tests__/unit/datas/nlog.test.ts` | Nlog（支出記録） |
| `src/client/__tests__/unit/datas/lantana.test.ts` | Lantana（気分値） |
| `src/client/__tests__/unit/datas/kc.test.ts` | KC（数値記録） |
| `src/client/__tests__/unit/datas/text-data.test.ts` | Text（テキスト注釈） |
| `src/client/__tests__/unit/datas/git-commit-log.test.ts` | GitCommitLog（Gitコミットログ） |
| `src/client/__tests__/unit/datas/gps-log.test.ts` | GPSLog（GPS位置情報） |
| `src/client/__tests__/unit/datas/idf-kyou.test.ts` | IDFKyou（ファイル） |
| `src/client/__tests__/unit/datas/notification-data.test.ts` | Notification（通知） |
| `src/client/__tests__/unit/datas/re-kyou.test.ts` | ReKyou（リポスト） |
| `src/client/__tests__/unit/datas/mi-re-kyou.test.ts` | MiReKyou（リポストタスク: target_id + Miのスケジュール項目、タイトルなし） |
| `src/client/__tests__/unit/datas/info-base.test.ts` | InfoBase（情報ベース） |
| `src/client/__tests__/unit/datas/info-identifier.test.ts` | InfoIdentifier（情報識別子） |
| `src/client/__tests__/unit/datas/meta-info-base.test.ts` | MetaInfoBase（メタ情報ベース） |
| `src/client/__tests__/unit/datas/circle-options.test.ts` | CircleOptions（円オプション） |
| `src/client/__tests__/unit/datas/lat-lng.test.ts` | LatLng（緯度経度） |
| `src/client/__tests__/unit/datas/kftl-template-element-data.test.ts` | KftlTemplateElementData（KFTLテンプレート要素） |
| `src/client/__tests__/unit/datas/share-kyous-info.test.ts` | ShareKyousInfo（共有情報） |
| `src/client/__tests__/unit/datas/dashboard-config.test.ts` | DashboardConfig（ダッシュボード設定: MI検索条件・Dnote検索条件） |
| `src/client/__tests__/unit/datas/saved-find-query-config.test.ts` | SavedFindQueryConfig（保存済み検索条件: parse の不正データ耐性・往復・作業用コピーの参照切り） |
| `src/client/__tests__/unit/datas/plaing-time-is-config.test.ts` | PlaingTimeIsConfig（実行中検索条件: plaing検索のカスタム条件。parse/to_json のラウンドトリップ） |
| `src/client/__tests__/unit/datas/rep-type-map.test.ts` | RepTypeローカライズマップ・ApplicationConfig未定義RepType自動追加時の表示名ローカライズ |

上記に加えて、エンティティ横断のテストが3つある。

| ファイル | テスト内容 |
|---------|-----------|
| `src/client/__tests__/unit/datas/attached-histories.test.ts` | 11エンティティの `load_attached_histories` / `load_attached_datas` |
| `src/client/__tests__/unit/datas/append-not-found-tags.test.ts` | ApplicationConfig のタグ構造に未登録タグを自動追加する処理 |
| `src/client/__tests__/unit/datas/kyou-load-all-request-count.test.ts` | `Kyou.load_all()` が1件あたりに飛ばすリクエスト本数の回帰テスト（`load_attached_histories` の二重呼び出し、`load_attached_timeis` からの `get_application_config` 呼び出しを検出する） |
| `src/client/__tests__/unit/datas/append-not-found-reps.test.ts` | ApplicationConfig の記録保管場所ツリーに未登録の rep を自動追加する処理（タグ版と対） |
| `src/client/__tests__/unit/datas/application-config-clone.test.ts` | ApplicationConfig の複製。設定ダイアログは複製を編集して「適用」で初めて送るので、複製が浅いと**キャンセルが効かなくなる** |
| `src/client/__tests__/unit/datas/kyou-typed-data-dispatch.test.ts` | `data_type` の接頭辞から型別データを振り分ける表。`mirekyou` を `mi` より**先**に見る必要があるので、分岐の書き順ではなく**接頭辞表を長い順に並べる**ことで構造的に保証する |
| `src/client/__tests__/unit/datas/info-base-lazy-abort-controller.test.ts` | `InfoBase` の `AbortController` を遅延生成すること（30万件ぶん先に作ると生成だけでメインスレッドが固まる） |
| `src/client/__tests__/unit/datas/info-base-lazy-attached-arrays.test.ts` | `InfoBase` の付随データ配列を遅延生成すること（同上） |

## テスト内容

### `clone()` のフィールド網羅

データモデルの `clone()` は「全フィールドを1行ずつ手で写す」実装になっており、
フィールドを増やしたときに `clone()` への追記を忘れるのが典型的な壊れ方。
その場合コピー元では値が入っているのにコピー先だけ既定値になり、
画面上はエラーにならず「編集ダイアログで値が消える」といった形で表面化する。

各テストは `__tests__/helpers/clone-parity.ts` の
`expectCloneCopiesAllFields(new Xxx())` 1行でこれを確認する。
ヘルパは全フィールドに型に応じた非既定値を詰めてから `clone()` し、
値が変わっていないフィールドを列挙して落とす。

意図的にコピーされないフィールド（`attached_*`、`is_attached_*_loaded`、
`is_checked_kyou`）はヘルパ側の既定除外リストにある。いずれも
「必要になった時点で読み直す」遅延ロード用の入れ物と、その読み込み済みフラグ。

> 以前は各ファイルに「代表的な4フィールドだけ確認する clone テスト」があったが、
> それでは肝心のフィールド取りこぼしを検出できなかった。
> あわせて `can be instantiated`（`instanceof` の確認）や
> `xxx defaults to empty string`（コンストラクタでの代入の確認）といった
> TypeScript が保証済みのテストも削除している。

### `load_attached_histories` の横断テスト

各モデルの履歴読み込みは
「自分のIDで履歴取得APIを呼ぶ → errors なら attached_histories を触らず返す →
成功したら詰める」という共通の形をしている。エンティティごとに呼ぶAPIと
レスポンスのキーが違い（`get_kmemo` → `kmemo_histories`、
`get_tag_histories_by_tag_id` → `tag_histories` など）、
ここを取り違えると「履歴だけ常に空」という静かな壊れ方をする。

`attached-histories.test.ts` が11エンティティ分をテーブル駆動で確認する。

`load_attached_datas` は abort 由来の例外を握りつぶす。画面遷移や検索のやり直しで
AbortController が発火したときに呼び出し側へ例外を投げないための処理で、
`try { return await ... } catch { ... }` の await が抜けていると catch に入らず素通りする
（async 関数で `return promise` は try/catch に捕まらない）。
この await 漏れは実際に12箇所あり、修正済み。

## 実行方法

```bash
npm run test_client_unit
```
