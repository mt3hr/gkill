# datas テスト仕様

## 概要

フロントエンドで使用する全22種のTypeScriptデータモデルクラスをテストする。各モデルのデフォルトコンストラクション、フィールド代入、シリアライゼーションを検証している。

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
| `src/client/__tests__/unit/datas/info-base.test.ts` | InfoBase（情報ベース） |
| `src/client/__tests__/unit/datas/info-identifier.test.ts` | InfoIdentifier（情報識別子） |
| `src/client/__tests__/unit/datas/meta-info-base.test.ts` | MetaInfoBase（メタ情報ベース） |
| `src/client/__tests__/unit/datas/circle-options.test.ts` | CircleOptions（円オプション） |
| `src/client/__tests__/unit/datas/lat-lng.test.ts` | LatLng（緯度経度） |
| `src/client/__tests__/unit/datas/kftl-template-element-data.test.ts` | KftlTemplateElementData（KFTLテンプレート要素） |
| `src/client/__tests__/unit/datas/share-kyous-info.test.ts` | ShareKyousInfo（共有情報） |

## テスト内容

各テストファイルで以下を検証:
- デフォルトコンストラクタによるインスタンス生成
- 各フィールドへの値代入と取得
- JSON シリアライゼーション / デシリアライゼーション

## 実行方法

```bash
npm run test_client_unit
```
